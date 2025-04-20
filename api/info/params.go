package infoHandler

type PoolParams struct {
	Address string `query:"address"`
	PoolId  string `query:"pid"`
}

type GetGenesisBalancesParams struct {
	ChainId        string       `query:"chain-id"`
	Pools          []PoolParams `query:"pools"`
	GenesisAddress string       `query:"genesis"`
	UserAddress    string       `query:"user" optional:"true"`
}

type GetGenesisPairParams struct {
	ChainId        string `query:"chain-id"`
	GenesisAddress string `query:"genesis"`
	PairAddress    string `query:"pair"`
	BaseAddress    string `query:"base"`
	QuoteAddress   string `query:"quote"`
	UserAddress    string `query:"user" optional:"true"`
	PoolId         string `query:"pid"`
}
