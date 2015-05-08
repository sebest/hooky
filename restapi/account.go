package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// Account is an account to access the service.
type Account struct {
	// ID is the ID of the Account.
	ID string `json:"id"`

	// Created is the date when the Account was created.
	Created string `json:"created"`

	// Key is the secret key to authenticate the Account ID.
	Key string `json:"key"`
}

func accountParams(r *rest.Request) (bson.ObjectId, error) {
	// TODO handle errors
	accountID := bson.ObjectIdHex(r.PathParam("account"))
	return accountID, nil
}

// NewAccountFromModel returns an API Account given a model Account.
func NewAccountFromModel(account *models.Account) *Account {
	return &Account{
		ID:      account.ID.Hex(),
		Created: account.ID.Time().UTC().Format(time.RFC3339),
		Key:     account.Key,
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
	_, err = b.NewApplication(account.ID, "default")
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = b.NewQueue(account.ID, "default", "default")
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewAccountFromModel(account))
}

// GetAccount handles GET request on /accounts/:account
func GetAccount(w rest.ResponseWriter, r *rest.Request) {
	accountID, err := accountParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: do not return key.
	b := GetBase(r)
	account, err := b.GetAccount(accountID)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if account == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewAccountFromModel(account))
}

// GetAccounts ...
func GetAccounts(w rest.ResponseWriter, r *rest.Request) {
	b := GetBase(r)
	lp := parseListQuery(r)
	var accounts []*models.Account
	lr := &models.ListResult{
		List: &accounts,
	}

	if err := b.GetAccounts(lp, lr); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lr.Count == 0 {
		rest.NotFound(w, r)
		return
	}
	rt := make([]*Account, len(accounts))
	for idx, account := range accounts {
		rt[idx] = NewAccountFromModel(account)
	}
	w.WriteJson(models.ListResult{
		List:    rt,
		HasMore: lr.HasMore,
		Total:   lr.Total,
		Count:   lr.Count,
		Page:    lr.Page,
		Pages:   lr.Pages,
	})
}

// DeleteAccount handles DELETE request on /accounts/:account
func DeleteAccount(w rest.ResponseWriter, r *rest.Request) {
	accountID, err := accountParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	if err := b.DeleteAccount(accountID); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
