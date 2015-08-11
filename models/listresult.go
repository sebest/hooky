package models

import (
	"math"
	"reflect"

	"gopkg.in/mgo.v2/bson"
)

// ListResult is the structure used for listing collections.
type ListResult struct {
	List    interface{} `json:"list"`
	HasMore bool        `json:"hasMore"`
	Total   int         `json:"total"`
	Count   int         `json:"count"`
	Page    int         `json:"page"`
	Pages   int         `json:"pages"`
}

func (b *Base) getItems(collection string, query bson.M, lp ListParams, lr *ListResult) (err error) {
	skip := lp.Limit * (lp.Page - 1)
	if lr.Total, err = b.db.C(collection).Find(query).Count(); err != nil {
		return
	}
	lr.Page = lp.Page
	lr.Pages = int(math.Ceil(float64(lr.Total) / float64(lp.Limit)))
	if lr.Page > lr.Pages {
		lr.Page = lr.Pages
	}
	if skip < lr.Total {
		if err = b.db.C(collection).Find(query).Sort("-_id").Skip(skip).Limit(lp.Limit).All(lr.List); err != nil {
			return
		}
		lr.Count = reflect.ValueOf(lr.List).Elem().Len()
		lr.HasMore = lr.Total > lr.Count+skip
	}
	return
}
