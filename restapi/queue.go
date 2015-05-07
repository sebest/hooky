package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// Queue is a list of Tasks with a common queue Name.
type Queue struct {
	// ID is the ID of the Queue.
	ID string `json:"id"`

	// Account is the ID of the Account owning the Queue.
	Account string `json:"account"`

	// Application is the name of the parent Application.
	Application string `bson:"application"`

	// Name is the queue's name.
	Name string `json:"name"`
}

func queueParams(r *rest.Request) (bson.ObjectId, string, string, error) {
	// TODO handle errors
	accountID := bson.ObjectIdHex(r.PathParam("account"))
	applicationName := r.PathParam("application")
	queueName := r.PathParam("queue")
	return accountID, applicationName, queueName, nil
}

// NewQueueFromModel returns a Queue object for use with the Rest API
// from a Queue model.
func NewQueueFromModel(queue *models.Queue) *Queue {
	return &Queue{
		ID:          queue.ID.Hex(),
		Account:     queue.Account.Hex(),
		Application: queue.Application,
		Name:        queue.Name,
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
	queue, err := b.NewQueue(accountID, applicationName, queueName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewQueueFromModel(queue))
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
