package infoHandler

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"

	"github.com/FudgyDRS/valhalla-api/pkg/utils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

func DialClient(jsonrpc string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(jsonrpc)
	if err != nil {
		err_ := fmt.Errorf("client connection failed: %v", err)
		logrus.Error(err_.Error())
		return nil, err_
	}
	return client, nil
}

func ViewFunction(client *ethclient.Client, contractAddress common.Address, parsedABI abi.ABI, methodName string, args ...interface{}) ([]byte, error) {
	data, err := parsedABI.Pack(methodName, args...)
	if err != nil {
		return nil, err
	}

	callMsg := ethereum.CallMsg{To: &contractAddress, Data: data}
	result, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetCallBytes(parsedABI abi.ABI, methodName string, args ...interface{}) ([]byte, error) {
	isArgsEmpty := func(args []interface{}) bool {
		if len(args) == 0 {
			return true
		}

		for _, arg := range args {
			if arg != nil {
				return false
			}
		}
		return true
	}
	method, ok := parsedABI.Methods[methodName]
	if !ok {
		logrus.Errorf("Method %s not found in ABI", methodName)
	}
	logrus.Infof("Method inputs: %#v", method.Inputs)

	var data []byte
	var err error
	if !isArgsEmpty(args) {
		logrus.Info(methodName)
		logrus.Info(args)
		if len(args) == 1 {
			if inner, ok := args[0].([]interface{}); ok {
				args = inner // unwrap it
			}
		}
		for i, arg := range args {
			logrus.Infof("arg[%d] type: %T, value: %#v", i, arg, arg)
		}

		data, err = parsedABI.Pack(methodName, args...)
	} else {
		for i, arg := range args {
			logrus.Infof("arg[%d] type: %T, value: %#v", i, arg, arg)
		}

		data, err = parsedABI.Pack(methodName)
	}

	return data, err
}

func createCall(parsedABI abi.ABI, contractAddress common.Address, methodName string, params ...interface{}) (Call, error) {
	callData, err := GetCallBytes(parsedABI, methodName, params...)
	if err != nil {
		return Call{}, fmt.Errorf("bytes for call %v failed: %v", methodName, err.Error())
	}
	fmt.Printf("\ninternal calldata: \n%v\n", callData)

	return Call{
		Target:   contractAddress,
		CallData: callData,
	}, nil
}

func ExtCodeSize(client *ethclient.Client, address common.Address) ([]byte, int, error) {
	ctx := context.Background()
	code, err := client.CodeAt(ctx, address, nil) // nil block number for the latest state
	if err != nil {
		return nil, 0, fmt.Errorf("geth client failed to get extcodesize: %v", err)
	}
	return code, len(code), nil
}

func GetStorageAt(client *ethclient.Client, address common.Address, slot int64) ([]byte, error) {
	slotHash := common.BigToHash(common.Big1)
	if slot != 0 {
		slotHash = common.BigToHash(big.NewInt(int64(slot)))
	}

	storage, err := client.StorageAt(context.Background(), address, slotHash, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %+v", err.Error())
	}

	return storage, nil
}

func ConstructCallData(methodName string, params []utils.Parameter) ([]byte, error) {
	// Create method signature
	methodSig := createMethodSignature(methodName, params)

	// Calculate function selector (first 4 bytes of the hash of the method signature)
	selector := crypto.Keccak256([]byte(methodSig))[:4]

	// Parse and pack parameters
	packedParams, err := packParameters(params)
	if err != nil {
		return nil, fmt.Errorf("failed to pack parameters: %v", err)
	}

	// Combine selector with packed parameters
	callData := append(selector, packedParams...)

	return callData, nil
}

func createMethodSignature(methodName string, params []utils.Parameter) string {
	var types []string
	for _, param := range params {
		types = append(types, param.Type)
	}
	return fmt.Sprintf("%s(%s)", methodName, strings.Join(types, ","))
}

func packParameters(params []utils.Parameter) ([]byte, error) {
	if len(params) == 0 {
		logrus.Error("param count 0")
		return []byte{}, nil
	}

	var arguments abi.Arguments
	var values []interface{}

	for _, param := range params {
		abiType, err := abi.NewType(param.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("invalid type %s: %v", param.Type, err)
		}
		arguments = append(arguments, abi.Argument{Type: abiType})

		value, err := parseParameterValue(param.Type, param.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameter value: %v", err)
		}
		values = append(values, value)
	}

	logrus.Error(values)
	packed, err := arguments.Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack values: %v", err)
	}

	return packed, nil
}

func parseParameterValue(paramType, paramValue string) (interface{}, error) {
	switch paramType {
	case "uint256":
		val, ok := new(big.Int).SetString(paramValue, 10)
		if !ok {
			return nil, fmt.Errorf("invalid uint256: %s", paramValue)
		}
		return val, nil

	case "address":
		return common.HexToAddress(paramValue), nil

	case "bytes":
		if !strings.HasPrefix(paramValue, "0x") {
			return nil, fmt.Errorf("bytes value must start with 0x")
		}
		data, err := hex.DecodeString(paramValue[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid bytes: %s", paramValue)
		}
		return data, nil

	case "bool":
		return paramValue == "true", nil

	case "string":
		return paramValue, nil

	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramType)
	}
}

func CallContract(
	client *ethclient.Client,
	contractAddress common.Address,
	methodName string,
	params []utils.Parameter,
) ([]byte, error) {
	fmt.Printf("\n current params in callcontract: %v", params)
	callData, err := ConstructCallData(methodName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to construct call data: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: callData,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %v", err)
	}

	return result, nil
}

func MulticallView(client *ethclient.Client, multicallAddress common.Address, calls []Calls) ([]MulticallResult, error) {
	var multicallViewInput []Call
	fmt.Print("\ngot multicall far0\n")
	for _, call := range calls {
		c, err := createCall(call.abi, call.contractAddress, call.method, call.params)
		if err != nil {
			return nil, fmt.Errorf("failed to create call: %v", err)
		}

		multicallViewInput = append(multicallViewInput, c)
	}

	parsedJSON, _ := abi.JSON(strings.NewReader(contractAbiMulticall))
	returnData, err := ViewFunction(client, multicallAddress, parsedJSON, "multicallView", multicallViewInput)
	if err != nil {
		return nil, fmt.Errorf("failed to execute multicallView: %v", err)
	}

	data, err := parsedJSON.Unpack("multicallView", returnData)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack multicallView result: %v", err)
	}

	var results []MulticallResult
	for _, v := range data {
		for _, vv := range v.([]struct {
			Success    bool   "json:\"success\""
			ReturnData []byte "json:\"returnData\""
		}) {
			results = append(results, MulticallResult{
				Success:    vv.Success,
				ReturnData: vv.ReturnData,
			})
		}
	}

	return results, nil
}

func parseGenesisParams(r *http.Request) (*GetGenesisBalancesParams, error) {
	q := r.URL.Query()

	chainID := q.Get("chain-id")
	genesis := q.Get("genesis")

	user := q.Get("user")
	if user == "" {
		user = "0x0000000000000000000000000000000000000000"
	} else {
		matched, err := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, user)
		if err != nil {
			return nil, fmt.Errorf("internal regex error: %v", err)
		}
		if !matched {
			return nil, fmt.Errorf("invalid user address: %s", user)
		}
	}

	// Handle nested params manually
	addresses := q["pools.address"]
	pids := q["pools.pid"]

	if len(addresses) != len(pids) {
		return nil, fmt.Errorf("mismatched pool addresses and pids")
	}

	pools := make([]PoolParams, len(addresses))
	for i := 0; i < len(addresses); i++ {
		pools[i] = PoolParams{
			Address: addresses[i],
			PoolId:  pids[i],
		}
	}

	params := &GetGenesisBalancesParams{
		ChainId:        chainID,
		Pools:          pools,
		GenesisAddress: genesis,
		UserAddress:    user,
	}

	return params, nil
}
