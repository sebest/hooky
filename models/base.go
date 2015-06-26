package models

import (
	"fmt"

	"github.com/tj/go-debug"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ModelsBaseDebug ...
	ModelsBaseDebug = debug.Debug("hooky.models.base")
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

func (b *Base) EnsureIndex() {
	b.EnsureApplicationIndex()
	b.EnsureQueueIndex()
	b.EnsureTaskIndex()
	b.EnsureAttemptIndex()
	// b.Migrate()
}

// Migrate migrates database schema.
func (b *Base) Migrate() {
	fmt.Println("Migrate schema: start")
	queueRef := make(map[string]bson.ObjectId)
	queues := []Queue{}
	if err := b.db.C("queues").Find(bson.M{}).All(&queues); err != nil {
		fmt.Printf("Can't list queues: %s\n", err)
	}
	for _, queue := range queues {
		queueRef[queue.Name] = queue.ID
	}

	cols := []string{"tasks", "attempts"}
	query := bson.M{
		"queue_id": bson.M{"$exists": 0},
	}
	for qName, qID := range queueRef {
		query["queue"] = qName
		update := bson.M{"$set": bson.M{"queue_id": qID}}
		for _, col := range cols {
			c, err := b.db.C(col).UpdateAll(query, update)
			if err != nil {
				fmt.Printf("Error updating %s: %s\n", col, err)
			}
			fmt.Printf("Updated %d documents in %s\n", c.Updated, col)
		}
	}
	fmt.Println("Migrate schema: done")
}

// CleanDeletedRessources cleans ressources that have been deleted.
func (b *Base) CleanDeletedRessources() error {
	query := bson.M{
		"deleted": true,
	}
	if c, err := b.db.C("attempts").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted attempts", deleted)
	} else {
		return fmt.Errorf("failed to clean deleted attempts: %s", err)
	}
	if c, err := b.db.C("tasks").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted tasks", deleted)
	} else {
		return fmt.Errorf("failed to clean deleted tasks: %s", err)
	}
	if c, err := b.db.C("queues").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted queues", deleted)
	} else {
		return fmt.Errorf("failed to clean deleted queues: %s", err)
	}
	if c, err := b.db.C("applications").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted applications", deleted)
	} else {
		return fmt.Errorf("failed to clean deleted applications: %s", err)
	}
	if c, err := b.db.C("accounts").RemoveAll(query); err == nil {
		deleted := c.Removed
		ModelsBaseDebug("Cleaned %d deleted accounts", deleted)
	} else {
		return fmt.Errorf("failed to clean deleted accounts: %s", err)
	}
	return nil
}
