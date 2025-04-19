package infoHandler

type GetGenesisBalancesParams struct {
	Tokens         []string `query:"tokens"`
	GenesisAddress string   `query:"genesis"`
	UserAddress    string   `query:"user"`
}
