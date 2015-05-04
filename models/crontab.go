package models

import (
	"errors"
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Crontab is a list of recurring Tasks.
type Crontab struct {
	// ID is the ID of the Crontab.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Crontab.
	Account bson.ObjectId `bson:"account"`

	// Name is the crontab's name.
	Name string `bson:"name"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewCrontab creates a new Crontab.
func (b *Base) NewCrontab(account bson.ObjectId, name string) (crontab *Crontab, err error) {
	if name == "default" {
		return nil, errors.New("The crontab name 'default' is reserved.")
	}
	crontab = &Crontab{
		ID:      bson.NewObjectId(),
		Account: account,
		Name:    name,
	}
	err = b.db.C("crontabs").Insert(crontab)
	return
}

// GetCrontab returns a list of Tasks from a crontab.
// func (b *Base) GetCrontab(crontab string) (tasks []*Task, err error) {
// 	query := bson.M{"crontab": crontab}
// 	err = b.db.C("tasks").Find(query).All(&tasks)
// 	return
// }

// DeleteCrontab deletes the Tasks from a crontab.
// func (b *Base) DeleteCrontab(crontab string) (err error) {
// 	query := bson.M{"crontab": crontab}
// 	_, err = b.db.C("tasks").RemoveAll(query)
// 	_, err = b.db.C("attempts").RemoveAll(query)
// 	return
// }

// EnsureCrontabIndex creates mongo indexes for Crontab.
func (b *Base) EnsureCrontabIndex() {
	index := mgo.Index{
		Key:        []string{"account", "name"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	if err := b.db.C("crontabs").EnsureIndex(index); err != nil {
		fmt.Printf("Error creating index on crontabs: %s\n", err)
	}
}
