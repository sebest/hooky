package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/sebest/hooky/models"
)

type Scheduler struct {
	tm         *models.TasksManager
	wg         sync.WaitGroup
	quit       chan bool
	querierSem chan bool
	workerSem  chan bool
}

func New(tm *models.TasksManager, maxQuerier int, maxWorker int) *Scheduler {
	return &Scheduler{
		tm:         tm,
		quit:       make(chan bool),
		querierSem: make(chan bool, maxQuerier),
		workerSem:  make(chan bool, maxWorker),
	}
}

func (s *Scheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

func (s *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-s.quit:
				return

			case s.querierSem <- true:
				go func() {
					s.workerSem <- true
					s.wg.Add(1)
					defer s.wg.Done()
					attempt, err := s.tm.Attempts.Next(10)
					if attempt != nil {
						s.wg.Add(1)
						go func() {
							defer s.wg.Done()
							s.worker(attempt)
							<-s.workerSem
						}()
						<-s.querierSem
						return
					} else if err != nil {
						fmt.Println(err)
					}
					<-s.workerSem
					time.Sleep(100 * time.Millisecond)
					<-s.querierSem
				}()
			}
		}
	}()
}

func (s *Scheduler) worker(attempt *models.Attempt) {
	result := make(chan *models.Attempt)
	defer close(result)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case attempt := <-result:
				if attempt == nil {
					return
				}
				_, err := s.tm.NextAttempt(attempt.TaskID, attempt.Status)
				if err != nil {
					fmt.Println(err)
				}
				return
			case <-time.After(5 * time.Second):
				s.tm.Attempts.Touch(attempt.ID, 10)
			}
		}
	}()
	attempt, err := s.tm.Attempts.Do(attempt)
	if err != nil {
		fmt.Println(err)
	}
	result <- attempt
	wg.Wait()
}
