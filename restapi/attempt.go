package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
)

// Attempt is used for the Rest API.
type Attempt struct {
	// ID is the Attempt ID.
	ID string `json:"id"`

	// Account is the ID of the Account owning the Task.
	Account string `json:"account"`

	// Application is the name of the parent Application.
	Application string `json:"application"`

	// Task is the task's name.
	Task string `json:"name"`

	// TaskID is the ID of the parent Webtask of this attempt.
	TaskID string `json:"task_id"`

	// Queue is the name of the parent Queue.
	Queue string `json:"queue"`

	// Created is the date schedule the Task was created.
	Created string `json:"created"`

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

	// Schedule is a cron specification describing the recurrency if any.
	Schedule string `json:"schedule,omitempty"`

	// At is a date representing the next time a attempt will be executed.
	At string `json:"at,omitempty"`

	// Status is either `pending`, `retrying`, `canceled`, `success` or `error`
	Status string `json:"status"`

	// StatusCode is the HTTP status code.
	StatusCode int32 `json:"status_code,omitempty"`

	// StatusMessage is a human readable message related to the Status.
	StatusMessage string `json:"status_message,omitempty"`
}

// NewAttemptFromModel returns a Task object for use with the Rest API
// from a Task model.
func NewAttemptFromModel(attempt *models.Attempt) *Attempt {
	return &Attempt{
		ID:            attempt.ID.Hex(),
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
		Created:       attempt.ID.Time().UTC().Format(time.RFC3339),
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
