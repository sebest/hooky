package models

import (
	"log"
	"time"

	"github.com/robfig/cron"
	"github.com/tj/go-debug"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ModelsTaskDebug ...
	ModelsTaskDebug = debug.Debug("hooky.models.task")
)

// TaskStatuses are the differents statuses that a Task can have.
var TaskStatuses = map[string]bool{
	"pending":  true,
	"retrying": true,
	"canceled": true,
	"success":  true,
	"error":    true,
}

// Task describes a Task.
type Task struct {
	// ID is the ID of the Task.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Task.
	Account bson.ObjectId `bson:"account"`

	// Application is the name of the parent Application.
	Application string `bson:"application"`

	// Name is the task's name.
	Name string `bson:"name"`

	// Queue is the name of the parent Queue.
	Queue string `bson:"queue"`

	// QueueID is the ID of the parent Queue
	QueueID bson.ObjectId `bson:"queue_id"`

	// URL is the URL that the worker with requests.
	URL string `bson:"url"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth HTTPAuth `bson:"auth"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `bson:"method"`

	// Headers are the HTTP headers that will be used when executing the request.
	Headers map[string]string `bson:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload string `bson:"payload,omitempty"`

	// Schedule is a cron specification describing the recurrency if any.
	Schedule string `bson:"schedule"`

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
	Retry *Retry `bson:"retry"`

	// CurrentAttempt is the current attempt ID for this task.
	CurrentAttempt bson.ObjectId `bson:"current_attempt"`

	// AttemptQueued is set to true when the attempt as been successfully created.
	AttemptQueued bool `bson:"attempt_queued"`

	// AttemptUpdated is the Unix timestamp of the last update of the current attempt.
	AttemptUpdated int64 `bson:"attempt_updated"`

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
func (b *Base) NewTask(account bson.ObjectId, applicationName string, name string, queueName string, URL string, auth HTTPAuth, method string, headers map[string]string, payload string, schedule string, retry *Retry, active bool) (task *Task, err error) {
	application, err := b.GetApplication(account, applicationName)
	if application == nil {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return
	}
	taskID := bson.NewObjectId()
	// If no name is provided we use the Task ID
	if name == "" {
		name = taskID.Hex()
	}
	// Default queue is 'default'
	if queueName == "" {
		queueName = "default"
	}
	queue, err := b.GetQueue(account, applicationName, queueName)
	if queue == nil {
		return nil, ErrQueueNotFound
	}
	if err != nil {
		return
	}
	// Default method is POST.
	if method == "" {
		method = "POST"
	}
	// Payload is only valid for POST requests.
	if method != "POST" {
		payload = ""
	}
	// Now as a Unix timestamp in nanoseconds
	nowNano := time.Now().UnixNano()
	// If schedule is defined we compute the next date of the first attempt,
	// otherwise it is right now.
	var at int64
	if schedule != "" {
		at, err = nextRun(schedule)
		if err != nil {
			return
		}
	} else {
		at = nowNano
	}
	// Define default parameters for our retry strategy.
	if retry == nil {
		if queue.Retry != nil {
			retry = queue.Retry
		} else {
			retry = &Retry{}
		}
	}
	retry.SetDefault()

	currentAttempt := bson.NewObjectId()

	// Create a new Task and store it.
	task = &Task{
		ID:             taskID,
		Account:        account,
		Application:    applicationName,
		Queue:          queue.Name,
		QueueID:        queue.ID,
		Name:           name,
		URL:            URL,
		HTTPAuth:       auth,
		Method:         method,
		Headers:        headers,
		Payload:        payload,
		At:             at,
		Status:         "pending",
		Active:         at > 0 && active,
		Schedule:       schedule,
		CurrentAttempt: currentAttempt,
		AttemptUpdated: nowNano,
		Retry:          retry,
	}
	err = b.db.C("tasks").Insert(task)
	_, err = b.ShouldRefreshSession(err)
	if err == nil {
		_, err = b.NewAttempt(task, false, false)
		if err != nil {
			log.Printf("NewTask error while adding an attempt: %s\n", err)
			err = nil
		}
	} else if mgo.IsDup(err) {
		change := mgo.Change{
			Update: bson.M{
				"$set": bson.M{
					"url":             task.URL,
					"method":          task.Method,
					"headers":         task.Headers,
					"payload":         task.Payload,
					"at":              task.At,
					"active":          task.At > 0 && active,
					"schedule":        task.Schedule,
					"retry":           task.Retry,
					"auth":            task.HTTPAuth,
					"current_attempt": currentAttempt,
					"attempt_queued":  false,
					"attempt_updated": nowNano,
					"deleted":         false,
				},
			},
			ReturnNew: true,
		}
		query := bson.M{
			"account":     task.Account,
			"application": task.Application,
			"name":        task.Name,
		}
		_, err = b.db.C("tasks").Find(query).Apply(change, task)
		_, err = b.ShouldRefreshSession(err)
		if err == nil {
			_, err = b.NewAttempt(task, true, false)
			if err != nil {
				log.Printf("NewTask error while adding an attempt (delete pending): %s\n", err)
				err = nil
			}
		}
	} else {
		task = nil
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
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		task = nil
		if err == mgo.ErrNotFound {
			err = nil
		}
	}
	return
}

// GetTaskByID returns a Task given its ID.
func (b *Base) GetTaskByID(taskID bson.ObjectId) (task *Task, err error) {
	task = &Task{}
	err = b.db.C("tasks").FindId(taskID).One(task)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		task = nil
		if err == mgo.ErrNotFound {
			err = nil
		}
	}
	return
}

// DeleteTask deletes a Task.
func (b *Base) DeleteTask(account bson.ObjectId, application string, name string) (err error) {
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	query := bson.M{
		"account":     account,
		"application": application,
		"task":        name,
	}
	_, err = b.db.C("attempts").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return
	}

	query = bson.M{
		"account":     account,
		"application": application,
		"name":        name,
	}
	_, err = b.db.C("tasks").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
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
	_, err = b.db.C("attempts").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return
	}

	_, err = b.db.C("tasks").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	return
}

// GetTasks returns a list of Tasks.
func (b *Base) GetTasks(account bson.ObjectId, application string, lp ListParams, lr *ListResult) (err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"deleted":     false,
	}
	if value, ok := lp.Filters["schedule"]; ok {
		if value == "true" {
			query["schedule"] = bson.M{"$ne": ""}
		} else if value == "false" {
			query["schedule"] = ""
		}
	}
	if value, ok := lp.Filters["status"]; ok {
		_, ok := TaskStatuses[value]
		if ok {
			query["status"] = value
		}
	}
	return b.getItems("tasks", query, lp, lr)
}

// SetAttemptQueuedForTask marks the attempt as queued for a given task ID.
func (b *Base) SetAttemptQueuedForTask(task *Task) (err error) {
	query := bson.M{
		"_id":             task.ID,
		"current_attempt": task.CurrentAttempt,
		"attempt_queued":  false,
	}
	update := bson.M{
		"$set": bson.M{
			"attempt_queued":  true,
			"attempt_updated": time.Now().UnixNano(),
		},
	}
	_, err = b.db.C("tasks").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	return
}

// NextAttemptForTask enqueue the next Attempt if any and returns it.
func (b *Base) NextAttemptForTask(attempt *Attempt) (nextAttempt *Attempt, err error) {
	status := attempt.Status
	taskID := attempt.TaskID

	task := &Task{}
	err = b.db.C("tasks").FindId(taskID).One(task)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
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

	isLatestAttempt := task.CurrentAttempt == attempt.ID

	nextAttemptID := bson.NewObjectId()
	if !isLatestAttempt {
		nextAttemptID = task.CurrentAttempt
		log.Printf("NextAttemptForTask: %s is not the latest attempt\n", attempt.ID.Hex())
	}

	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"status":          status,
				"updated":         now.Unix(),
				"executed":        now.Unix(),
				"last_" + status:  now.Unix(),
				"at":              at,
				"active":          at > 0,
				"current_attempt": nextAttemptID,
				"attempt_queued":  false,
				"attempt_updated": time.Now().UnixNano(),
			},
			"$inc": bson.M{
				"executions":     1,
				"errors":         errors,
				"retry.attempts": retryAttempts,
			},
		},
		ReturnNew: true,
	}
	newTask := &Task{}
	_, err = b.db.C("tasks").FindId(taskID).Apply(change, newTask)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return nil, err
	}
	if err == nil && isLatestAttempt {
		nextAttempt, err = b.NewAttempt(newTask, true, false)
	}
	if err = b.AckAttempt(attempt.ID); err != nil {
		return nil, err
	}
	return
}

// ForceAttemptForTask ...
func (b *Base) ForceAttemptForTask(account bson.ObjectId, application string, name string) (attempt *Attempt, err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"name":        name,
		"deleted":     false,
	}
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"at":              time.Now().UnixNano(),
				"current_attempt": bson.NewObjectId(),
				"attempt_queued":  false,
				"attempt_updated": time.Now().UnixNano(),
			},
		},
	}
	var task *Task
	_, err = b.db.C("tasks").Find(query).Apply(change, task)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		attempt = nil
		if err == mgo.ErrNotFound {
			err = nil
		}
	} else if err == nil {
		attempt, err = b.NewAttempt(task, true, true)
		_, err = b.ShouldRefreshSession(err)
		if err != nil {
			attempt = nil
		}
	}
	return
}

// FixIntegrity ...
func (b *Base) FixIntegrity() error {
	ModelsTaskDebug("Fixing Tasks and Attempts integrity")
	// TODO: check indexes

	// Fixing Tasks
	query := bson.M{
		"active":          true,
		"deleted":         false,
		"attempt_queued":  false,
		"attempt_updated": bson.M{"$lte": time.Now().UnixNano() - 180*1000000000},
	}

	iter := b.db.C("tasks").Find(query).Iter()
	err := iter.Err()
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return err
	}
	task := &Task{}
	for iter.Next(task) {
		log.Printf("Fixing task %s\n", task.ID.Hex())
		var attempt *Attempt
		if task.CurrentAttempt.Valid() {
			attempt, err = b.GetAttempt(task.CurrentAttempt)
			if err != nil {
				log.Printf("Error getting attempt %s while fixing task %s: %s\n", task.CurrentAttempt.Hex(), task.Name, err)
				continue
			}
		} else {
			task.CurrentAttempt = bson.NewObjectId()
			update := bson.M{
				"$set": bson.M{
					"current_attempt": task.CurrentAttempt,
				},
			}
			err := b.db.C("tasks").UpdateId(task.ID, update)
			_, err = b.ShouldRefreshSession(err)
			if err != nil {
				continue
			}
		}
		if attempt == nil {
			b.NewAttempt(task, true, false)
		}
	}
	if err := iter.Close(); err != nil {
		_, err = b.ShouldRefreshSession(err)
		return err
	}

	// Fixing Attempts
	query = bson.M{
		"status":   bson.M{"$in": []string{"success", "error"}},
		"finished": bson.M{"$lte": time.Now().Unix() - 180},
		"acked":    false,
	}

	iter = b.db.C("attempts").Find(query).Iter()
	err = iter.Err()
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return err
	}
	attempt := &Attempt{}
	for iter.Next(attempt) {
		b.NextAttemptForTask(attempt)
	}
	if err := iter.Close(); err != nil {
		_, err = b.ShouldRefreshSession(err)
		return err
	}
	return nil
}

// EnsureTaskIndex creates mongo indexes for Application.
func (b *Base) EnsureTaskIndex() (err error) {
	index := mgo.Index{
		Key:        []string{"account", "application", "name"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	err = b.db.C("tasks").EnsureIndex(index)
	_, err = b.ShouldRefreshSession(err)
	return
}
