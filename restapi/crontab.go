package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
)

// Crontab is a list of Tasks with a common crontab Name.
type Crontab struct {
	// ID is the ID of the Crontab.
	ID string `json:"id"`

	// Account is the ID of the Account owning the Crontab.
	Account string `json:"account"`

	// Name is the crontab's name.
	Name string `json:"name"`
}

// NewCrontabFromModel returns a Crontab object for use with the Rest API
// from a Crontab model.
func NewCrontabFromModel(crontab *models.Crontab) *Crontab {
	return &Crontab{
		ID:      crontab.ID.Hex(),
		Account: crontab.Account.Hex(),
		Name:    crontab.Name,
	}
}

// PutCrontab ...
func PutCrontab(w rest.ResponseWriter, r *rest.Request) {
	name := r.PathParam("name")
	rc := &Crontab{}
	if err := r.DecodeJsonPayload(rc); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := GetBase(r)
	account := GetAccount(r)
	crontab, err := b.NewCrontab(*account, name)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewCrontabFromModel(crontab))
}

// GetCrontab ...
// func GetCrontab(w rest.ResponseWriter, r *rest.Request) {
// 	name := r.PathParam("name")
// 	b := models.GetBase(r)
// 	tasks, err := b.CrontabGet(name)
// 	if err != nil {
// 		rest.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	if len(tasks) == 0 {
// 		rest.NotFound(w, r)
// 		return
// 	}
// 	rc := &Crontab{
// 		Tasks: make([]*Task, len(tasks)),
// 	}
// 	for idx, task := range tasks {
// 		rc.Tasks[idx] = NewTaskFromModel(task)
// 	}
// 	w.WriteJson(rc)
// }
