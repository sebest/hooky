package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/sebest/hooky/models"
)

// Scheduler schedules the Attempts of the Tasks.
type Scheduler struct {
	tm            *models.TasksManager
	wg            sync.WaitGroup
	quit          chan bool
	querierSem    chan bool
	workerSem     chan bool
	touchInterval int64
}

// New creates a new Scheduler.
func New(tm *models.TasksManager, maxQuerier int, maxWorker int, touchInterval int) *Scheduler {
	return &Scheduler{
		tm:            tm,
		quit:          make(chan bool),
		querierSem:    make(chan bool, maxQuerier),
		workerSem:     make(chan bool, maxWorker),
		touchInterval: int64(touchInterval),
	}
}

// Stop stops the Scheduler.
func (s *Scheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

// Start starts the Scheduler.
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
					attempt, err := s.tm.Attempts.Next(s.touchInterval * 2)
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

// worker executes the Attempts.
func (s *Scheduler) worker(attempt *models.Attempt) {
	result := make(chan *models.Attempt)
	defer close(result)
	var wg sync.WaitGroup
	wg.Add(1)
	// Start a goroutine to touch/reserve the Attempt.
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
			case <-time.After(time.Duration(s.touchInterval) * time.Second):
				s.tm.Attempts.Touch(attempt.ID, s.touchInterval*2)
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
