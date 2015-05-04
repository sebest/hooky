package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
)

// Application is a list of Tasks with a common application Name.
type Application struct {
	// ID is the ID of the Application.
	ID string `json:"id"`

	// Account is the ID of the Account owning the Application.
	Account string `json:"account"`

	// Name is the application's name.
	Name string `json:"name"`
}

// NewApplicationFromModel returns a Application object for use with the Rest API
// from a Application model.
func NewApplicationFromModel(application *models.Application) *Application {
	return &Application{
		ID:      application.ID.Hex(),
		Account: application.Account.Hex(),
		Name:    application.Name,
	}
}

// PutApplication ...
func PutApplication(w rest.ResponseWriter, r *rest.Request) {
	name := r.PathParam("name")
	rc := &Application{}
	if err := r.DecodeJsonPayload(rc); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := GetBase(r)
	account := GetAccount(r)
	application, err := b.NewApplication(*account, name)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewApplicationFromModel(application))
}

// GetApplication ...
// func GetApplication(w rest.ResponseWriter, r *rest.Request) {
// 	name := r.PathParam("name")
// 	b := models.GetBase(r)
// 	tasks, err := b.ApplicationGet(name)
// 	if err != nil {
// 		rest.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	if len(tasks) == 0 {
// 		rest.NotFound(w, r)
// 		return
// 	}
// 	rc := &Application{
// 		Tasks: make([]*Task, len(tasks)),
// 	}
// 	for idx, task := range tasks {
// 		rc.Tasks[idx] = NewTaskFromModel(task)
// 	}
// 	w.WriteJson(rc)
// }
