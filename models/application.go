package models

import (
	"errors"
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ErrDeleteDefaultApplication is returned when trying to delete the default application.
	ErrDeleteDefaultApplication = errors.New("can not delete default application")
	// ErrApplicationNotFound is returned when the application does not exist.
	ErrApplicationNotFound = errors.New("application does not exist")
)

// Application is a list of recurring Tasks.
type Application struct {
	// ID is the ID of the Application.
	ID bson.ObjectId `bson:"_id"`

	// Account is the ID of the Account owning the Application.
	Account bson.ObjectId `bson:"account"`

	// Name is the Application's name.
	Name string `bson:"name"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewApplication creates a new Application.
func (b *Base) NewApplication(account bson.ObjectId, name string) (application *Application, err error) {
	application = &Application{
		ID:      bson.NewObjectId(),
		Account: account,
		Name:    name,
	}
	err = b.db.C("applications").Insert(application)
	return
}

// GetApplication returns an Application.
func (b *Base) GetApplication(account bson.ObjectId, name string) (application *Application, err error) {
	query := bson.M{
		"account": account,
		"name":    name,
		"deleted": false,
	}
	application = &Application{}
	err = b.db.C("applications").Find(query).One(application)
	if err == mgo.ErrNotFound {
		err = nil
		application = nil
	}
	return
}

// DeleteApplication deletes an Application and all its children.
func (b *Base) DeleteApplication(account bson.ObjectId, name string) (err error) {
	if name == "default" {
		return ErrDeleteDefaultApplication
	}
	query := bson.M{
		"account": account,
		"name":    name,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("applications").UpdateAll(query, update); err == nil {
		query := bson.M{
			"account":     account,
			"application": name,
		}
		if _, err = b.db.C("queues").UpdateAll(query, update); err == nil {
			if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
				_, err = b.db.C("attempts").UpdateAll(query, update)
			}
		}
	}
	return
}

// DeleteApplications deletes all Applications owns by an Account.
func (b *Base) DeleteApplications(account bson.ObjectId) (err error) {
	query := bson.M{
		"account": account,
		"name":    bson.M{"$ne": "default"},
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	if _, err = b.db.C("applications").UpdateAll(query, update); err == nil {
		query = bson.M{
			"account": account,
		}
		if _, err = b.db.C("queues").UpdateAll(query, update); err == nil {
			if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
				_, err = b.db.C("attempts").UpdateAll(query, update)
			}
		}
	}
	return
}

// GetApplications returns a list of Applications.
func (b *Base) GetApplications(account bson.ObjectId, lp ListParams, lr *ListResult) (err error) {
	query := bson.M{
		"account": account,
		"deleted": false,
	}
	return b.getItems("applications", query, lp, lr)
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
