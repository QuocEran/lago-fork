package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hibiken/asynq"
)

const (
	defaultConcurrency = 10

	queueCritical = "critical"
	queueDefault  = "default"
	queueLow      = "low"
)

const (
	TaskTypeRuntimeProbe = "runtime:probe"
	TaskTypeRetryProbe   = "runtime:retry_probe"
)

type ServerConfig struct {
	Concurrency              int
	RetryDelayFunc           asynq.RetryDelayFunc
	TaskCheckInterval        time.Duration
	DelayedTaskCheckInterval time.Duration
}

func ParseRedisConnOpt(redisURL string, redisPassword string, redisDB int) (asynq.RedisClientOpt, error) {
	if redisURL == "" {
		return asynq.RedisClientOpt{}, errors.New("redis URL is required")
	}

	parsedURL, err := url.Parse(redisURL)
	if err != nil {
		return asynq.RedisClientOpt{}, fmt.Errorf("parse redis URL: %w", err)
	}

	if parsedURL.Host == "" {
		return asynq.RedisClientOpt{}, errors.New("redis URL host is required")
	}

	return asynq.RedisClientOpt{
		Addr:     parsedURL.Host,
		Password: redisPassword,
		DB:       redisDB,
		Network:  "tcp",
	}, nil
}

func NewServer(redisConnOpt asynq.RedisConnOpt, cfg ServerConfig) *asynq.Server {
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	retryDelayFunc := cfg.RetryDelayFunc
	if retryDelayFunc == nil {
		retryDelayFunc = asynq.DefaultRetryDelayFunc
	}

	taskCheckInterval := cfg.TaskCheckInterval
	if taskCheckInterval <= 0 {
		taskCheckInterval = time.Second
	}

	delayedTaskCheckInterval := cfg.DelayedTaskCheckInterval
	if delayedTaskCheckInterval <= 0 {
		delayedTaskCheckInterval = 5 * time.Second
	}

	return asynq.NewServer(redisConnOpt, asynq.Config{
		Concurrency:       concurrency,
		TaskCheckInterval: taskCheckInterval,
		Queues: map[string]int{
			queueCritical: 6,
			queueDefault:  3,
			queueLow:      1,
		},
		StrictPriority:           true,
		RetryDelayFunc:           retryDelayFunc,
		DelayedTaskCheckInterval: delayedTaskCheckInterval,
		ErrorHandler: asynq.ErrorHandlerFunc(func(_ context.Context, task *asynq.Task, err error) {
			slog.Error("asynq task failed",
				slog.String("task_type", task.Type()),
				slog.String("error", err.Error()),
			)
		}),
	})
}

func NewScheduler(redisConnOpt asynq.RedisConnOpt) *asynq.Scheduler {
	return asynq.NewScheduler(redisConnOpt, &asynq.SchedulerOpts{Location: time.UTC})
}

func RegisterDefaultSchedules(scheduler *asynq.Scheduler) error {
	if scheduler == nil {
		return errors.New("scheduler is required")
	}

	_, err := scheduler.Register("*/5 * * * *", NewRuntimeProbeTask("scheduled"), asynq.Queue(queueLow), asynq.MaxRetry(5), asynq.Unique(10*time.Minute))
	if err != nil {
		return fmt.Errorf("register runtime probe schedule: %w", err)
	}

	return nil
}

func NewRuntimeProbeTask(source string) *asynq.Task {
	return asynq.NewTask(TaskTypeRuntimeProbe, []byte(source))
}

func NewRetryProbeTask(source string) *asynq.Task {
	return asynq.NewTask(TaskTypeRetryProbe, []byte(source))
}

func EnqueueRuntimeProbe(ctx context.Context, client *asynq.Client, source string) (*asynq.TaskInfo, error) {
	if client == nil {
		return nil, errors.New("asynq client is required")
	}

	task := NewRuntimeProbeTask(source)
	info, err := client.EnqueueContext(ctx, task,
		asynq.Queue(queueDefault),
		asynq.MaxRetry(10),
		asynq.Timeout(30*time.Second),
		asynq.Unique(10*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("enqueue runtime probe task: %w", err)
	}

	return info, nil
}

func EnqueueRetryProbe(ctx context.Context, client *asynq.Client, source string) (*asynq.TaskInfo, error) {
	if client == nil {
		return nil, errors.New("asynq client is required")
	}

	task := NewRetryProbeTask(source)
	info, err := client.EnqueueContext(ctx, task,
		asynq.Queue(queueCritical),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("enqueue retry probe task: %w", err)
	}

	return info, nil
}

func NewDefaultServeMux() *asynq.ServeMux {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskTypeRuntimeProbe, func(_ context.Context, task *asynq.Task) error {
		slog.Info("runtime probe processed", slog.String("payload", string(task.Payload())))
		return nil
	})
	mux.HandleFunc(TaskTypeRetryProbe, func(_ context.Context, task *asynq.Task) error {
		slog.Info("retry probe processed", slog.String("payload", string(task.Payload())))
		return nil
	})

	return mux
}
