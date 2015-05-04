package restapi

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
)

// Account is an account to access the service.
type Account struct {
	// ID is the ID of the Account.
	ID string `json:"id"`

	// Key is the secret key to authenticate the Account ID.
	Key string `json:"key"`
}

// NewAccountFromModel returns an API Account given a model Account.
func NewAccountFromModel(account *models.Account) *Account {
	return &Account{
		ID:  account.ID.Hex(),
		Key: account.Key,
	}
}

// PostAccount handles POST requests on /accounts
func PostAccount(w rest.ResponseWriter, r *rest.Request) {
	b := GetBase(r)
	account, err := b.NewAccount()
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewAccountFromModel(account))
}
