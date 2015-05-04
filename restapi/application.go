package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
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

func applicationParams(r *rest.Request) (bson.ObjectId, string, error) {
	// TODO handle errors
	accountID := bson.ObjectIdHex(r.PathParam("account"))
	applicationName := r.PathParam("application")
	return accountID, applicationName, nil
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
	accountID, applicationName, err := applicationParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc := &Application{}
	if err := r.DecodeJsonPayload(rc); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := GetBase(r)
	application, err := b.NewApplication(accountID, applicationName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewApplicationFromModel(application))
}

// DeleteApplications ...
func DeleteApplications(w rest.ResponseWriter, r *rest.Request) {
	accountID, _, err := applicationParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	if err := b.DeleteApplications(accountID); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DeleteApplication ...
func DeleteApplication(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, err := applicationParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	if err := b.DeleteApplication(accountID, applicationName); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
