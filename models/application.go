package models

import (
	"errors"
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Application is a list of recurring Tasks.
type Application struct {
	// ID is the ID of the Application.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Application.
	Account bson.ObjectId `bson:"account"`

	// Name is the application's name.
	Name string `bson:"name"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewApplication creates a new Application.
func (b *Base) NewApplication(account bson.ObjectId, name string) (application *Application, err error) {
	if name == "default" {
		return nil, errors.New("The application name 'default' is reserved.")
	}
	application = &Application{
		ID:      bson.NewObjectId(),
		Account: account,
		Name:    name,
	}
	err = b.db.C("applications").Insert(application)
	return
}

// GetApplication returns a list of Tasks from a application.
// func (b *Base) GetApplication(application string) (tasks []*Task, err error) {
// 	query := bson.M{"application": application}
// 	err = b.db.C("tasks").Find(query).All(&tasks)
// 	return
// }

// DeleteApplication deletes the Tasks from a application.
// func (b *Base) DeleteApplication(application string) (err error) {
// 	query := bson.M{"application": application}
// 	_, err = b.db.C("tasks").RemoveAll(query)
// 	_, err = b.db.C("attempts").RemoveAll(query)
// 	return
// }

// EnsureApplicationIndex creates mongo indexes for Application.
func (b *Base) EnsureApplicationIndex() {
	index := mgo.Index{
		Key:        []string{"account", "name"},
		Unique:     true,
		Background: false,
		Sparse:     true,
	}
	if err := b.db.C("applications").EnsureIndex(index); err != nil {
		fmt.Printf("Error creating index on applications: %s\n", err)
	}
}
