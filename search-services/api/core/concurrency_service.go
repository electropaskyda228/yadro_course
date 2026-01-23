package core

import (
	"sync/atomic"
)

type ConcurrencyHandler struct {
	limit     int
	semaphore chan struct{}
	stopped   atomic.Bool
}

func NewConcurrencyHandler(limit int) (*ConcurrencyHandler, error) {
	if limit <= 0 {
		limit = 1
	}
	return &ConcurrencyHandler{
		limit:     limit,
		semaphore: make(chan struct{}, limit),
	}, nil
}

func (ch *ConcurrencyHandler) Start() error {
	ch.stopped.Store(false)
	return nil
}

func (ch *ConcurrencyHandler) Stop() {
	ch.stopped.Store(true)
	ch.Wait()
}

func (ch *ConcurrencyHandler) Wait() {
	for i := 0; i < ch.limit; i++ {
		ch.semaphore <- struct{}{}
	}
	for i := 0; i < ch.limit; i++ {
		<-ch.semaphore
	}
}

func (ch *ConcurrencyHandler) Submit(f func()) string {
	if ch.stopped.Load() {
		return StatusRejected
	}

	if f == nil {
		return StatusRejected
	}

	select {
	case ch.semaphore <- struct{}{}:
		go func() {
			f()
			<-ch.semaphore
		}()
		return StatusAccepted
	default:
		return StatusRejected
	}
}
