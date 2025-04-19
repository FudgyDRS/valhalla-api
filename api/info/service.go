package infoHandler

import (
	"net/http"

	"github.com/fudgydrs/valhalla-api/pkg/utils"
	// "github.com/sirupsen/logrus"
)

func VersionRequest(r *http.Request, parameters ...interface{}) (interface{}, error) {
	return utils.VersionResponse{
		Version: Version,
	}, nil
}

func GetGenesisBalances(r *http.Request, parameters ...*GetGenesisBalancesParams) (interface{}, error) {
	return utils.VersionResponse{
		Version: Version,
	}, nil
}
