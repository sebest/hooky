package models

import "gopkg.in/mgo.v2"

type ListParams struct {
	Filters map[string]string
	Sort    map[string]string
	Page    int
	Limit   int
}

type Base struct {
	db *mgo.Database
}

func NewBase(db *mgo.Database) *Base {
	return &Base{
		db: db,
	}
}

func (b *Base) EnsureIndex() {
	b.EnsureApplicationIndex()
	b.EnsureQueueIndex()
	b.EnsureTaskIndex()
}
