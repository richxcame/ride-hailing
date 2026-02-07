package async_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/richxcame/ride-hailing/pkg/async"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestCaptureContext(t *testing.T) {
	correlationID := "test-correlation-123"
	ctx := logger.ContextWithCorrelationID(context.Background(), correlationID)

	tc := async.CaptureContext(ctx, "test-task")

	assert.Equal(t, correlationID, tc.CorrelationID)
	assert.Equal(t, "test-task", tc.TaskName)
	assert.False(t, tc.StartTime.IsZero())
}

func TestTaskContext_NewContext(t *testing.T) {
	correlationID := "test-correlation-456"
	ctx := logger.ContextWithCorrelationID(context.Background(), correlationID)

	tc := async.CaptureContext(ctx, "test-task")
	newCtx := tc.NewContext()

	// Verify correlation ID is preserved
	extractedID := logger.CorrelationIDFromContext(newCtx)
	assert.Equal(t, correlationID, extractedID)
}

func TestTaskContext_NewContextWithTimeout(t *testing.T) {
	correlationID := "test-correlation-789"
	ctx := logger.ContextWithCorrelationID(context.Background(), correlationID)

	tc := async.CaptureContext(ctx, "test-task")
	newCtx, cancel := tc.NewContextWithTimeout(100 * time.Millisecond)
	defer cancel()

	// Verify correlation ID is preserved
	extractedID := logger.CorrelationIDFromContext(newCtx)
	assert.Equal(t, correlationID, extractedID)

	// Verify timeout works
	select {
	case <-newCtx.Done():
		// Expected after timeout
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

func TestGo_PropagatesContext(t *testing.T) {
	correlationID := "test-go-correlation"
	ctx := logger.ContextWithCorrelationID(context.Background(), correlationID)

	var capturedID string
	var wg sync.WaitGroup
	wg.Add(1)

	async.Go(ctx, "test-task", func(ctx context.Context) {
		defer wg.Done()
		capturedID = logger.CorrelationIDFromContext(ctx)
	})

	wg.Wait()
	assert.Equal(t, correlationID, capturedID)
}

func TestGo_RecoversPanic(t *testing.T) {
	ctx := context.Background()

	// This should not panic the test
	async.Go(ctx, "panic-task", func(ctx context.Context) {
		panic("test panic")
	})

	// Give goroutine time to complete
	time.Sleep(50 * time.Millisecond)
}

func TestGoWithTimeout_TimesOut(t *testing.T) {
	ctx := context.Background()

	var timedOut bool
	var wg sync.WaitGroup
	wg.Add(1)

	async.GoWithTimeout(ctx, "timeout-task", 50*time.Millisecond, func(ctx context.Context) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			timedOut = true
		case <-time.After(100 * time.Millisecond):
			timedOut = false
		}
	})

	wg.Wait()
	assert.True(t, timedOut)
}

func TestGoWithCallback_Success(t *testing.T) {
	ctx := context.Background()

	var callbackErr error
	callbackCalled := false
	var wg sync.WaitGroup
	wg.Add(1)

	async.GoWithCallback(ctx, "callback-task", func(ctx context.Context) error {
		return nil
	}, func(err error) {
		defer wg.Done()
		callbackCalled = true
		callbackErr = err
	})

	wg.Wait()
	assert.True(t, callbackCalled)
	assert.NoError(t, callbackErr)
}

func TestGoWithCallback_Error(t *testing.T) {
	ctx := context.Background()

	var callbackErr error
	var wg sync.WaitGroup
	wg.Add(1)

	async.GoWithCallback(ctx, "callback-task", func(ctx context.Context) error {
		return assert.AnError
	}, func(err error) {
		defer wg.Done()
		callbackErr = err
	})

	wg.Wait()
	assert.Error(t, callbackErr)
}

func TestRunAll_AllComplete(t *testing.T) {
	ctx := context.Background()

	var results []int
	var mu sync.Mutex

	async.RunAll(ctx, "batch-task",
		func(ctx context.Context) {
			mu.Lock()
			results = append(results, 1)
			mu.Unlock()
		},
		func(ctx context.Context) {
			mu.Lock()
			results = append(results, 2)
			mu.Unlock()
		},
		func(ctx context.Context) {
			mu.Lock()
			results = append(results, 3)
			mu.Unlock()
		},
	)

	assert.Len(t, results, 3)
	assert.Contains(t, results, 1)
	assert.Contains(t, results, 2)
	assert.Contains(t, results, 3)
}

func TestRunAll_PropagatesContext(t *testing.T) {
	correlationID := "batch-correlation"
	ctx := logger.ContextWithCorrelationID(context.Background(), correlationID)

	var capturedIDs []string
	var mu sync.Mutex

	async.RunAll(ctx, "batch-task",
		func(ctx context.Context) {
			mu.Lock()
			capturedIDs = append(capturedIDs, logger.CorrelationIDFromContext(ctx))
			mu.Unlock()
		},
		func(ctx context.Context) {
			mu.Lock()
			capturedIDs = append(capturedIDs, logger.CorrelationIDFromContext(ctx))
			mu.Unlock()
		},
	)

	assert.Len(t, capturedIDs, 2)
	for _, id := range capturedIDs {
		assert.Equal(t, correlationID, id)
	}
}

func TestWithCorrelationID(t *testing.T) {
	ctx := context.Background()
	correlationID := "new-correlation"

	newCtx := async.WithCorrelationID(ctx, correlationID)
	extractedID := async.GetCorrelationID(newCtx)

	assert.Equal(t, correlationID, extractedID)
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"

	newCtx := async.WithUserID(ctx, userID)
	extractedID := async.GetUserID(newCtx)

	assert.Equal(t, userID, extractedID)
}

func TestGetUserID_NotSet(t *testing.T) {
	ctx := context.Background()
	userID := async.GetUserID(ctx)
	assert.Empty(t, userID)
}
