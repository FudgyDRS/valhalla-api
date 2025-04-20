package infoHandler

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type GetGenesisPairResponse struct {
	PairAddress      string `json:"token"`
	PoolId           string `json:"pool-id"`
	BaseBalance      string `json:"base-balance"`
	QuoteBalance     string `json:"quote-balance"`
	GenesisBalance   string `json:"genesis-balance"`
	UserBalance      string `json:"user-balance"`
	UserStake        string `json:"user-stake"`
	UserReward       string `json:"user-reward"`
	UserBaseBalance  string `json:"user-base-balance"`
	UserQuoteBalance string `json:"user-quote-balance"`
}

type GetGenesisBalanceResponse struct {
	Token          string `json:"token"`
	PoolId         string `json:"pool-id"`
	GenesisBalance string `json:"genesis-balance"`
	UserBalance    string `json:"user-balance"`
	UserStake      string `json:"user-stake"`
	UserReward     string `json:"user-reward"`
}

type GetGenesisBalancesResponse struct {
	Pools []GetGenesisBalanceResponse `json:"pools"`
}

type Call struct {
	Target   common.Address
	CallData []byte
}

type Calls struct {
	contractAddress common.Address
	abi             abi.ABI
	method          string
	params          interface{}
}

type MulticallResult struct {
	Success    bool
	ReturnData []byte
}
