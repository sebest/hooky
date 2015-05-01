package models

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
	"github.com/sebest/hooky/store"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Task describes a Webtask.
type Task struct {
	// ID is the ID of the task.
	ID bson.ObjectId `bson:"_id,omitempty"`

	// Crontab is the name of the parent crontab.
	Crontab string `bson:"crontab,omitempty"`

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
}

// ErrorRate is the error rate of the task from 0 to 100 percent.
func (h *Task) ErrorRate() int {
	if h.Executions == 0 {
		return 0
	}
	return int(h.Errors * 100 / h.Executions)
}

// TasksManager ...
type TasksManager struct {
	store    *store.Store
	Attempts *AttemptsManager
}

func (tm *TasksManager) init() {
	db := tm.store.DB()
	defer db.Session.Close()

	index := mgo.Index{
		Key:        []string{"crontab"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	if err := db.C("tasks").EnsureIndex(index); err != nil {
		fmt.Printf("Error creating index on tasks: %s\n", err)
	}
}

// NewTasksManager ...
func NewTasksManager(store *store.Store) *TasksManager {
	tm := &TasksManager{
		store:    store,
		Attempts: NewAttemptsManager(store),
	}
	tm.init()
	return tm
}

func nextRun(schedule string) (int64, error) {
	sched, err := cron.Parse(schedule)
	if err != nil {
		return 0, err
	}
	return sched.Next(time.Now().UTC()).UnixNano(), nil
}

// New creates a new Task.
func (tm *TasksManager) New(URL string, method string, headers map[string]string, payload string, schedule string, retry Retry, crontab string) (task *Task, err error) {
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
		ID:       bson.NewObjectId(),
		Crontab:  crontab,
		URL:      URL,
		Method:   method,
		Headers:  headers,
		Payload:  payload,
		At:       at,
		Status:   "pending",
		Active:   at > 0,
		Schedule: schedule,
		Retry:    retry,
	}
	db := tm.store.DB()
	defer db.Session.Close()
	err = db.C("tasks").Insert(task)
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
		_, err = db.C("tasks").Find(bson.M{"crontab": crontab}).Apply(change, task)
		if err == nil {
			err = tm.Attempts.RemovePending(task.ID)
		}
	}
	if err == nil {
		_, err = tm.Attempts.New(task)
	}
	return
}

// Get returns a Task given its ID.
func (tm *TasksManager) Get(taskID bson.ObjectId) (task *Task, err error) {
	db := tm.store.DB()
	defer db.Session.Close()
	task = &Task{}
	err = db.C("tasks").FindId(taskID).One(task)
	return
}

// NextAttempt enqueue the next Attempt if any and returns it.
func (tm *TasksManager) NextAttempt(taskID bson.ObjectId, status string) (attempt *Attempt, err error) {
	db := tm.store.DB()
	defer db.Session.Close()
	task := &Task{}
	if err = db.C("tasks").FindId(taskID).One(task); err != nil {
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
	_, err = db.C("tasks").FindId(taskID).Apply(change, task)
	if task.Active && task.At != 0 {
		attempt, err = tm.Attempts.New(task)
	}
	return
}
