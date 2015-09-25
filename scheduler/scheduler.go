package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/store"
)

// Scheduler schedules the Attempts of the Tasks.
type Scheduler struct {
	store                 *store.Store
	wg                    sync.WaitGroup
	quit                  chan bool
	querierSem            chan bool
	workerSem             chan bool
	touchInterval         int64
	cleanFinishedAttempts int64
}

// New creates a new Scheduler.
func New(store *store.Store, maxQuerier int, maxWorker int, touchInterval int, cleanFinishedAttempts int) *Scheduler {
	return &Scheduler{
		store:                 store,
		quit:                  make(chan bool),
		querierSem:            make(chan bool, maxQuerier),
		workerSem:             make(chan bool, maxWorker),
		touchInterval:         int64(touchInterval),
		cleanFinishedAttempts: int64(cleanFinishedAttempts),
	}
}

// Stop stops the Scheduler.
func (s *Scheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

// Start starts the Scheduler.
func (s *Scheduler) Start() {
	// Cleaner
	go func() {
		clean := func() {
			db := s.store.DB()
			defer db.Session.Close()
			b := models.NewBase(db)
			if _, err := b.CleanFinishedAttempts(s.cleanFinishedAttempts); err != nil && err != models.ErrDatabase {
				log.Printf("Scheduler error with CleanFinishedAttempts: %s\n", err)
			}
			if err := b.CleanDeletedRessources(); err != nil && err != models.ErrDatabase {
				log.Printf("Scheduler error with CleanDeletedRessources: %s\n", err)
			}
		}
		clean()
		for {
			select {
			case <-s.quit:
				return
			case <-time.After(time.Second * 60):
				clean()
			}
		}
	}()

	// Fixer
	go func() {
		fix := func() {
			db := s.store.DB()
			defer db.Session.Close()
			b := models.NewBase(db)
			if err := b.FixIntegrity(); err != nil && err != models.ErrDatabase {
				log.Printf("Scheduler error with FixIntegrity: %s\n", err)
			}
			if err := b.FixQueues(); err != nil && err != models.ErrDatabase {
				log.Printf("Scheduler error with FixQueues: %s\n", err)
			}
		}
		fix()
		for {
			select {
			case <-s.quit:
				return
			case <-time.After(time.Second * 60):
				fix()
			}
		}
	}()

	// Attempts scheduler
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
					} else if err != nil && err != models.ErrDatabase {
						log.Printf("Scheduler error with NextAttempt: %#v\n", err)
					}
					<-s.workerSem
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
	go func(attempt *models.Attempt) {
		defer wg.Done()
		for {
			select {
			case attempt := <-result:
				if attempt == nil {
					return
				}
				_, err := b.NextAttemptForTask(attempt)
				if err != nil && err != models.ErrDatabase {
					log.Printf("Scheduler error with NextAttemptForTask: %#v\n", err)
				}
				return
			case <-time.After(time.Duration(s.touchInterval) * time.Second):
				b.TouchAttempt(attempt.ID, s.touchInterval*2)
			}
		}
	}(attempt)
	err := b.DoAttempt(attempt)
	if err == nil {
		result <- attempt
	} else {
		if err != models.ErrDatabase {
			log.Printf("Scheduler error with DoAttempt: %#v\n", err)
		}
		result <- nil
	}
	wg.Wait()
}
