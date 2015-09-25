package models

import (
	"errors"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// DefaultMaxInFlight is the maximum number of tasks executed in parallel.
	DefaultMaxInFlight = 10
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

	// MaxInFlight is the maximum number of attempts executed in parallel.
	MaxInFlight int `bson:"max_in_flight"`

	// AvailableInFlight is the available number of slots to execute tasks in parallel.
	AvailableInFlight int `bson:"available_in_flight"`

	// AttemptsInFlight is the list of attempts currently in flight.
	AttemptsInFlight []bson.ObjectId `bson:"attempts_in_flight"`
}

// NewQueue creates a new Queue.
func (b *Base) NewQueue(account bson.ObjectId, applicationName string, name string, retry *Retry, maxInFlight int) (queue *Queue, err error) {
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
	// Define default parameter for maxInFlight.
	if maxInFlight == 0 {
		maxInFlight = DefaultMaxInFlight
	}

	queue = &Queue{
		ID:                bson.NewObjectId(),
		Account:           account,
		Application:       applicationName,
		Name:              name,
		Retry:             retry,
		MaxInFlight:       maxInFlight,
		AvailableInFlight: maxInFlight,
	}
	err = b.db.C("queues").Insert(queue)
	_, err = b.ShouldRefreshSession(err)
	if mgo.IsDup(err) {
		query := bson.M{
			"account":     queue.Account,
			"application": queue.Application,
			"name":        queue.Name,
		}
		if err = b.db.C("queues").Find(query).One(queue); err != nil {
			_, err = b.ShouldRefreshSession(err)
			return nil, err
		}
		incMaxInFlight := maxInFlight - queue.MaxInFlight
		change := mgo.Change{
			Update: bson.M{
				"$set": bson.M{
					"retry": retry,
				},
				"$inc": bson.M{
					"max_in_flight":       incMaxInFlight,
					"available_in_flight": incMaxInFlight,
				},
			},
			ReturnNew: true,
		}
		_, err = b.db.C("queues").Find(query).Apply(change, queue)
		_, err = b.ShouldRefreshSession(err)
	} else {
		queue = nil
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
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		queue = nil
		if err == mgo.ErrNotFound {
			err = ErrQueueNotFound
		}
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
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	// TODO update tasks using this queue to default queue
	// TODO update pending attemps to default queue
	query := bson.M{
		"account":     account,
		"application": application,
		"queue":       name,
	}
	_, err = b.db.C("attempts").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return
	}
	_, err = b.db.C("tasks").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return
	}

	query = bson.M{
		"account":     account,
		"application": application,
		"name":        name,
	}
	_, err = b.db.C("queues").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	return
}

// DeleteQueues deletes all Queues owns by an Account.
func (b *Base) DeleteQueues(account bson.ObjectId, application string) (err error) {
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}

	query := bson.M{
		"account":     account,
		"application": application,
	}
	_, err = b.db.C("attempts").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return
	}

	_, err = b.db.C("tasks").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return
	}

	query = bson.M{
		"account":     account,
		"application": application,
		"name":        bson.M{"$ne": "default"},
	}
	_, err = b.db.C("queues").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	return
}

// EnQueue checks if a queue reached its max_in_flight.
func (b *Base) EnQueue(queueID bson.ObjectId, attemptID bson.ObjectId) (full bool, err error) {
	query := bson.M{
		"_id":                queueID,
		"attempts_in_flight": attemptID,
	}
	nb, err := b.db.C("queues").Find(query).Count()
	_, err = b.ShouldRefreshSession(err)
	if err != nil && err != mgo.ErrNotFound {
		return false, err
	}
	// this attemptID is already in the queue
	if nb == 1 {
		return false, nil
	}
	query = bson.M{
		"_id": queueID,
		"available_in_flight": bson.M{"$gt": 0},
		"attempts_in_flight":  bson.M{"$ne": attemptID},
	}
	update := bson.M{
		"$inc":  bson.M{"available_in_flight": -1},
		"$push": bson.M{"attempts_in_flight": attemptID},
	}
	if err := b.db.C("queues").Update(query, update); err != nil {
		_, err := b.ShouldRefreshSession(err)
		if err == mgo.ErrNotFound {
			// the queue is full
			return true, nil
		}
		return false, err
	}
	// the queue is not full
	return false, nil
}

// DeQueue increases the available_in_flight by one.
func (b *Base) DeQueue(queueID bson.ObjectId, attemptID bson.ObjectId) (err error) {
	query := bson.M{
		"_id":                queueID,
		"attempts_in_flight": attemptID,
	}
	update := bson.M{
		"$inc":  bson.M{"available_in_flight": 1},
		"$pull": bson.M{"attempts_in_flight": attemptID},
	}
	err = b.db.C("queues").Update(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err == mgo.ErrNotFound {
		err = nil
	}
	return
}

// FixQueues fixes unconsistencies with available_in_flight.
func (b *Base) FixQueues() (err error) {
	return
}

// EnsureQueueIndex creates mongo indexes for Queue.
func (b *Base) EnsureQueueIndex() (err error) {
	index := mgo.Index{
		Key:        []string{"account", "application", "name"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	err = b.db.C("queues").EnsureIndex(index)
	_, err = b.ShouldRefreshSession(err)
	return
}
