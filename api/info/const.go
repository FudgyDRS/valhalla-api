package infoHandler

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const Version string = "Valhalla API v0.0.1"

type ChainInfo struct {
	RPC  string
	ID   string
	Name string
}

var SupportedChains = map[string]ChainInfo{
	"1": {
		RPC:  "https://eth.llamarpc.com",
		ID:   "01",
		Name: "Ethereum Mainnet",
	},
	"137": {
		RPC:  "https",
		ID:   "89",
		Name: "Polygon Mainnet",
	},
	"56": {
		RPC:  "https://bsc-rpc.publicnode.com",
		ID:   "38",
		Name: "Binance Smart Chain",
	},
	"146": {
		RPC:  "https://rpc.soniclabs.com",
		ID:   "146",
		Name: "Sonic Mainnet",
	},
	"0x92": {
		RPC:  "https://rpc.soniclabs.com",
		ID:   "146",
		Name: "Sonic Mainnet",
	},
}

func GetChainInfo(chainId string) (ChainInfo, error) {
	chain, exists := SupportedChains[chainId]
	if !exists {
		return ChainInfo{}, fmt.Errorf("chain ID %v not supported", chainId)
	}
	return chain, nil
}

var multicallAddressMap = map[string]string{
	"0x92": "0xd782fF720cbB9c8337e02013eE3ccBb54B5471D9",
	"146":  "0xd782fF720cbB9c8337e02013eE3ccBb54B5471D9",
}

func getMulticallAddress(chainId string) (common.Address, error) {
	if multicallAddress, found := multicallAddressMap[chainId]; found {
		return common.HexToAddress(multicallAddress), nil
	}
	return common.Address{}, fmt.Errorf("multicall address could not be found for %v", chainId)
}
