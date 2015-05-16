package restapi

import (
	"expvar"

	"github.com/ant0ine/go-json-rest/rest"
)

func GetStatus(w rest.ResponseWriter, r *rest.Request) {
	// b := GetBase(r)
	status := make(map[string]string)
	status["status"] = "ok"
	status["attemptsError"] = expvar.Get("attemptsError").String()
	status["attemptsSuccess"] = expvar.Get("attemptsSuccess").String()
	w.WriteJson(status)
}
