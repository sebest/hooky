package models

import (
	"math/rand"

	"gopkg.in/mgo.v2/bson"
)

// Account is an account to access the service.
type Account struct {
	// ID is the ID of the Account.
	ID bson.ObjectId `bson:"_id"`

	// Key is the secret key to authenticate the Account ID.
	Key string `bson:"key"`
}

// NewAccount creates a new Account.
func (b *Base) NewAccount() (account *Account, err error) {
	account = &Account{
		ID:  bson.NewObjectId(),
		Key: randKey(32),
	}
	err = b.db.C("accounts").Insert(account)
	return
}

//AuthenticateAccount authenticates an Account.
func (b *Base) AuthenticateAccount(account bson.ObjectId, key string) (bool, error) {
	query := bson.M{"_id": account, "key": key}
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
