package models

import (
	"expvar"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tj/go-debug"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	statsAttemptsSuccess = expvar.NewInt("attemptsSuccess")
	statsAttemptsError   = expvar.NewInt("attemptsError")
	// ModelsAttemptDebug ...
	ModelsAttemptDebug = debug.Debug("hooky.models.attempt")
)

// AttemptStatuses
var AttemptStatuses = map[string]bool{
	"pending": true,
	"success": true,
	"error":   true,
}

// Attempt describes a HTTP request that must be perform for a task.
type Attempt struct {
	// ID is the ID of the attempt.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Task.
	Account bson.ObjectId `bson:"account"`

	// Application is the name of the parent Application.
	Application string `bson:"application"`

	// Task is the task's name.
	Task string `bson:"task"`

	// TaskID is the ID of the parent Task of this attempt.
	TaskID bson.ObjectId `bson:"task_id"`

	// Queue is the name of the parent Queue.
	Queue string `bson:"queue"`

	// QueueID is the ID of the parent Queue
	QueueID bson.ObjectId `bson:"queue_id"`

	// URL is the URL that the worker with requests.
	URL string `bson:"url"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth HTTPAuth `json:"auth"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `bson:"method"`

	// Headers are the HTTP headers that will be used when executing the request.
	Headers map[string]string `bson:"headers"`

	// Payload is a arbitrary data that will be POSTed on the URL.
	Payload string `bson:"payload"`

	// Reserved is a Unix timestamp until when the attempt is reserved by a worker.
	Reserved int64 `bson:"reserved"`

	// At is a Unix timestamp representing the time a request must be performed.
	At int64 `bson:"at"`

	// Finished is a Unix timestamp representing the time the attempt finished.
	Finished int64 `bson:"finished,omitempty"`

	// Status is either `pending`, `running`, `success` or `error`
	Status string `bson:"status"`

	// StatusCode is the HTTP status code.
	StatusCode int32 `bson:"status_code,omitempty"`

	// StatusMessage is a human readable message related to the Status.
	StatusMessage string `bson:"status_message,omitempty"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewAttempt creates a new Attempt.
func (b *Base) NewAttempt(task *Task, deletePending bool, force bool) (*Attempt, error) {
	if deletePending {
		if _, err := b.DeletePendingAttempts(task.ID); err != nil {
			return nil, err
		}
	}
	if !force && (!task.Active || task.At == 0 || task.Deleted) {
		return nil, nil
	}
	attempt := &Attempt{
		ID:          bson.NewObjectId(),
		TaskID:      task.ID,
		Account:     task.Account,
		Application: task.Application,
		Queue:       task.Queue,
		QueueID:     task.QueueID,
		Task:        task.Name,
		URL:         task.URL,
		HTTPAuth:    task.HTTPAuth,
		Method:      task.Method,
		Headers:     task.Headers,
		Payload:     task.Payload,
		Reserved:    task.At,
		At:          task.At,
		Status:      "pending",
	}
	if err := b.db.C("attempts").Insert(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

// GetAttempt returns an Attempt.
func (b *Base) GetAttempt(attemptID bson.ObjectId) (*Attempt, error) {
	attempt := &Attempt{}
	if err := b.db.C("attempts").FindId(attemptID).One(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

// DeletePendingAttempts deletes all pending Attempts for a given Task ID.
func (b *Base) DeletePendingAttempts(taskID bson.ObjectId) (bool, error) {
	query := bson.M{
		"task_id": taskID,
		"status":  "pending",
		"deleted": false,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	c, err := b.db.C("attempts").UpdateAll(query, update)
	if err != nil {
		return false, err
	}
	return c.Updated > 0, nil
}

// GetAttempts returns a list of Attempts.
func (b *Base) GetAttempts(account bson.ObjectId, application string, task string, lp ListParams, lr *ListResult) (err error) {
	query := bson.M{
		"account":     account,
		"application": application,
		"task":        task,
		"deleted":     false,
	}
	if value, ok := lp.Filters["status"]; ok {
		_, ok := AttemptStatuses[value]
		if ok {
			query["status"] = value
		}
	}
	return b.getItems("attempts", query, lp, lr)
}

// ForceTaskAttempt ...
func (b *Base) ForceTaskAttempt(account bson.ObjectId, application string, name string) (attempt *Attempt, err error) {
	var task *Task
	task, err = b.GetTask(account, application, name)
	if err == nil {
		task.At = time.Now().UnixNano()
		attempt, err = b.NewAttempt(task, true, true)
	}
	return
}

// NextAttempt reserves and returns the next Attempt.
func (b *Base) NextAttempt(ttr int64) (*Attempt, error) {
	var fullQueues []bson.ObjectId
	now := time.Now().UnixNano()
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"reserved": now + (ttr * 1000000000),
				"status":   "running",
			},
		},
		ReturnNew: true,
	}
	query := bson.M{
		"status":   bson.M{"$in": []string{"pending", "running"}},
		"reserved": bson.M{"$lt": now},
		"deleted":  false,
	}
	for {
		if len(fullQueues) > 0 {
			query["queue_id"] = bson.M{"$nin": fullQueues}
		}
		attempt := &Attempt{}
		_, err := b.db.C("attempts").Find(query).Apply(change, attempt)
		if err == mgo.ErrNotFound {
			// ModelsAttemptDebug("No attempt found")
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		full, err := b.QueueFull(attempt.QueueID, attempt.ID)
		if err != nil {
			return nil, err
		}
		if !full {
			ModelsAttemptDebug("Found an attempt in queue %s", attempt.QueueID.Hex())
			return attempt, nil
		}
		ModelsAttemptDebug("Queue %s full", attempt.QueueID)
		fullQueues = append(fullQueues, attempt.QueueID)
	}
}

// DoAttempt executes the attempt.
func (b *Base) DoAttempt(attempt *Attempt) (*Attempt, error) {
	var status string
	var statusMessage string
	var statusCode int
	if strings.HasPrefix(attempt.URL, "test://") {
		ModelsAttemptDebug("Test attempt %s starting", attempt.URL)
		time.Sleep(10 * time.Second)
		status = "success"
		statusCode = 200
		statusMessage = "Test attempt"
		ModelsAttemptDebug("Test attempt %s done", attempt.URL)
	} else {
		ModelsAttemptDebug("Starting attempt [%s] for task %s", attempt.ID.Hex(), attempt.Task)
		var data io.Reader
		contentType := "text/plain"
		if attempt.Method == "POST" && attempt.Payload != "" {
			data = strings.NewReader(attempt.Payload)
			if attempt.Payload[0] == '{' {
				contentType = "application/json"
			}
		}
		req, err := http.NewRequest(attempt.Method, attempt.URL, data)
		if err != nil {
			return nil, err
		}
		req.Header.Add("User-Agent", "Hooky")
		req.Header.Add("X-Hooky-Account", attempt.Account.Hex())
		req.Header.Add("X-Hooky-Application", attempt.Application)
		req.Header.Add("X-Hooky-Queue", attempt.Queue)
		req.Header.Add("X-Hooky-Task-Name", attempt.Task)
		req.Header.Add("X-Hooky-Attempt-ID", attempt.ID.Hex())
		req.Header.Add("Content-Type", contentType)
		for k, v := range attempt.Headers {
			req.Header.Add(k, v)
		}
		if attempt.HTTPAuth.Username != "" || attempt.HTTPAuth.Password != "" {
			req.SetBasicAuth(attempt.HTTPAuth.Username, attempt.HTTPAuth.Password)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			status = "error"
			statusMessage = err.Error()
		} else {
			defer resp.Body.Close()
			statusMessage = resp.Status
			statusCode = resp.StatusCode
			if statusCode == 200 {
				status = "success"
			} else {
				status = "error"
			}
		}
		ModelsAttemptDebug("Attempt [%s] %s %s : %d -> %s", attempt.ID.Hex(), attempt.Method, attempt.URL, statusCode, status)
	}

	if err := b.FillQueue(attempt.QueueID, attempt.ID); err != nil {
		return nil, err
	}
	if status == "success" {
		statsAttemptsSuccess.Add(1)
	} else {
		statsAttemptsError.Add(1)
	}
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"finished":       time.Now().Unix(),
				"status":         status,
				"status_code":    statusCode,
				"status_message": statusMessage,
			},
		},
		ReturnNew: true,
	}
	_, err := b.db.C("attempts").FindId(attempt.ID).Apply(change, attempt)
	if err != nil {
		return nil, err
	}
	return attempt, nil
}

// TouchAttempt reserves an attemptsttempt for more time.
func (b *Base) TouchAttempt(attemptID bson.ObjectId, seconds int64) error {
	update := bson.M{
		"$set": bson.M{
			"reserved": time.Now().UnixNano() + (seconds * 1000000000),
		},
	}
	return b.db.C("attempts").UpdateId(attemptID, update)
}

// CleanFinishedAttempts cleans attempts that are finished since more than X seconds.
func (b *Base) CleanFinishedAttempts(seconds int64) (deleted int, err error) {
	query := bson.M{
		"finished": bson.M{"$lte": time.Now().Unix() - seconds},
	}
	c, err := b.db.C("attempts").RemoveAll(query)
	if err != nil {
		err = fmt.Errorf("failed cleaning finished attempts: %s", err)
	} else {
		deleted = c.Removed
		ModelsAttemptDebug("Cleaned %d finished attempts", deleted)
	}
	return
}

// EnsureAttemptIndex creates mongo indexes for Application.
func (b *Base) EnsureAttemptIndex() {
	index1 := mgo.Index{
		Key:        []string{"account", "application", "task"},
		Unique:     false,
		Background: false,
		Sparse:     true,
	}
	if err := b.db.C("attempts").EnsureIndex(index1); err != nil {
		fmt.Printf("Error creating index on attempts: %s\n", err)
	}
	index2 := mgo.Index{
		Key:        []string{"status", "reserved", "deleted"},
		Unique:     false,
		Background: true,
		Sparse:     true,
	}
	if err := b.db.C("attempts").EnsureIndex(index2); err != nil {
		fmt.Printf("Error creating index on attempts: %s\n", err)
	}
}

// GetAttemptByID returns a Attempt given its ID.
func (b *Base) GetAttemptByID(attemptID bson.ObjectId) (attempt *Attempt, err error) {
	attempt = &Attempt{}
	err = b.db.C("attempts").FindId(attemptID).One(attempt)
	if err == mgo.ErrNotFound {
		err = nil
		attempt = nil
	}
	return
}
