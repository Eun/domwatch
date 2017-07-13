package api1

import (
	"net/http"
	"strings"
)

func (api *API) statsRoute(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Method, "GET") == false {
		api.writeError(w, "Must be a GET request")
		return
	}

	var result struct {
		Domains int
		Users   int
	}
	err := api.db.Raw("SELECT (SELECT count(*) FROM domains) AS 'domains', (SELECT count(*) FROM emails) AS 'users'").Scan(&result).Error
	if err != nil {
		api.logError(w, err)
		return
	}

	api.writeSuccessResponse(w, result)

}
