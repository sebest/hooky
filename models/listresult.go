package models

import (
	"reflect"

	"gopkg.in/mgo.v2/bson"
)

type ListResult struct {
	List    interface{} `json:"list"`
	HasMore bool        `json:"hasMore"`
	Total   int         `json:"total"`
	Count   int         `json:"count"`
	Page    int         `json:"page"`
	Pages   int         `json:"pages"`
}

func (b *Base) getItems(collection string, query bson.M, lp ListParams, lr *ListResult) (err error) {
	limit := lp.Limit
	skip := lp.Limit * (lp.Page - 1)
	lr.Page = (skip / limit) + 1
	if lr.Total, err = b.db.C(collection).Find(query).Count(); err != nil {
		return
	}
	lr.Pages = (lr.Total / limit) + 1
	if skip < lr.Total {
		if err = b.db.C(collection).Find(query).Skip(skip).Limit(limit).All(lr.List); err != nil {
			return
		}
		lr.Count = reflect.ValueOf(lr.List).Elem().Len()
		lr.HasMore = lr.Total > lr.Count+skip
	}
	return
}
