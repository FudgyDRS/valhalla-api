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

// type GetGenesisBalanceResponse struct {
// 	Token          string `json:"token"`
// 	GenesisBalance string `json:"genesis-balance"`
// 	UserBalance    string `json:"user-balance"`
// 	UserStake      string `json:"user-stake"`
// 	UserReward     string `json:"user-reward"`
// }

// type GetGenesisBalancesResponse struct {
// 	Pools []GetGenesisBalanceResponse `json:"pools"`
// }

// func GetGenesisBalances(r *http.Request) (GetGenesisBalancesResponse, error) {
// 	params, err := parseGenesisParams(r)
// 	if err != nil {
// 		return GetGenesisBalancesResponse{}, utils.ErrInternal(err.Error())
// 	}

// 	LogGenesisParams(params)

// 	// // var err error
// 	// // response := &GetGenesisBalancesResponse{}
// 	var calls []Calls
// 	// var results []MulticallResult
// 	calls = []Calls{
// 		{contractAddress: params.Pools[0].Address, abi: parsedErc20ABI, method: "balanceOf", params: params.GenesisAddress},
// 		{contractAddress: params.Pools[0].Address, abi: parsedErc20ABI, method: "balanceOf", params: params.UserAddress},                          // skip if user address = address(0)
// 		{contractAddress: params.GenesisAddress, abi: parsedGenesisABI, method: "userInfo", params: {params.UserAddress, params.Pools[0].PoolId}}, // skip if user address = address(0)
// 		{contractAddress: params.Pools[1].Address, abi: parsedErc20ABI, method: "balanceOf", params: params.GenesisAddress},
// 		{contractAddress: params.Pools[1].Address, abi: parsedErc20ABI, method: "balanceOf", params: params.UserAddress},                          // skip if user address = address(0)
// 		{contractAddress: params.GenesisAddress, abi: parsedGenesisABI, method: "userInfo", params: {params.UserAddress, params.Pools[1].PoolId}}, // skip if user address = address(0)
// 	}

// 	logrus.Info(calls)

// 	return GetGenesisBalancesResponse{}, nil
// }

func GetGenesisBalances(r *http.Request) (GetGenesisBalancesResponse, error) {
	// Parse the request parameters into the params struct
	params, err := parseGenesisParams(r)
	if err != nil {
		return GetGenesisBalancesResponse{}, utils.ErrInternal(err.Error())
	}

	// Log the parsed params (optional, for debugging purposes)
	LogGenesisParams(params)

	chainInfo, err := GetChainInfo(params.ChainId)
	if err != nil {
		return GetGenesisBalancesResponse{}, err
	}
	client, err := DialClient(chainInfo.RPC)
	if err != nil {
		err_ := fmt.Errorf("dial client %v failed: %v", chainInfo.RPC, err.Error())
		logrus.Error(err_)
		return GetGenesisBalancesResponse{}, err_
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

	logrus.Info("I got here")

	responseData, err := handleMulticallResponse(results, params)
	if err != nil {
		return GetGenesisBalancesResponse{}, utils.ErrInternal(fmt.Errorf("failed to parse multicall response: %v", err).Error())
	}
	logrus.Info("Generated multicall responseData:", responseData)

	// return GetGenesisBalancesResponse{
	// 	Pools: responseData,
	// }, nil
	return GetGenesisBalancesResponse{}, nil
}

//	type GetGenesisBalanceResponse struct {
//		Token          string `json:"token"`
//		GenesisBalance string `json:"genesis-balance"`
//		UserBalance    string `json:"user-balance"`
//		UserStake      string `json:"user-stake"`
//		UserReward     string `json:"user-reward"`
//	}
func handleMulticallResponse(results []MulticallResult, params *GetGenesisBalancesParams) ([]GetGenesisBalanceResponse, error) {
	var responses []GetGenesisBalanceResponse

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
	parsedGenesisABI, _ := abi.JSON(strings.NewReader(contractAbiMulticall))

	for poolIndex, _ := range params.Pools {
		// Create a new response for each pool
		response := GetGenesisBalanceResponse{
			Token:          "",     // Will fill it from the ABI unpack
			GenesisBalance: "null", // Default value, will change if valid
			UserBalance:    "null", // Default value, will change if valid
			UserStake:      "null", // Default value, will change if valid
			UserReward:     "null", // Default value, will change if valid
		}

		// Parse balanceOf results
		if len(results) > poolIndex*2 {
			// The first balanceOf result is for the Genesis balance
			genesisBalance, err := parsedErc20ABI.Unpack("balanceOf", results[poolIndex*2].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack balanceOf for genesis: %v", err)
			}
			response.GenesisBalance = genesisBalance[0].(*big.Int).String() // Convert to string
		}

		// The second balanceOf result is for the user balance (if user address is not address(0))
		if len(results) > poolIndex*2+1 && params.UserAddress != "0x0000000000000000000000000000000000000000" {
			userBalance, err := parsedErc20ABI.Unpack("balanceOf", results[poolIndex*2+1].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack balanceOf for user: %v", err)
			}
			response.UserBalance = userBalance[0].(*big.Int).String() // Convert to string
		}

		// Parse userInfo result (skip if user address is address(0))
		if len(results) > (poolIndex+1)*2 && params.UserAddress != "0x0000000000000000000000000000000000000000" {
			userInfoData, err := parsedGenesisABI.Unpack("userInfo", results[(poolIndex+1)*2].ReturnData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack userInfo: %v", err)
			}

			userInfo := userInfoData[0].(struct {
				Amount     *big.Int
				RewardDebt *big.Int
			})

			response.UserStake = userInfo.Amount.String()      // User stake
			response.UserReward = userInfo.RewardDebt.String() // Reward debt
		}

		// Add the processed response for the current pool
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

		// addCall(poolAddress, parsedErc20ABI, "balanceOf", genesisAddress)
		// if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(poolAddress, parsedErc20ABI, "balanceOf", userAddress)
		// }

		// callData, err := GetCallBytes(parsedGenesisABI, "userInfo", userAddress, poolId)
		// if err != nil {
		// 	return nil
		// }
		// User info for each pool (userInfo method)
		// if params.UserAddress != "0x0000000000000000000000000000000000000000" {
		addCall(genesisAddress, parsedGenesisABI, "userInfo", []interface{}{poolId, userAddress})
		// }
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
