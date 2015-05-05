package store

import (
	"time"

	"gopkg.in/mgo.v2"
)

type Store struct {
	session *mgo.Session
}

func New(url string) (*Store, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	session.SetSyncTimeout(10 * time.Second)
	session.SetSocketTimeout(20 * time.Second)
	session.SetSafe(&mgo.Safe{})
	return &Store{
		session: session,
	}, nil
}

func (s *Store) DB() *mgo.Database {
	return s.session.Copy().DB("")
}
