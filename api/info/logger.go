package infoHandler

import "github.com/sirupsen/logrus"

const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorCyan   = "\033[36m"
	ColorYellow = "\033[33m"
)

func LogGenesisParams(params *GetGenesisBalancesParams) {
	logrus.Infof("%sChain ID:%s        %s", ColorCyan, ColorReset, params.ChainId)
	logrus.Infof("%sGenesis Address:%s %s", ColorCyan, ColorReset, params.GenesisAddress)
	logrus.Infof("%sUser Address:%s    %s", ColorCyan, ColorReset, params.UserAddress)

	logrus.Infof("%sPools:%s", ColorYellow, ColorReset)
	for i, pool := range params.Pools {
		logrus.Infof("  %s[%d]%s Address: %s%s%s  PID: %s%s%s",
			ColorBlue, i, ColorReset,
			ColorGreen, pool.Address, ColorReset,
			ColorGreen, pool.PoolId, ColorReset,
		)
	}
}
