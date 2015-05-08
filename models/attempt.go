package models

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

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

	// TaskID is the ID of the parent Webtask of this attempt.
	TaskID bson.ObjectId `bson:"task_id"`

	// Queue is the name of the parent Queue.
	Queue string `bson:"queue"`

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

	// Status is either `pending`, `success` or `error`
	Status string `bson:"status"`

	// StatusCode is the HTTP status code.
	StatusCode int32 `bson:"status_code,omitempty"`

	// StatusMessage is a human readable message related to the Status.
	StatusMessage string `bson:"status_message,omitempty"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewAttempt creates a new Attempt.
func (b *Base) NewAttempt(task *Task) (*Attempt, error) {
	attempt := &Attempt{
		ID:          bson.NewObjectId(),
		TaskID:      task.ID,
		Account:     task.Account,
		Application: task.Application,
		Queue:       task.Queue,
		Task:        task.Name,
		URL:         task.URL,
		HTTPAuth:    task.HTTPAuth,
		Method:      task.Method,
		Headers:     task.Headers,
		Payload:     task.Payload,
		Reserved:    task.At,
		Status:      "pending",
	}
	if err := b.db.C("attempts").Insert(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

// DoAttempt executes the attempt.
func (b *Base) DoAttempt(attempt *Attempt) (*Attempt, error) {
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
	var status string
	var statusMessage string
	var statusCode int
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
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"status":         status,
				"status_code":    statusCode,
				"status_message": statusMessage,
			},
		},
		ReturnNew: true,
	}
	_, err = b.db.C("attempts").FindId(attempt.ID).Apply(change, attempt)
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

// GetAttempt returns an Attempt.
func (b *Base) GetAttempt(attemptID bson.ObjectId) (*Attempt, error) {
	attempt := &Attempt{}
	if err := b.db.C("attempts").FindId(attemptID).One(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
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

// NextAttempt reserves and returns the next Attempt.
func (b *Base) NextAttempt(ttr int64) (*Attempt, error) {
	now := time.Now().UnixNano()
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"reserved": now + (ttr * 1000000000),
			},
		},
		ReturnNew: true,
	}
	query := bson.M{
		"status":   "pending",
		"reserved": bson.M{"$lt": now},
		"deleted":  false,
	}
	attempt := &Attempt{}
	_, err := b.db.C("attempts").Find(query).Apply(change, attempt)
	if err == mgo.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return attempt, nil
}

// DeletePendingAttempts deletes all pending Attempts for a given Task ID.
func (b *Base) DeletePendingAttempts(taskID bson.ObjectId) error {
	query := bson.M{
		"task_id": taskID,
		"status":  "pending",
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	_, err := b.db.C("attempts").UpdateAll(query, update)
	return err
}

// DeleteAllAttempts deletes all pending Attempts for a given Task ID.
func (b *Base) DeleteAllAttempts(taskID bson.ObjectId) error {
	query := bson.M{
		"task_id": taskID,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	_, err := b.db.C("attempts").UpdateAll(query, update)
	return err
}

// EnsureAttemptIndex creates mongo indexes for Application.
func (b *Base) EnsureAttemptIndex() {
	index := mgo.Index{
		Key:        []string{"account", "application", "task"},
		Unique:     false,
		Background: false,
		Sparse:     true,
	}
	if err := b.db.C("attempts").EnsureIndex(index); err != nil {
		fmt.Printf("Error creating index on attempts: %s\n", err)
	}
}
