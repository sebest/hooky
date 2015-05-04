package models

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Task describes a Webtask.
type Task struct {
	// ID is the ID of the Task.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Task.
	Account bson.ObjectId `bson:"account"`

	// Application is the name of the parent application.
	Application string `bson:"application"`

	// Name is the task's name.
	Name string `bson:"name"`

	// URL is the URL that the worker with requests.
	URL string `bson:"url"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `bson:"method"`

	// Headers are the HTTP headers that will be used when executing the request.
	Headers map[string]string `bson:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload string `bson:"payload,omitempty"`

	// Schedule is a cron specification describing the recurrency if any.
	Schedule string `bson:"schedule,omitempty"`

	// At is a Unix timestamp representing the next time a request must be performed.
	At int64 `bson:"at"`

	// Status is either `pending`, `retrying`, `canceled`, `success` or `error`
	Status string `bson:"status"`

	// Executed is the timestamp of the last time a attempt was executed.
	Executed int64 `bson:"executed,omitempty"`

	// Active is the task active.
	Active bool `bson:"active"`

	// Errors counts the number of attempts that failed.
	Errors int `bson:"errors,omitempty"`

	// LastError is the timestamp of the last attempt in error status
	LastError int64 `bson:"last_error,omitempty"`

	// LastSuccess is the timestamp of the last attempt in success status
	LastSuccess int64 `bson:"last_success,omitempty"`

	// Executions counts the number of attempts that were executed.
	Executions int `bson:"executions,omitempty"`

	// Retry is the retry strategy parameters in case of errors.
	Retry Retry `bson:"retry"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// ErrorRate is the error rate of the task from 0 to 100 percent.
func (h *Task) ErrorRate() int {
	if h.Executions == 0 {
		return 0
	}
	return int(h.Errors * 100 / h.Executions)
}

func nextRun(schedule string) (int64, error) {
	sched, err := cron.Parse(schedule)
	if err != nil {
		return 0, err
	}
	return sched.Next(time.Now().UTC()).UnixNano(), nil
}

// NewTask creates a new Task.
func (b *Base) NewTask(account bson.ObjectId, application string, name string, URL string, method string, headers map[string]string, payload string, schedule string, retry Retry) (task *Task, err error) {
	taskID := bson.NewObjectId()
	if application == "" {
		application = "default"
	}
	if name == "" {
		name = taskID.Hex()
	}

	// Default method is POST.
	if method == "" {
		method = "POST"
	}
	// Payload is only valid for POST requests.
	if method != "POST" {
		payload = ""
	}
	// If `schedule` is defined we compute the next date of the first attempt,
	// otherwise it is right now.
	var at int64
	if schedule != "" {
		at, err = nextRun(schedule)
		if err != nil {
			return
		}
	} else {
		at = time.Now().UnixNano()
	}
	// Define default parameters for our retry strategy.
	if retry.MaxAttempts == 0 {
		retry.MaxAttempts = 1
	}
	if retry.Max == 0 {
		retry.Max = 300
	}
	if retry.Min == 0 {
		retry.Min = 10
	}
	if retry.Factor == 0 {
		retry.Factor = 2
	}
	if retry.MaxAttempts == 0 {
		retry.MaxAttempts = 60
	}

	// Create a new `Task` and store it.
	task = &Task{
		ID:          taskID,
		Account:     account,
		Application: application,
		Name:        name,
		URL:         URL,
		Method:      method,
		Headers:     headers,
		Payload:     payload,
		At:          at,
		Status:      "pending",
		Active:      at > 0,
		Schedule:    schedule,
		Retry:       retry,
	}
	err = b.db.C("tasks").Insert(task)
	if mgo.IsDup(err) {
		change := mgo.Change{
			Update: bson.M{
				"$set": bson.M{
					"url":      URL,
					"method":   method,
					"headers":  headers,
					"payload":  payload,
					"at":       at,
					"active":   at > 0,
					"schedule": schedule,
					"retry":    retry,
				},
			},
			ReturnNew: true,
		}
		query := bson.M{
			"account":     account,
			"application": application,
			"name":        name,
		}
		_, err = b.db.C("tasks").Find(query).Apply(change, task)
		if err == nil {
			err = b.DeletePendingAttempts(task.ID)
		}
	}
	if err == nil {
		_, err = b.NewAttempt(task)
	}
	return
}

// GetTask returns a Task.
func (b *Base) GetTask(account bson.ObjectId, application string, name string) (task *Task, err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"name":        name,
		"deleted":     false,
	}
	task = &Task{}
	err = b.db.C("tasks").Find(query).One(task)
	if err == mgo.ErrNotFound {
		err = nil
		task = nil
	}
	return
}

// GetTaskByID returns a Task given its ID.
func (b *Base) GetTaskByID(taskID bson.ObjectId) (task *Task, err error) {
	task = &Task{}
	err = b.db.C("tasks").FindId(taskID).One(task)
	if err == mgo.ErrNotFound {
		err = nil
		task = nil
	}
	return
}

// NextAttemptForTask enqueue the next Attempt if any and returns it.
func (b *Base) NextAttemptForTask(taskID bson.ObjectId, status string) (attempt *Attempt, err error) {
	task := &Task{}
	if err = b.db.C("tasks").FindId(taskID).One(task); err != nil {
		return nil, err
	}
	var at int64
	if task.Active && task.Schedule != "" {
		at, err = nextRun(task.Schedule)
	}

	now := time.Now().UTC()

	errors := 0
	retryAttempts := 1
	if status == "error" {
		errors = 1

		at, err = task.Retry.NextAttempt(now.UnixNano())
		if err == nil {
			status = "retrying"
		}
	} else if status == "success" {
		retryAttempts = -task.Retry.Attempts
	}

	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"status":         status,
				"updated":        now.Unix(),
				"executed":       now.Unix(),
				"last_" + status: now.Unix(),
				"at":             at,
				"active":         at > 0,
			},
			"$inc": bson.M{
				"executions":     1,
				"errors":         errors,
				"retry.attempts": retryAttempts,
			},
		},
		ReturnNew: true,
	}
	_, err = b.db.C("tasks").FindId(taskID).Apply(change, task)
	if task.Active && task.At != 0 && !task.Deleted {
		attempt, err = b.NewAttempt(task)
	}
	return
}

// DeleteTasks deletes all Tasks from an Application.
func (b *Base) DeleteTasks(account bson.ObjectId, application string) (err error) {
	query := bson.M{
		"account":     account,
		"application": application,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
		_, err = b.db.C("attempts").UpdateAll(query, update)
	}
	return
}

// DeleteTask deletes a Task.
func (b *Base) DeleteTask(account bson.ObjectId, application string, task string) (err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"name":        task,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
		query := bson.M{
			"account":     account,
			"application": application,
			"task":        task,
		}
		_, err = b.db.C("attempts").UpdateAll(query, update)
	}
	return
}

// EnsureTaskIndex creates mongo indexes for Application.
func (b *Base) EnsureTaskIndex() {
	index := mgo.Index{
		Key:        []string{"account", "application", "name"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	if err := b.db.C("tasks").EnsureIndex(index); err != nil {
		fmt.Printf("Error creating index on tasks: %s\n", err)
	}
}
