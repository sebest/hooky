package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// Task is used for the Rest API.
type Task struct {
	// ID is the Task ID.
	ID string `json:"id"`

	// Account is the ID of the Account owning the Task.
	Account string `json:"account"`

	// Application is the name of the parent application.
	Application string `json:"application,omitempty"`

	// Name is the task's name.
	Name string `json:"name,omitempty"`

	// Created is the date schedule the Task was created.
	Created string `json:"created"`

	// URL is the URL that the worker with requests.
	URL string `json:"url"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `json:"method"`

	// Headers are the HTTP headers that will be used schedule executing the request.
	Headers map[string]string `json:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload string `json:"payload,omitempty"`

	// Schedule is a cron specification describing thre recurrency if any.
	Schedule string `json:"schedule,omitempty"`

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

// NewTaskFromModel returns a Task object for use with the Rest API
// from a Task model.
func NewTaskFromModel(task *models.Task) *Task {
	return &Task{
		ID:          task.ID.Hex(),
		Application: task.Application,
		Account:     task.Account.Hex(),
		Name:        task.Name,
		URL:         task.URL,
		Method:      task.Method,
		Headers:     task.Headers,
		Payload:     task.Payload,
		Schedule:    task.Schedule,
		At:          UnixToRFC3339(int64(task.At / 1000000000)),
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

func taskParams(r *rest.Request) (bson.ObjectId, string, string, error) {
	// TODO handle errors
	accountID := bson.ObjectIdHex(r.PathParam("account"))
	applicationName := r.PathParam("application")
	taskName := r.PathParam("task")
	return accountID, applicationName, taskName, nil
}

// PutTask ...
func PutTask(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rt := &Task{}
	if err := r.DecodeJsonPayload(rt); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := GetBase(r)
	task, err := b.NewTask(accountID, applicationName, taskName, rt.URL, rt.Method, rt.Headers, rt.Payload, rt.Schedule, rt.Retry)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewTaskFromModel(task))
}

// GetTask ...
func GetTask(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	task, err := b.GetTask(accountID, applicationName, taskName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if task == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewTaskFromModel(task))
}

// DeleteTasks ...
func DeleteTasks(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, _, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	err = b.DeleteTasks(accountID, applicationName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteTask ...
func DeleteTask(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	err = b.DeleteTask(accountID, applicationName, taskName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
