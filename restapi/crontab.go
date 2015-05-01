package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
)

// Crontab is a list of Tasks with a common crontab Name.
type Crontab struct {
	Name  string  `json:"name"`
	Tasks []*Task `json:"tasks"`
}

// PostCrontab handles POST requests on /crontabs
func (ra *RestAPI) PostCrontab(w rest.ResponseWriter, r *rest.Request) {
	rc := &Crontab{}
	if err := r.DecodeJsonPayload(rc); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for idx, rt := range rc.Tasks {
		task, err := ra.tm.New(rt.URL, rt.Method, rt.Headers, rt.Payload, rt.Schedule, rt.Retry, rc.Name)
		if err != nil {
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rc.Tasks[idx] = NewTaskFromModel(task)
	}
	w.WriteJson(rc)
}
