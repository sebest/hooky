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

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewQueue creates a new Queue.
func (b *Base) NewQueue(account bson.ObjectId, application string, name string) (queue *Queue, err error) {
	queue = &Queue{
		ID:          bson.NewObjectId(),
		Account:     account,
		Application: application,
		Name:        name,
	}
	err = b.db.C("queues").Insert(queue)
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
