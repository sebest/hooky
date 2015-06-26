package models

import (
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
