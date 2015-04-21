package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// RestAPI handles the Rest API endpoints of the service.
type RestAPI struct {
	m   *models.Manager
	api *rest.Api
}

// Task is used for the Rest API.
type Task struct {
	// ID is the Task ID.
	ID string `json:"id"`

	// Created is the date when the Task was created.
	Created string `json:"created"`

	// URL is the URL that the worker with requests.
	URL string `json:"url"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `json:"method"`

	// Headers are the HTTP headers that will be used when executing the request.
	Headers map[string]string `json:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload string `json:"payload,omitempty"`

	// When is a cron specification describing thre recurrency if any.
	When string `json:"when,omitempty"`

	// At is a date representing the next time a attempt will be executed.
	At string `json:"at,omitempty"`

	// Status is either `pending`, `retrying`, `canceled`, `success` or `error`
	Status string `json:"status"`

	// Executed is the date of the last time a attempt was executed.
	Executed string `json:"executed,omitempty"`

	// Active is the task active.
	Active bool `json:"active"`

	// Errors counts the number of attempts that failed.
	Errors int `json:"errors"`

	// LastError is the date of the last attempt in error status
	LastError string `json:"lastError,omitempty"`

	// LastSuccess is the date of the last attempt in success status
	LastSuccess string `json:"lastSuccess,omitempty"`

	// Executions counts the number of attempts that were executed.
	Executions int `json:"executions"`

	// ErrorRate is the rate of errors in percent.
	ErrorRate int `json:"errorRate"`

	// Retry is the retry strategy parameters in case of errors.
	Retry models.Retry `json:"retry"`
}

// UnixToRFC3339 converts a Unix timestamp to a human readable format.
func UnixToRFC3339(ts int64) string {
	if ts > 0 {
		return time.Unix(ts, 0).UTC().Format(time.RFC3339)
	}
	return ""
}

// NewTaskFromModel returns a Task object for use with the Rest API
// from a Task from the model.
func NewTaskFromModel(task *models.Task) *Task {
	return &Task{
		URL:         task.URL,
		Method:      task.Method,
		Headers:     task.Headers,
		Payload:     task.Payload,
		When:        task.When,
		At:          UnixToRFC3339(int64(task.At / 1000000000)),
		ID:          task.ID.Hex(),
		Created:     task.ID.Time().UTC().Format(time.RFC3339),
		Status:      task.Status,
		Executed:    UnixToRFC3339(task.Executed),
		Active:      task.Active,
		Executions:  task.Executions,
		Errors:      task.Errors,
		LastSuccess: UnixToRFC3339(task.LastSuccess),
		LastError:   UnixToRFC3339(task.LastError),
		ErrorRate:   task.ErrorRate(),
		Retry:       task.Retry,
	}
}

// New creates a new instance of the Rest API.
func New(m *models.Manager) (*RestAPI, error) {
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)

	ra := &RestAPI{
		m:   m,
		api: api,
	}

	router, err := rest.MakeRouter(
		rest.Post("/tasks", ra.PostTask),
		rest.Get("/tasks/:taskID", ra.GetTask),
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

// PostTask handles POST requests on /tasks
func (ra *RestAPI) PostTask(w rest.ResponseWriter, r *rest.Request) {
	rt := &Task{}
	if err := r.DecodeJsonPayload(rt); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	task, err := ra.m.Tasks.New(rt.URL, rt.Method, rt.Headers, rt.Payload, rt.When, rt.Retry)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := ra.m.Attempts.New(task); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewTaskFromModel(task))
}

// GetTask handles GET requests on /tasks/:taskID
func (ra *RestAPI) GetTask(w rest.ResponseWriter, r *rest.Request) {
	taskID := r.PathParam("taskID")
	task, err := ra.m.Tasks.Get(bson.ObjectIdHex(taskID))
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewTaskFromModel(task))
}
