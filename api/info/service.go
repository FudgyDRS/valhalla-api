package infoHandler

import (
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/FudgyDRS/valhalla-api/pkg/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	// "github.com/sirupsen/logrus"
)

func VersionRequest(r *http.Request, parameters ...interface{}) (interface{}, error) {
	return utils.VersionResponse{
		Version: Version,
	}, nil
}

func GetGenesisBalances(r *http.Request) (GetGenesisBalancesResponse, error) {
	// Parse the request parameters into the params struct
	params, err := parseGenesisParams(r)
	if err != nil {
		return GetGenesisBalancesResponse{}, utils.ErrInternal(err.Error())
	}

	// Log the parsed params (optional, for debugging purposes)
	LogGenesisParams(params)

	client, err := GetClientForChain(params.ChainId)
	if err != nil {
		return GetGenesisBalancesResponse{}, err
	}
	multicallAddress, err := getMulticallAddress(params.ChainId)
	if err != nil {
		return GetGenesisBalancesResponse{}, err
	}
	calls := createMulticallParams(params)
	// logrus.Info("Generated multicall parameters:", calls)

	results, err := MulticallView(client, multicallAddress, calls)
	if err != nil {
		return GetGenesisBalancesResponse{}, utils.ErrInternal(fmt.Errorf("multicall view failed: %v", err).Error())
	}

	responseData, err := handleMulticallResponse(results, params)
	if err != nil {
		return GetGenesisBalancesResponse{}, utils.ErrInternal(fmt.Errorf("failed to parse multicall response: %v", err).Error())
	}
	logrus.Info("Generated multicall responseData:", responseData)

	return GetGenesisBalancesResponse{
		Pools: responseData,
	}, nil
}

// type GetGenesisPairParams struct {
// 	ChainId        string `query:"chain-id"`
// 	GenesisAddress string `query:"genesis"`
// 	PairAddress    string `query:"pair"`
// 	BaseAddress    string `query:"base"`
// 	QuoteAddress   string `query:"quote"`
// 	UserAddress    string `query:"user" optional:"true"`
// }

//http://localhost:8080/api/info?query=get-genesis-pair&chain-id=146&genesis=0x23Ee13d49e78811d063722D9228547a7dF73E42E&user=0x04301b0c3bC192C28DD3CAF345C4aE6E979EC040&pid=18&pair=0xAC60849b0456baD97E75E8f84C245Bd9C2Fc9766&quote=0xb1e25689D55734FD3ffFc939c4C3Eb52DFf8A794

func GetGenesisPairBalance(r *http.Request) (GetGenesisPairResponse, error) {
	params, err := parseGenesisPairParams(r)
	if err != nil {
		return GetGenesisPairResponse{}, utils.ErrInternal(err.Error())
	}

	LogGenesisPairParams(params)

	client, err := GetClientForChain(params.ChainId)
	if err != nil {
		return GetGenesisPairResponse{}, err
	}
	multicallAddress, err := getMulticallAddress(params.ChainId)
	if err != nil {
		return GetGenesisPairResponse{}, err
	}
	calls := createMulticallPairParams(params)

	results, err := MulticallView(client, multicallAddress, calls)
	if err != nil {
		return GetGenesisPairResponse{}, utils.ErrInternal(fmt.Errorf("multicall view failed: %v", err).Error())
	}

	responseData, err := handleMulticallPairResponse(results, params)
	if err != nil {
		return GetGenesisPairResponse{}, utils.ErrInternal(fmt.Errorf("failed to parse multicall response: %v", err).Error())
	}
	logrus.Info("Generated multicall responseData:", responseData)

	return responseData, nil
}

func handleMulticallResponse(results []MulticallResult, params *GetGenesisBalancesParams) ([]GetGenesisBalanceResponse, error) {
	var responses []GetGenesisBalanceResponse

	logrus.Info("results: ", results)

	parsedErc20ABI, _ := abi.JSON(strings.NewReader(`
	[{
		"type": "function",
		"name": "name",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "symbol",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "decimals",
		"inputs": [],
		"outputs": [{"name":"","type": "uint8","internalType":"uint8"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "totalSupply",
		"inputs": [],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "balanceOf",
		"inputs": [{"name":"account","type": "address","internalType":"address"}],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	}]`,
	))
	parsedGenesisABI, _ := abi.JSON(strings.NewReader(contractAbiGenesis))

	var resultIndex = 0
	for _, pool := range params.Pools {
		// Create a new response for each pool
		response := GetGenesisBalanceResponse{
			Token:          pool.Address, // Will fill it from the ABI unpack
			PoolId:         pool.PoolId,  // Will fill it from the ABI unpack
			GenesisBalance: "null",       // Default value, will change if valid
			UserBalance:    "null",       // Default value, will change if valid
			UserStake:      "null",       // Default value, will change if valid
			UserReward:     "null",       // Default value, will change if valid
		}

		genesisBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack balanceOf for genesis: %v", err)
		}
		response.GenesisBalance = genesisBalance[0].(*big.Int).String()
		resultIndex += 1

		if params.UserAddress != "0x0000000000000000000000000000000000000000" {
			userBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack balanceOf for user: %v", err)
			}
			response.UserBalance = userBalance[0].(*big.Int).String() // Convert to string
			resultIndex += 1
		}

		// Parse userInfo result (skip if user address is address(0))
		if params.UserAddress != "0x0000000000000000000000000000000000000000" {
			userInfoData, err := parsedGenesisABI.Unpack("userInfo", results[resultIndex].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack userInfo: %v", err)
			}

			response.UserStake = userInfoData[0].(*big.Int).String() // User stake
			// response.UserReward = userInfoData[1].(*big.Int).String() // Reward debt
			resultIndex += 1
		}

		if params.UserAddress != "0x0000000000000000000000000000000000000000" {
			userInfoData, err := parsedGenesisABI.Unpack("pendingVAL", results[resultIndex].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack userInfo: %v", err)
			}

			response.UserReward = userInfoData[0].(*big.Int).String()
			resultIndex += 1
		}
		logrus.Info(response)

		responses = append(responses, response)
	}

	return responses, nil
}

func createMulticallParams(params *GetGenesisBalancesParams) []Calls {
	var calls []Calls
	parsedErc20ABI, _ := abi.JSON(strings.NewReader(`
	[{
		"type": "function",
		"name": "name",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "symbol",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "decimals",
		"inputs": [],
		"outputs": [{"name":"","type": "uint8","internalType":"uint8"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "totalSupply",
		"inputs": [],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "balanceOf",
		"inputs": [{"name":"account","type": "address","internalType":"address"}],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	}]`,
	))
	parsedGenesisABI, _ := abi.JSON(strings.NewReader(contractAbiGenesis))

	// Helper function to append calls, skipping user-related ones when user address is address(0)
	addCall := func(contractAddress common.Address, abi abi.ABI, method string, params interface{}) {
		calls = append(calls, Calls{
			contractAddress: contractAddress,
			abi:             abi,
			method:          method,
			params:          params,
		})
	}

	// Create calls for each pool dynamically
	for _, pool := range params.Pools {
		// Pool-related calls (balanceOf)
		poolAddress := common.HexToAddress(pool.Address)
		genesisAddress := common.HexToAddress(params.GenesisAddress)
		userAddress := common.HexToAddress(params.UserAddress)
		poolId := new(big.Int)
		_, success := poolId.SetString(pool.PoolId, 10) // Base 10 for decimal numbers
		if !success {
			return nil
		}

		addCall(poolAddress, parsedErc20ABI, "balanceOf", genesisAddress)

		if params.UserAddress != "0x0000000000000000000000000000000000000000" {
			addCall(poolAddress, parsedErc20ABI, "balanceOf", userAddress)
		}

		if params.UserAddress != "0x0000000000000000000000000000000000000000" {
			addCall(genesisAddress, parsedGenesisABI, "userInfo", []interface{}{poolId, userAddress})
		}

		if params.UserAddress != "0x0000000000000000000000000000000000000000" {
			addCall(genesisAddress, parsedGenesisABI, "pendingVAL", []interface{}{poolId, userAddress})
		}
	}

	return calls
}

func processMulticallAndParse(client *ethclient.Client, multicallAddress common.Address, params *GetGenesisBalancesParams) (*GetGenesisBalancesResponse, error) {
	// Generate the multicall parameters based on the input
	calls := createMulticallParams(params)

	// Execute the multicall
	results, err := MulticallView(client, multicallAddress, calls)
	if err != nil {
		return nil, fmt.Errorf("multicall view failed: %v", err)
	}

	logrus.Info("just before handleMulticallResponse")
	// Parse the results
	responses, err := handleMulticallResponse(results, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multicall response: %v", err)
	}

	// Return the structured response
	return &GetGenesisBalancesResponse{
		Pools: responses,
	}, nil
}

func handleMulticallPairResponse(results []MulticallResult, params *GetGenesisPairParams) (GetGenesisPairResponse, error) {
	// var response GetGenesisBalanceResponse

	logrus.Info("results: ", results)

	parsedErc20ABI, _ := abi.JSON(strings.NewReader(`
	[{
		"type": "function",
		"name": "name",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "symbol",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "decimals",
		"inputs": [],
		"outputs": [{"name":"","type": "uint8","internalType":"uint8"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "totalSupply",
		"inputs": [],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "balanceOf",
		"inputs": [{"name":"account","type": "address","internalType":"address"}],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	}]`,
	))
	parsedGenesisABI, _ := abi.JSON(strings.NewReader(contractAbiGenesis))

	var resultIndex = 0
	response := GetGenesisPairResponse{
		PairAddress:      params.PairAddress,
		PoolId:           params.PoolId,
		BaseBalance:      "null",
		QuoteBalance:     "null",
		GenesisBalance:   "null",
		UserBalance:      "null", // Default value, will change if valid
		UserStake:        "null", // Default value, will change if valid
		UserReward:       "null", // Default value, will change if valid
		UserBaseBalance:  "null",
		UserQuoteBalance: "null",
	}

	baseBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
	if err != nil {
		return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack balanceOf for baseBalance: %v", err)
	}
	response.BaseBalance = baseBalance[0].(*big.Int).String()
	resultIndex += 1

	quoteBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
	if err != nil {
		return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack balanceOf for quoteBalance: %v", err)
	}
	response.QuoteBalance = quoteBalance[0].(*big.Int).String()
	resultIndex += 1

	genesisBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
	if err != nil {
		return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack balanceOf for genesisBalance: %v", err)
	}
	response.GenesisBalance = genesisBalance[0].(*big.Int).String()
	resultIndex += 1

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		userBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
		if err != nil {
			return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack balanceOf for userBalance: %v", err)
		}
		response.UserBalance = userBalance[0].(*big.Int).String() // Convert to string
		resultIndex += 1
	}

	// Parse userInfo result (skip if user address is address(0))
	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		userInfoData, err := parsedGenesisABI.Unpack("userInfo", results[resultIndex].ReturnData)
		if err != nil {
			return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack userInfo: %v", err)
		}

		response.UserStake = userInfoData[0].(*big.Int).String() // User stake
		// response.UserReward = userInfoData[1].(*big.Int).String() // Reward debt
		resultIndex += 1
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		userInfoData, err := parsedGenesisABI.Unpack("pendingVAL", results[resultIndex].ReturnData)
		if err != nil {
			return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack userInfo: %v", err)
		}
		response.UserReward = userInfoData[0].(*big.Int).String() // Reward awaiting
		resultIndex += 1
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		userBaseBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
		if err != nil {
			return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack balanceOf for userBalance: %v", err)
		}
		response.UserBaseBalance = userBaseBalance[0].(*big.Int).String() // Convert to string
		resultIndex += 1
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		userQuoteBalance, err := parsedErc20ABI.Unpack("balanceOf", results[resultIndex].ReturnData)
		if err != nil {
			return GetGenesisPairResponse{}, fmt.Errorf("failed to unpack balanceOf for userBalance: %v", err)
		}
		response.UserQuoteBalance = userQuoteBalance[0].(*big.Int).String() // Convert to string
		resultIndex += 1
	}

	logrus.Info(response)

	return response, nil
}

func createMulticallPairParams(params *GetGenesisPairParams) []Calls {
	var calls []Calls
	parsedErc20ABI, _ := abi.JSON(strings.NewReader(`
	[{
		"type": "function",
		"name": "name",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "symbol",
		"inputs": [],
		"outputs": [{"name":"","type":"string","internalType":"string"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "decimals",
		"inputs": [],
		"outputs": [{"name":"","type": "uint8","internalType":"uint8"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "totalSupply",
		"inputs": [],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	},
	{
		"type": "function",
		"name": "balanceOf",
		"inputs": [{"name":"account","type": "address","internalType":"address"}],
		"outputs": [{"name":"","type": "uint256","internalType":"uint256"}],
		"stateMutability":"view"
	}]`,
	))
	parsedGenesisABI, _ := abi.JSON(strings.NewReader(contractAbiGenesis))

	addCall := func(contractAddress common.Address, abi abi.ABI, method string, params interface{}) {
		calls = append(calls, Calls{
			contractAddress: contractAddress,
			abi:             abi,
			method:          method,
			params:          params,
		})
	}

	genesisAddress := common.HexToAddress(params.GenesisAddress)
	pairAddress := common.HexToAddress(params.PairAddress)
	baseAddress := common.HexToAddress(params.BaseAddress)
	quoteAddress := common.HexToAddress(params.QuoteAddress)
	userAddress := common.HexToAddress(params.UserAddress)
	poolId := new(big.Int)
	_, success := poolId.SetString(params.PoolId, 10) // Base 10 for decimal numbers
	if !success {
		return nil
	}

	addCall(baseAddress, parsedErc20ABI, "balanceOf", pairAddress)
	addCall(quoteAddress, parsedErc20ABI, "balanceOf", pairAddress)
	addCall(pairAddress, parsedErc20ABI, "balanceOf", genesisAddress)

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(pairAddress, parsedErc20ABI, "balanceOf", userAddress)
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(genesisAddress, parsedGenesisABI, "userInfo", []interface{}{poolId, userAddress})
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(genesisAddress, parsedGenesisABI, "pendingVAL", []interface{}{poolId, userAddress})
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(baseAddress, parsedErc20ABI, "balanceOf", userAddress)
	}

	if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(quoteAddress, parsedErc20ABI, "balanceOf", userAddress)
	}

	return calls
}

func processMulticallPairAndParse(client *ethclient.Client, multicallAddress common.Address, params *GetGenesisPairParams) (*GetGenesisPairResponse, error) {
	// Generate the multicall parameters based on the input
	calls := createMulticallPairParams(params)

	// Execute the multicall
	results, err := MulticallView(client, multicallAddress, calls)
	if err != nil {
		return nil, fmt.Errorf("multicall view failed: %v", err)
	}

	logrus.Info("just before handleMulticallResponse")
	// Parse the results
	responses, err := handleMulticallPairResponse(results, params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multicall response: %v", err)
	}

	// Return the structured response
	return &responses, nil
}
