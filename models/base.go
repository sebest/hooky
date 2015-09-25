package models

import (
	"fmt"
	"io"
	"net"

	"github.com/tj/go-debug"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ModelsBaseDebug ...
	ModelsBaseDebug = debug.Debug("hooky.models.base")

	ErrDatabase = fmt.Errorf("Database Error")
)

type ListParams struct {
	Fields  []string
	Filters map[string]string
	Sort    map[string]string
	Page    int
	Limit   int
}

type Base struct {
	db *mgo.Database
}

func NewBase(db *mgo.Database) *Base {
	return &Base{
		db: db,
	}
}

// ShouldRefreshSession checks if we should refresh the mongo session
func (b *Base) ShouldRefreshSession(err error) (bool, error) {
	if err == io.EOF {
		b.db.Session.Refresh()
		return true, ErrDatabase
	}
	opError, ok := err.(*net.OpError)
	if ok && opError.Op == "read" {
		b.db.Session.Refresh()
		return true, ErrDatabase
	}
	queryError, ok := err.(*mgo.QueryError)
	if ok && queryError.Message == "not master" {
		b.db.Session.Refresh()
		return true, ErrDatabase
	}
	return false, err
}

func (b *Base) Bootstrap() error {
	if err := b.EnsureApplicationIndex(); err != nil {
		return err
	}
	if err := b.EnsureQueueIndex(); err != nil {
		return err
	}
	if err := b.EnsureTaskIndex(); err != nil {
		return err
	}
	if err := b.EnsureAttemptIndex(); err != nil {
		return err
	}
	if err := b.migrate(); err != nil {
		return err
	}
	return nil
}

// Migrate migrates database schema.
func (b *Base) migrate() error {
	query := bson.M{
		"active":         true,
		"deleted":        false,
		"schedule":       bson.M{"$ne": ""},
		"attempt_queued": bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"attempt_queued":  false,
			"attempt_updated": 0,
		},
	}
	_, err := b.db.C("tasks").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return err
	}

	query = bson.M{
		"acked":  bson.M{"$exists": false},
		"status": bson.M{"$in": []string{"success", "error"}},
	}
	update = bson.M{
		"$set": bson.M{
			"acked": true,
		},
	}
	_, err = b.db.C("attempts").UpdateAll(query, update)
	_, err = b.ShouldRefreshSession(err)
	if err != nil {
		return err
	}
	return nil
}

// CleanDeletedRessources cleans ressources that have been deleted.
func (b *Base) CleanDeletedRessources() error {
	query := bson.M{
		"deleted": true,
	}
	iter := b.db.C("attempts").Find(query).Iter()
	if err := iter.Err(); err != nil {
		_, err = b.ShouldRefreshSession(err)
		return err
	}

	deleted := 0
	attempt := &Attempt{}
	for iter.Next(attempt) {
		b.DeQueue(attempt.QueueID, attempt.ID)
		err := b.db.C("attempts").RemoveId(attempt.ID)
		refreshed, err := b.ShouldRefreshSession(err)
		if refreshed {
			continue
		} else if err != nil {
			ModelsBaseDebug("Cleaned %d deleted attempts", deleted)
			return err
		}
		deleted++
	}
	ModelsBaseDebug("Cleaned %d deleted attempts", deleted)
	if err := iter.Close(); err != nil {
		_, err = b.ShouldRefreshSession(err)
		return err
	}

	if c, err := b.db.C("tasks").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted tasks", deleted)
	} else {
		_, err = b.ShouldRefreshSession(err)
		return err
	}
	if c, err := b.db.C("queues").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted queues", deleted)
	} else {
		_, err = b.ShouldRefreshSession(err)
		return err
	}
	if c, err := b.db.C("applications").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted applications", deleted)
	} else {
		_, err = b.ShouldRefreshSession(err)
		return err
	}
	if c, err := b.db.C("accounts").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted accounts", deleted)
	} else {
		_, err = b.ShouldRefreshSession(err)
		return err
	}
	return nil
}
