package models

import (
	"math/rand"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Account is an account to access the service.
type Account struct {
	// ID is the ID of the Account.
	ID bson.ObjectId `bson:"_id"`

	// Name is display name for the Account.
	Name *string `bson:"name,omitempty"`

	// Key is the secret key to authenticate the Account ID.
	Key string `bson:"key"`

	// Deleted
	Deleted bool `bson:"deleted"`
}

// NewAccount creates a new Account.
func (b *Base) NewAccount(name *string) (account *Account, err error) {
	account = &Account{
		ID:   bson.NewObjectId(),
		Name: name,
		Key:  randKey(32),
	}
	err = b.db.C("accounts").Insert(account)
	return
}

// UpdateAccount updates an Account.
func (b *Base) UpdateAccount(accountID bson.ObjectId, name *string) (account *Account, err error) {
	if name == nil {
		return b.GetAccount(accountID)
	}
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"name": name,
			},
		},
		ReturnNew: true,
	}
	query := bson.M{
		"_id": accountID,
	}
	_, err = b.db.C("accounts").Find(query).Apply(change, account)
	return
}

// GetAccount returns an Account given its ID.
func (b *Base) GetAccount(accountID bson.ObjectId) (account *Account, err error) {
	query := bson.M{
		"_id":     accountID,
		"deleted": false,
	}
	account = &Account{}
	err = b.db.C("accounts").Find(query).One(account)
	if err == mgo.ErrNotFound {
		err = nil
		account = nil
	}
	return
}

// DeleteAccount deletes an Account given its ID.
func (b *Base) DeleteAccount(account bson.ObjectId) (err error) {
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
		},
	}
	err = b.db.C("accounts").UpdateId(account, update)
	if err == nil {
		query := bson.M{
			"account": account,
		}
		if _, err = b.db.C("applications").UpdateAll(query, update); err == nil {
			if _, err = b.db.C("queues").UpdateAll(query, update); err == nil {
				if _, err = b.db.C("tasks").UpdateAll(query, update); err == nil {
					_, err = b.db.C("attempts").UpdateAll(query, update)
				}
			}
		}
	}
	return
}

// GetAccounts returns a list of Accounts.
func (b *Base) GetAccounts(lp ListParams, lr *ListResult) (err error) {
	query := bson.M{
		"deleted": false,
	}
	return b.getItems("accounts", query, lp, lr)
}

//AuthenticateAccount authenticates an Account.
func (b *Base) AuthenticateAccount(account bson.ObjectId, key string) (bool, error) {
	query := bson.M{
		"_id":     account,
		"key":     key,
		"deleted": false,
	}
	n, err := b.db.C("accounts").Find(query).Count()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randKey(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
