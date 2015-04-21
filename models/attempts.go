package models

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sebest/hooky/store"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Attempt describes a HTTP request that must be perform for a Webtask.
type Attempt struct {
	// ID is the ID of the attempt.
	ID bson.ObjectId `bson:"_id"`

	// URL is the URL that the worker with requests.
	URL string `bson:"url"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `bson:"method"`

	// Headers are the HTTP headers that will be used when executing the request.
	Headers map[string]string `bson:"headers"`

	// Payload is a arbitrary data that will be POSTed on the URL.
	Payload string `bson:"payload"`

	// taskID is the ID of the parent Webtask of this attempt.
	TaskID bson.ObjectId `bson:"task_id"`

	// Reserved is a Unix timestamp until when the attempt is reserved by a worker.
	Reserved int64 `bson:"reserved"`

	// Status is either `pending`, `success` or `error`
	Status string `bson:"status"`

	// StatusCode is the HTTP status code.
	StatusCode int32 `bson:"status_code,omitempty"`

	// StatusMessage is a human readable message related to the Status.
	StatusMessage string `bson:"status_message,omitempty"`
}

// AttemptsManager manages the Tasks Attempts.
type AttemptsManager struct {
	store  *store.Store
	client *http.Client
}

// NewAttemptsManager returns a new NewAttemptsManager.
func NewAttemptsManager(store *store.Store) *AttemptsManager {
	return &AttemptsManager{
		store:  store,
		client: &http.Client{},
	}
}

func NewAttempt(store *store.Store, task *Task) (*Attempt, error) {
	attempt := &Attempt{
		ID:       bson.NewObjectId(),
		TaskID:   task.ID,
		URL:      task.URL,
		Method:   task.Method,
		Headers:  task.Headers,
		Payload:  task.Payload,
		Reserved: task.At,
		Status:   "pending",
	}
	db := store.DB()
	defer db.Session.Close()
	if err := db.C("attempts").Insert(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

// New creates a new Attempt.
func (am *AttemptsManager) New(task *Task) (*Attempt, error) {
	return NewAttempt(am.store, task)
}

// Do executes the attempt.
func (am *AttemptsManager) Do(attempt *Attempt) (*Attempt, error) {
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
	req.Header.Add("X-Hooky-Task-ID", attempt.TaskID.Hex())
	req.Header.Add("X-Hooky-Attempt-ID", attempt.ID.Hex())
	req.Header.Add("Content-Type", contentType)
	for k, v := range attempt.Headers {
		req.Header.Add(k, v)
	}
	var status string
	var statusMessage string
	var statusCode int
	resp, err := am.client.Do(req)
	if err != nil {
		status = "error"
		statusMessage = err.Error()
	} else {
		statusMessage = resp.Status
		statusCode = resp.StatusCode
		if statusCode == 200 {
			status = "success"
		} else {
			status = "error"
		}
	}
	db := am.store.DB()
	defer db.Session.Close()
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
	_, err = db.C("attempts").FindId(attempt.ID).Apply(change, attempt)
	if err != nil {
		return nil, err
	}
	return attempt, nil
}

// Touch reserves an attemptsttempt for more time.
func (am *AttemptsManager) Touch(attemptID bson.ObjectId, seconds int64) error {
	update := bson.M{"$set": bson.M{"reserved": time.Now().UnixNano() + (seconds * 1000000000)}}
	db := am.store.DB()
	defer db.Session.Close()
	return db.C("attempts").UpdateId(attemptID, update)
}

// Get returns an Attempt.
func (am *AttemptsManager) Get(attemptID bson.ObjectId) (*Attempt, error) {
	db := am.store.DB()
	defer db.Session.Close()
	attempt := &Attempt{}
	if err := db.C("attempts").FindId(attemptID).One(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

// Next reserves and returns the next Attempt.
func (am *AttemptsManager) Next(ttr int64) (*Attempt, error) {
	now := time.Now().UnixNano()
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"reserved": now + (ttr * 1000000000)}},
		ReturnNew: true,
	}
	query := bson.M{"status": "pending", "reserved": bson.M{"$lt": now}}
	attempt := &Attempt{}
	db := am.store.DB()
	defer db.Session.Close()
	_, err := db.C("attempts").Find(query).Apply(change, attempt)
	if err == mgo.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return attempt, nil
}
