package models

import (
	"errors"
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ErrDeleteDefaultQueue is returned when trying to delete the default queue.
	ErrDeleteDefaultQueue = errors.New("can not delete default queue")
	// ErrQueueNotFound is returned when the queue does not exist.
	ErrQueueNotFound = errors.New("queue does not exist")
)

// Queue ...
type Queue struct {
	// ID is the ID of the Queue.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Queue.
	Account bson.ObjectId `bson:"account"`

	// Application is the name of the parent Application.
	Application string `bson:"application"`

	// Name is the Queue's name.
	Name string `bson:"name"`

	// Retry is the retry strategy parameters in case of errors.
	Retry *Retry `bson:"retry"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewQueue creates a new Queue.
func (b *Base) NewQueue(account bson.ObjectId, applicationName string, name string, retry *Retry) (queue *Queue, err error) {
	application, err := b.GetApplication(account, applicationName)
	if application == nil {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return
	}
	// Define default parameters for our retry strategy.
	if retry == nil {
		retry = &Retry{}
	}
	retry.SetDefault()

	queue = &Queue{
		ID:          bson.NewObjectId(),
		Account:     account,
		Application: applicationName,
		Name:        name,
		Retry:       retry,
	}
	err = b.db.C("queues").Insert(queue)
	if mgo.IsDup(err) {
		change := mgo.Change{
			Update: bson.M{
				"$set": bson.M{
					"retry": queue.Retry,
				},
			},
			ReturnNew: true,
		}
		query := bson.M{
			"account":     queue.Account,
			"application": queue.Application,
			"name":        queue.Name,
		}
		_, err = b.db.C("queues").Find(query).Apply(change, queue)
	}
	return
}

// GetQueue returns a Queue.
func (b *Base) GetQueue(account bson.ObjectId, application string, name string) (queue *Queue, err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"name":        name,
		"deleted":     false,
	}
	queue = &Queue{}
	err = b.db.C("queues").Find(query).One(queue)
	if err == mgo.ErrNotFound {
		err = nil
		queue = nil
	}
	return
}

// GetQueues returns a list of Queues.
func (b *Base) GetQueues(account bson.ObjectId, application string, lp ListParams, lr *ListResult) (err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"deleted":     false,
	}
	return b.getItems("queues", query, lp, lr)
}

// DeleteQueue deletes an Queue and all its children.
func (b *Base) DeleteQueue(account bson.ObjectId, application string, name string) (err error) {
	if name == "default" {
		return ErrDeleteDefaultQueue
	}
	query := bson.M{
		"account":     account,
		"application": application,
		"name":        name,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	// TODO update taks using this queue to default queue
	// TODO update pending attemps to default queue
	if _, err = b.db.C("queues").UpdateAll(query, update); err == nil {
		query := bson.M{
			"account":     account,
			"application": application,
			"queue":       name,
		}
		if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
			_, err = b.db.C("attempts").UpdateAll(query, update)
		}
	}
	return
}

// DeleteQueues deletes all Queues owns by an Account.
func (b *Base) DeleteQueues(account bson.ObjectId, application string) (err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"name":        bson.M{"$ne": "default"},
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("queues").UpdateAll(query, update); err == nil {
		query = bson.M{
			"account":     account,
			"application": application,
		}
		if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
			_, err = b.db.C("attempts").UpdateAll(query, update)
		}
	}
	return
}

// EnsureQueueIndex creates mongo indexes for Queue.
func (b *Base) EnsureQueueIndex() {
	index := mgo.Index{
		Key:        []string{"account", "application", "name"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	if err := b.db.C("queues").EnsureIndex(index); err != nil {
		fmt.Printf("Error creating index on queues: %s\n", err)
	}
}
