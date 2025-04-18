package infoHandler

import (
	"net/http"

	"github.com/fudgydrs/valhalla-api/pkg/utils"
	// "github.com/sirupsen/logrus"
)

func GetGenesisBalances(r *http.Request, parameters ...interface{}) (interface{}, error) {
	return utils.VersionResponse{
		Version: Version,
	}, nil
}

func GetPairsRequest(r *http.Request, parameters ...interface{}) (interface{}, error) {
	return GetPairsResponse{
		Pairs: Pairs,
	}, nil
}
