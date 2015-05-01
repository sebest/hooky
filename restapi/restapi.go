package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
)

// RestAPI handles the Rest API endpoints of the service.
type RestAPI struct {
	tm  *models.TasksManager
	api *rest.Api
}

// New creates a new instance of the Rest API.
func New(tm *models.TasksManager) (*RestAPI, error) {
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)

	ra := &RestAPI{
		tm:  tm,
		api: api,
	}

	router, err := rest.MakeRouter(
		rest.Post("/tasks", ra.PostTask),
		rest.Get("/tasks/:taskID", ra.GetTask),
		rest.Post("/crontabs", ra.PostCrontab), // or PUT /crontabs/NAME
	)
	if err != nil {
		return nil, err
	}
	api.SetApp(router)

	return ra, nil
}

// MakeHandler returns http.Handlers of the Rest API.
func (ra *RestAPI) MakeHandler() http.Handler {
	return ra.api.MakeHandler()
}
