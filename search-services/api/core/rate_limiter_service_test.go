package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiterHandler(t *testing.T) {
	tests := []struct {
		name     string
		rate     int
		expected int
	}{
		{"positive rate", 10, 10},
		{"zero rate", 0, 1},
		{"negative rate", -1, 1},
		{"single rate", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := NewRateLimiterHandler(tt.rate)
			assert.NoError(t, err)
			assert.NotNil(t, rl)
		})
	}
}

func TestRateLimiterHandler_StartStop(t *testing.T) {
	rl, err := NewRateLimiterHandler(10)
	require.NoError(t, err)

	err = rl.Start()
	assert.NoError(t, err)

	rl.Stop()
}

func TestRateLimiterHandler_Wait_Immediate(t *testing.T) {
	rl, err := NewRateLimiterHandler(100)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)
	defer rl.Stop()

	err = rl.Wait(context.Background())
	assert.NoError(t, err)
}

func TestRateLimiterHandler_Wait_ContextTimeout(t *testing.T) {
	rl, err := NewRateLimiterHandler(1)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)
	defer rl.Stop()

	err = rl.Wait(context.Background())
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = rl.Wait(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestRateLimiterHandler_Submit_Success(t *testing.T) {
	rl, err := NewRateLimiterHandler(100)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)
	defer rl.Stop()

	executed := false
	status := rl.Submit(func() {
		executed = true
	})

	assert.Equal(t, StatusAccepted, status)
	assert.True(t, executed)
}

func TestRateLimiterHandler_Submit_NilFunction(t *testing.T) {
	rl, err := NewRateLimiterHandler(10)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)
	defer rl.Stop()

	status := rl.Submit(nil)
	assert.Equal(t, StatusRejected, status)
}

func TestRateLimiterHandler_Submit_RateLimit(t *testing.T) {
	rl, err := NewRateLimiterHandler(2)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)
	defer rl.Stop()

	executed1 := false
	status1 := rl.Submit(func() {
		executed1 = true
	})
	assert.Equal(t, StatusAccepted, status1)
	assert.True(t, executed1)

	executed2 := false
	status2 := rl.Submit(func() {
		executed2 = true
	})
	assert.Equal(t, StatusAccepted, status2)
	assert.True(t, executed2)

	start := time.Now()
	executed3 := false
	status3 := rl.Submit(func() {
		executed3 = true
	})
	elapsed := time.Since(start)

	assert.Equal(t, StatusAccepted, status3)
	assert.True(t, executed3)
	assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond)
}

func TestRateLimiterHandler_Wait_WhenStopped(t *testing.T) {
	rl, err := NewRateLimiterHandler(10)
	require.NoError(t, err)

	err = rl.Wait(context.Background())
	assert.NoError(t, err)
}

func TestRateLimiterHandler_Stop_DuringWait(t *testing.T) {
	rl, err := NewRateLimiterHandler(1)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)

	err = rl.Wait(context.Background())
	assert.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- rl.Wait(context.Background())
	}()

	time.Sleep(10 * time.Millisecond)
	rl.Stop()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout")
	}
}

func TestRateLimiterHandler_Reuse(t *testing.T) {
	rl, err := NewRateLimiterHandler(10)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)

	executed1 := false
	status1 := rl.Submit(func() {
		executed1 = true
	})
	assert.Equal(t, StatusAccepted, status1)
	assert.True(t, executed1)

	rl.Stop()

	err = rl.Start()
	require.NoError(t, err)

	executed2 := false
	status2 := rl.Submit(func() {
		executed2 = true
	})
	assert.Equal(t, StatusAccepted, status2)
	assert.True(t, executed2)

	rl.Stop()
}

func TestRateLimiterHandler_Submit_Multiple(t *testing.T) {
	rl, err := NewRateLimiterHandler(5)
	require.NoError(t, err)

	err = rl.Start()
	require.NoError(t, err)
	defer rl.Stop()

	count := 0
	for i := 0; i < 3; i++ {
		status := rl.Submit(func() {
			count++
		})
		assert.Equal(t, StatusAccepted, status)
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 3, count)
}
