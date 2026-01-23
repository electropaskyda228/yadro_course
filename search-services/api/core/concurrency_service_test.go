package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyHandler_Basic(t *testing.T) {
	handler, err := NewConcurrencyHandler(1)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	done := make(chan bool)
	status := handler.Submit(func() {
		done <- true
	})
	assert.Equal(t, StatusAccepted, status)

	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Задача не выполнилась")
	}

	handler.Wait()
	handler.Stop()
}

func TestConcurrencyHandler_Limit(t *testing.T) {
	handler, err := NewConcurrencyHandler(2)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	var running atomic.Int32
	var maxRunning atomic.Int32

	blocker := make(chan struct{})
	task1Started := make(chan struct{})

	status1 := handler.Submit(func() {
		close(task1Started)
		<-blocker
	})
	assert.Equal(t, StatusAccepted, status1)

	<-task1Started

	task2Done := make(chan struct{})
	status2 := handler.Submit(func() {
		current := running.Add(1)
		for {
			old := maxRunning.Load()
			if current <= old {
				break
			}
			if maxRunning.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		running.Add(-1)
		close(task2Done)
	})
	assert.Equal(t, StatusAccepted, status2)

	status3 := handler.Submit(func() {
		t.Error("Эта задача не должна выполниться")
	})
	assert.Equal(t, StatusRejected, status3)

	close(blocker)

	<-task2Done

	handler.Wait()
	handler.Stop()

	assert.LessOrEqual(t, int(maxRunning.Load()), 2)
}

func TestConcurrencyHandler_Stop(t *testing.T) {
	handler, err := NewConcurrencyHandler(1)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	taskStarted := make(chan struct{})
	taskCompleted := make(chan struct{})

	status := handler.Submit(func() {
		close(taskStarted)
		time.Sleep(100 * time.Millisecond)
		close(taskCompleted)
	})
	assert.Equal(t, StatusAccepted, status)

	<-taskStarted

	handler.Stop()

	select {
	case <-taskCompleted:
		// OK
	case <-time.After(50 * time.Millisecond):
		t.Error("Задача не завершилась")
	}

	assert.True(t, handler.stopped.Load())
}

func TestConcurrencyHandler_Wait(t *testing.T) {
	handler, err := NewConcurrencyHandler(2)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	tasksDone := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		status := handler.Submit(func() {
			time.Sleep(50 * time.Millisecond)
			mu.Lock()
			tasksDone++
			mu.Unlock()
		})
		if i < 2 {
			assert.Equal(t, StatusAccepted, status)
		} else {
			assert.Equal(t, StatusRejected, status)
		}
	}

	handler.Wait()

	mu.Lock()
	done := tasksDone
	mu.Unlock()

	assert.Equal(t, 2, done)
	handler.Stop()
}

func TestConcurrencyHandler_Reuse(t *testing.T) {
	handler, err := NewConcurrencyHandler(1)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	done1 := make(chan bool)
	status1 := handler.Submit(func() {
		done1 <- true
	})
	assert.Equal(t, StatusAccepted, status1)
	<-done1

	handler.Stop()

	err = handler.Start()
	require.NoError(t, err)

	done2 := make(chan bool)
	status2 := handler.Submit(func() {
		done2 <- true
	})
	assert.Equal(t, StatusAccepted, status2)
	<-done2

	handler.Stop()
}

func TestConcurrencyHandler_NilFunction(t *testing.T) {
	handler, err := NewConcurrencyHandler(1)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	status := handler.Submit(nil)
	assert.Equal(t, StatusRejected, status)

	handler.Stop()
}

func TestConcurrencyHandler_StoppedImmediate(t *testing.T) {
	handler, err := NewConcurrencyHandler(1)
	require.NoError(t, err)

	handler.stopped.Store(true)

	status := handler.Submit(func() {
		t.Error("Эта задача не должна выполниться")
	})
	assert.Equal(t, StatusRejected, status)
}

func TestNewConcurrencyHandler(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{"positive", 5, 5},
		{"zero", 0, 1},
		{"negative", -1, 1},
		{"one", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewConcurrencyHandler(tt.limit)
			assert.NoError(t, err)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.expected, handler.limit)
			assert.False(t, handler.stopped.Load())
		})
	}
}

func TestConcurrencyHandler_ConcurrentSafe(t *testing.T) {
	handler, err := NewConcurrencyHandler(5)
	require.NoError(t, err)

	err = handler.Start()
	require.NoError(t, err)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var rejectCount atomic.Int32

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := handler.Submit(func() {
				time.Sleep(10 * time.Millisecond)
			})
			if status == StatusAccepted {
				successCount.Add(1)
			} else {
				rejectCount.Add(1)
			}
		}()
	}

	wg.Wait()

	handler.Wait()
	handler.Stop()

	total := successCount.Load() + rejectCount.Load()
	assert.Equal(t, int32(20), total)
}
