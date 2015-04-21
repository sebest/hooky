package models

import "github.com/sebest/hooky/store"

type Manager struct {
	Tasks    *TasksManager
	Attempts *AttemptsManager
}

func NewManager(s *store.Store) *Manager {
	return &Manager{
		Tasks:    NewTasksManager(s),
		Attempts: NewAttemptsManager(s),
	}
}
