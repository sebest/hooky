package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/store"
)

// Scheduler schedules the Attempts of the Tasks.
type Scheduler struct {
	store         *store.Store
	wg            sync.WaitGroup
	quit          chan bool
	querierSem    chan bool
	workerSem     chan bool
	touchInterval int64
}

// New creates a new Scheduler.
func New(store *store.Store, maxQuerier int, maxWorker int, touchInterval int) *Scheduler {
	return &Scheduler{
		store:         store,
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
					db := s.store.DB()
					b := models.NewBase(db)
					attempt, err := b.NextAttempt(s.touchInterval * 2)
					db.Session.Close()
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
	db := s.store.DB()
	defer db.Session.Close()
	b := models.NewBase(db)
	// Start a goroutine to touch/reserve the Attempt.
	go func() {
		defer wg.Done()
		for {
			select {
			case attempt := <-result:
				if attempt == nil {
					return
				}
				_, err := b.NextAttemptForTask(attempt.TaskID, attempt.Status)
				if err != nil {
					fmt.Println(err)
				}
				return
			case <-time.After(time.Duration(s.touchInterval) * time.Second):
				b.TouchAttempt(attempt.ID, s.touchInterval*2)
			}
		}
	}()
	attempt, err := b.DoAttempt(attempt)
	if err != nil {
		fmt.Println(err)
	}
	result <- attempt
	wg.Wait()
}
