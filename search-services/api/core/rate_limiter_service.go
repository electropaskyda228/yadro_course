package core

import (
	"context"
	"sync"
	"time"
)

type RateLimiterHandler struct {
	rate    int
	tokens  chan time.Time
	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func NewRateLimiterHandler(rate int) (*RateLimiterHandler, error) {
	if rate <= 0 {
		rate = 1
	}

	rl := &RateLimiterHandler{
		rate:    rate,
		tokens:  make(chan time.Time, 1),
		stopCh:  make(chan struct{}),
		stopped: true,
	}
	rl.tokens <- time.Now()
	return rl, nil
}

func (rl *RateLimiterHandler) tokenGenerator() {
	defer rl.wg.Done()

	ticker := time.NewTicker(time.Duration(float64(time.Second) / float64(rl.rate)))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case rl.tokens <- time.Now():
			default:
			}
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *RateLimiterHandler) Wait(ctx context.Context) error {
	rl.mu.Lock()
	if rl.stopped {
		rl.mu.Unlock()
		return nil
	}
	rl.mu.Unlock()

	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.stopCh:
		return nil
	}
}

func (rl *RateLimiterHandler) Submit(f func()) string {
	if f == nil {
		return StatusRejected // Не принимаем nil функции
	}

	if err := rl.Wait(context.Background()); err != nil {
		return StatusRejected
	}

	f()
	return StatusAccepted
}

func (rl *RateLimiterHandler) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.stopped {
		return
	}
	rl.stopped = true

	close(rl.stopCh)
	rl.wg.Wait()
}

func (rl *RateLimiterHandler) Start() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.stopped {
		return nil
	}

	// Создаем новый stopCh для повторного использования
	rl.stopCh = make(chan struct{})
	rl.stopped = false

	rl.wg.Add(1)
	go rl.tokenGenerator()

	return nil
}
