package sqs

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

type state struct {
	shutdownCh chan struct{}
	shutdownFn sync.Once

	sync.Mutex
	running bool

	runLock sync.RWMutex
}

func newState() *state {
	s := &state{
		shutdownCh: make(chan struct{}),
	}
	s.runLock.Lock()
	return s
}

func (s *state) start() {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		s.running = true
		s.runLock.Unlock()
	}
}
func (s *state) pause() {
	s.Lock()
	defer s.Unlock()

	if s.running {
		s.running = false
		s.runLock.Lock()
	}
}
func (s *state) shutdown() {
	s.shutdownFn.Do(func() {
		s.start()
		close(s.shutdownCh)
	})
}

func (s *state) run(wg *sync.WaitGroup, id int) {
	defer wg.Done()

	for i := 0; ; i++ {
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		s.runLock.RLock()
		fmt.Printf("%d: %d\n", id, i)
		s.runLock.RUnlock()

		time.Sleep(100 * time.Millisecond)
	}
}

// TestLocking ensures that the idea behind our runlock is correct.
func TestLocking(t *testing.T) {
	s := newState()

	var wg sync.WaitGroup
	wg.Add(2)
	go s.run(&wg, 1)
	go s.run(&wg, 2)

	log.Println("Initialized")
	time.Sleep(time.Second)

	log.Println("Starting")
	s.start()
	s.start()
	time.Sleep(time.Second)

	log.Println("Pausing")
	s.pause()
	s.pause()
	time.Sleep(time.Second)

	log.Println("Starting")
	s.start()
	s.start()
	time.Sleep(time.Second)

	// pause again to ensure we can go from paused -> start correctly
	s.pause()
	time.Sleep(time.Second)

	log.Println("Shutdown")
	s.shutdown()
	s.shutdown()

	wg.Wait()
}
