package infoHandler

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

const Version string = "Valhalla API v0.0.1"

type ChainInfo struct {
	RPC  []string
	ID   string
	Name string
}

var SupportedChains = map[string]ChainInfo{
	// "1": {
	// 	RPC:  "https://eth.llamarpc.com",
	// 	ID:   "01",
	// 	Name: "Ethereum Mainnet",
	// },
	// "137": {
	// 	RPC:  "https",
	// 	ID:   "89",
	// 	Name: "Polygon Mainnet",
	// },
	// "56": {
	// 	RPC:  "https://bsc-rpc.publicnode.com",
	// 	ID:   "38",
	// 	Name: "Binance Smart Chain",
	// },
	"146": {
		RPC: []string{
			"https://rpc.soniclabs.com",
			"https://sonic.drpc.org",
			"https://sonic-rpc.publicnode.com",
			"https://rpc.ankr.com/sonic_mainnet",
			"https://sonic.api.onfinality.io/public",
		},
		ID:   "146",
		Name: "Sonic Mainnet",
	},
	"0x92": {
		RPC: []string{
			"https://rpc.soniclabs.com",
			"https://sonic.drpc.org",
			"https://sonic-rpc.publicnode.com",
			"https://rpc.ankr.com/sonic_mainnet",
			"https://sonic.api.onfinality.io/public",
		},
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

func shuffle(rpcs []string) []string {
	shuffled := make([]string, len(rpcs))
	copy(shuffled, rpcs)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func GetClientForChain(chainId string) (*ethclient.Client, error) {
	chain, err := GetChainInfo(chainId)
	if err != nil {
		return nil, err
	}

	rpcs := shuffle(chain.RPC)
	var lastErr error

	for _, rpc := range rpcs {
		client, err := ethclient.Dial(rpc)
		if err == nil {
			return client, nil
		}
		logrus.Warnf("RPC %s failed: %v", rpc, err)
		lastErr = err
	}

	return nil, fmt.Errorf("all RPCs failed for chain %s: %v", chainId, lastErr)
}
