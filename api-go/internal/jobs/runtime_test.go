package jobs_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/jobs"
)

func TestEnqueueRuntimeProbe_UsesUniqueness(t *testing.T) {
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisOpt := asynq.RedisClientOpt{Addr: redisServer.Addr()}
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	ctx := context.Background()
	_, err = jobs.EnqueueRuntimeProbe(ctx, client, "duplicate-key")
	require.NoError(t, err)

	_, err = jobs.EnqueueRuntimeProbe(ctx, client, "duplicate-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestServerProcessesRuntimeProbeTask(t *testing.T) {
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisOpt := asynq.RedisClientOpt{Addr: redisServer.Addr()}
	server := jobs.NewServer(redisOpt, jobs.ServerConfig{Concurrency: 1})
	defer server.Shutdown()

	processed := make(chan struct{}, 1)
	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TaskTypeRuntimeProbe, func(_ context.Context, _ *asynq.Task) error {
		processed <- struct{}{}
		return nil
	})

	require.NoError(t, server.Start(mux))

	client := asynq.NewClient(redisOpt)
	defer client.Close()

	_, err = jobs.EnqueueRuntimeProbe(context.Background(), client, "worker-check")
	require.NoError(t, err)

	select {
	case <-processed:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker to process runtime probe task")
	}
}

func TestServerRetriesFailedTask(t *testing.T) {
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisOpt := asynq.RedisClientOpt{Addr: redisServer.Addr()}
	server := jobs.NewServer(redisOpt, jobs.ServerConfig{
		Concurrency: 1,
		RetryDelayFunc: func(_ int, _ error, _ *asynq.Task) time.Duration {
			return 50 * time.Millisecond
		},
		TaskCheckInterval:        50 * time.Millisecond,
		DelayedTaskCheckInterval: 50 * time.Millisecond,
	})
	defer server.Shutdown()

	var attempts atomic.Int32
	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TaskTypeRetryProbe, func(_ context.Context, _ *asynq.Task) error {
		if attempts.Add(1) == 1 {
			return errors.New("force first attempt failure")
		}
		return nil
	})

	require.NoError(t, server.Start(mux))

	client := asynq.NewClient(redisOpt)
	defer client.Close()

	_, err = jobs.EnqueueRetryProbe(context.Background(), client, "retry-check")
	require.NoError(t, err)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if attempts.Load() >= 2 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("expected at least 2 attempts, got %d", attempts.Load())
}
