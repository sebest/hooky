package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// Application is a list of Tasks with a common application Name.
type Application struct {
	// ID is the ID of the Application.
	ID string `json:"id"`

	// Created is the date when the Application was created.
	Created string `json:"created"`

	// Account is the ID of the Account owning the Application.
	Account string `json:"account"`

	// Name is the application's name.
	Name string `json:"name"`
}

func applicationParams(r *rest.Request) (bson.ObjectId, string, error) {
	// TODO handle errors
	accountID := bson.ObjectIdHex(r.PathParam("account"))
	applicationName := r.PathParam("application")
	if applicationName == "" {
		applicationName = "default"
	}
	return accountID, applicationName, nil
}

// NewApplicationFromModel returns a Application object for use with the Rest API
// from a Application model.
func NewApplicationFromModel(application *models.Application) *Application {
	return &Application{
		ID:      application.ID.Hex(),
		Created: application.ID.Time().UTC().Format(time.RFC3339),
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

// GetApplication ...
func GetApplication(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, err := applicationParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	application, err := b.GetApplication(accountID, applicationName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if application == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewApplicationFromModel(application))
}

// GetApplications ...
func GetApplications(w rest.ResponseWriter, r *rest.Request) {
	accountID, _, err := applicationParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	lp := parseListQuery(r)
	var applications []*models.Application
	lr := &models.ListResult{
		List: &applications,
	}

	if err := b.GetApplications(accountID, lp, lr); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lr.Count == 0 {
		rest.NotFound(w, r)
		return
	}
	rt := make([]*Application, len(applications))
	for idx, application := range applications {
		rt[idx] = NewApplicationFromModel(application)
	}
	w.WriteJson(models.ListResult{
		List:    rt,
		HasMore: lr.HasMore,
		Total:   lr.Total,
		Count:   lr.Count,
		Page:    lr.Page,
		Pages:   lr.Pages,
	})
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
		if err == models.ErrDeleteDefaultApplication {
			rest.Error(w, err.Error(), http.StatusForbidden)
		} else {
			rest.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
