package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// Queue ...
type Queue struct {
	// ID is the ID of the Queue.
	ID string `json:"id"`

	// Created is the date when the Queue was created.
	Created string `json:"created"`

	// Account is the ID of the Account owning the Queue.
	Account string `json:"account"`

	// Application is the name of the parent Application.
	Application string `json:"application"`

	// Name is the queue's name.
	Name string `json:"name"`

	// Retry is the retry strategy parameters in case of errors.
	Retry *models.Retry `json:"retry"`

	// MaxInFlight is the maximum number of attempts executed in parallel.
	MaxInFlight int `json:"maxInFlight"`

	// InFlight is the current number of attempts executed in parallel.
	InFlight int `json:"inFlight"`
}

func queueParams(r *rest.Request) (bson.ObjectId, string, string, error) {
	accountID, err := PathAccountID(r)
	if err != nil {
		return accountID, "", "", err
	}
	// TODO handle errors
	applicationName := r.PathParam("application")
	queueName := r.PathParam("queue")
	return accountID, applicationName, queueName, nil
}

// NewQueueFromModel returns a Queue object for use with the Rest API
// from a Queue model.
func NewQueueFromModel(queue *models.Queue) *Queue {
	return &Queue{
		ID:          queue.ID.Hex(),
		Created:     queue.ID.Time().UTC().Format(time.RFC3339),
		Account:     queue.Account.Hex(),
		Application: queue.Application,
		Name:        queue.Name,
		Retry:       queue.Retry,
		MaxInFlight: queue.MaxInFlight,
		InFlight:    len(queue.AttemptsInFlight),
	}
}

// PutQueue ...
func PutQueue(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, queueName, err := queueParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc := &Queue{}
	if err := r.DecodeJsonPayload(rc); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := GetBase(r)
	queue, err := b.NewQueue(accountID, applicationName, queueName, rc.Retry, rc.MaxInFlight)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if queue == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewQueueFromModel(queue))
}

// GetQueue ...
func GetQueue(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, queueName, err := queueParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	queue, err := b.GetQueue(accountID, applicationName, queueName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if queue == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewQueueFromModel(queue))
}

// DeleteQueue ...
func DeleteQueue(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, queueName, err := queueParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	if err := b.DeleteQueue(accountID, applicationName, queueName); err != nil {
		if err == models.ErrDeleteDefaultApplication {
			rest.Error(w, err.Error(), http.StatusForbidden)
		} else {
			rest.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// DeleteQueues ...
func DeleteQueues(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, _, err := queueParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	if err := b.DeleteQueues(accountID, applicationName); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetQueues ...
func GetQueues(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, _, err := queueParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	lp := parseListQuery(r)
	var queues []*models.Queue
	lr := &models.ListResult{
		List: &queues,
	}

	if err := b.GetQueues(accountID, applicationName, lp, lr); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lr.Count == 0 {
		rest.NotFound(w, r)
		return
	}
	rt := make([]*Queue, len(queues))
	for idx, queue := range queues {
		rt[idx] = NewQueueFromModel(queue)
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
