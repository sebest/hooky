package restapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ErrInvalidAttemptID is returned when an invalid Attempt ID is found.
	ErrInvalidAttemptID = errors.New("invalid attempt ID")
)

// Attempt is used for the Rest API.
type Attempt struct {
	// ID is the Attempt ID.
	ID string `json:"id"`

	// Created is the date when the Attempt was created.
	Created string `json:"created"`

	// Account is the ID of the Account owning the Task.
	Account string `json:"account"`

	// Application is the name of the parent Application.
	Application string `json:"application"`

	// Task is the task's name.
	Task string `json:"name"`

	// TaskID is the ID of the parent Task of this attempt.
	TaskID string `json:"taskID"`

	// Queue is the name of the parent Queue.
	Queue string `json:"queue"`

	// URL is the URL that the worker with requests.
	URL string `json:"url"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth models.HTTPAuth `json:"auth"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `json:"method"`

	// Headers are the HTTP headers that will be used schedule executing the request.
	Headers map[string]string `json:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload string `json:"payload,omitempty"`

	// At is a date representing the time this attempt will be executed.
	At string `json:"at,omitempty"`

	// Finished is a Unix timestamp representing the time the attempt finished.
	Finished string `json:"finished,omitempty"`

	// Status is either `pending`, `retrying`, `canceled`, `success` or `error`
	Status string `json:"status"`

	// StatusCode is the HTTP status code.
	StatusCode int32 `json:"statusCode,omitempty"`

	// StatusMessage is a human readable message related to the StatusCode.
	StatusMessage string `json:"statusMessage,omitempty"`
}

// NewAttemptFromModel returns a Task object for use with the Rest API
// from a Task model.
func NewAttemptFromModel(attempt *models.Attempt) *Attempt {
	return &Attempt{
		ID:            attempt.ID.Hex(),
		Created:       attempt.ID.Time().UTC().Format(time.RFC3339),
		Application:   attempt.Application,
		Account:       attempt.Account.Hex(),
		Queue:         attempt.Queue,
		Task:          attempt.Task,
		TaskID:        attempt.TaskID.Hex(),
		URL:           attempt.URL,
		Method:        attempt.Method,
		HTTPAuth:      attempt.HTTPAuth,
		Headers:       attempt.Headers,
		Payload:       attempt.Payload,
		At:            UnixToRFC3339(int64(attempt.At / 1000000000)),
		Finished:      UnixToRFC3339(attempt.Finished),
		Status:        attempt.Status,
		StatusCode:    attempt.StatusCode,
		StatusMessage: attempt.StatusMessage,
	}
}

// GetAttempts ...
func GetAttempts(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	lp := parseListQuery(r)
	var attempts []*models.Attempt
	lr := &models.ListResult{
		List: &attempts,
	}

	if err := b.GetAttempts(accountID, applicationName, taskName, lp, lr); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lr.Count == 0 {
		rest.NotFound(w, r)
		return
	}
	rt := make([]*Attempt, len(attempts))
	for idx, attempt := range attempts {
		rt[idx] = NewAttemptFromModel(attempt)
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

// PostAttempt ...
func PostAttempt(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	attempt, err := b.ForceAttemptForTask(accountID, applicationName, taskName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if attempt == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewAttemptFromModel(attempt))
}

// GetAttempt ...
func GetAttempt(w rest.ResponseWriter, r *rest.Request) {
	attemptID := r.PathParam("attempt")
	if !bson.IsObjectIdHex(attemptID) {
		rest.Error(w, ErrInvalidAttemptID.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	attempt, err := b.GetAttemptByID(bson.ObjectIdHex(attemptID))
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if attempt == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewAttemptFromModel(attempt))
}
