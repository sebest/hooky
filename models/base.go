package models

import "gopkg.in/mgo.v2"

type Base struct {
	db *mgo.Database
}

func NewBase(db *mgo.Database) *Base {
	return &Base{
		db: db,
	}
}

func (b *Base) EnsureIndex() {
	b.EnsureCrontabIndex()
	b.EnsureTaskIndex()
}
