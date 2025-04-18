package infoHandler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/fudgydrs/valhalla-api/pkg/utils"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("\nRecovered from panic: %v", rec)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	handlerWithCORS := utils.EnableCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		var response interface{}
		var err error

		w.Header().Set("Content-Type", "application/json")
		switch query.Get("query") {
		case "version":
			response, err = VersionRequest(r)
			HandleResponse(w, r, response, err)
			return
		case "get-genesis-balances":
			response, err = GetGenesisBalances(r)
			HandleResponse(w, r, response, err)
			return
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(utils.ErrMalformedRequest("Invalid query parameter"))
			return
		}
	}))

	handlerWithCORS.ServeHTTP(w, r)
}

func HandleResponse(w http.ResponseWriter, r *http.Request, response interface{}, err error) {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
}
