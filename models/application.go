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

// DeleteApplications deletes all Applications owns by an Account.
func (b *Base) DeleteApplications(account bson.ObjectId) (err error) {
	query := bson.M{
		"account": account,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("applications").UpdateAll(query, update); err == nil {
		if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
			_, err = b.db.C("attempts").UpdateAll(query, update)
		}
	}
	return
}

// DeleteApplication deletes an Application and all its children.
func (b *Base) DeleteApplication(account bson.ObjectId, application string) (err error) {
	query := bson.M{
		"account": account,
		"name":    application,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("applications").UpdateAll(query, update); err == nil {
		query := bson.M{
			"account":     account,
			"application": application,
		}
		if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
			_, err = b.db.C("attempts").UpdateAll(query, update)
		}
	}
	return
}

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
