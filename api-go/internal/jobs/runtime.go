package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/jobs/handlers"
	invsvc "github.com/getlago/lago/api-go/internal/services/invoices"
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

	// Clock job: finalize draft invoices every minute.
	_, err = scheduler.Register("*/1 * * * *",
		asynq.NewTask(handlers.TaskTypeFinalizeInvoice, nil),
		asynq.Queue(queueCritical), asynq.MaxRetry(3), asynq.Unique(5*time.Minute))
	if err != nil {
		return fmt.Errorf("register finalize invoices schedule: %w", err)
	}

	// Clock job: mark overdue invoices every 10 minutes.
	_, err = scheduler.Register("*/10 * * * *",
		asynq.NewTask(handlers.TaskTypeMarkPaymentOverdue, nil),
		asynq.Queue(queueDefault), asynq.MaxRetry(3), asynq.Unique(10*time.Minute))
	if err != nil {
		return fmt.Errorf("register mark payment overdue schedule: %w", err)
	}

	// Clock job: validate events every 5 minutes.
	_, err = scheduler.Register("*/5 * * * *",
		asynq.NewTask(handlers.TaskTypeValidateEvents, nil),
		asynq.Queue(queueDefault), asynq.MaxRetry(3), asynq.Unique(5*time.Minute))
	if err != nil {
		return fmt.Errorf("register validate events schedule: %w", err)
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

// EnqueueFinalizeInvoice enqueues a task to finalize one draft invoice.
func EnqueueFinalizeInvoice(ctx context.Context, client *asynq.Client, organizationID, invoiceID string) (*asynq.TaskInfo, error) {
	if client == nil {
		return nil, errors.New("asynq client is required")
	}
	task, err := handlers.NewFinalizeInvoiceTask(organizationID, invoiceID)
	if err != nil {
		return nil, err
	}
	info, err := client.EnqueueContext(ctx, task,
		asynq.Queue(queueCritical),
		asynq.MaxRetry(5),
		asynq.Timeout(60*time.Second),
		asynq.Unique(5*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("enqueue finalize invoice: %w", err)
	}
	return info, nil
}

// EnqueueSendHTTPWebhook enqueues a task to deliver one outbound webhook.
func EnqueueSendHTTPWebhook(ctx context.Context, client *asynq.Client, webhookID string) (*asynq.TaskInfo, error) {
	if client == nil {
		return nil, errors.New("asynq client is required")
	}
	task, err := handlers.NewSendHTTPWebhookTask(webhookID)
	if err != nil {
		return nil, err
	}
	info, err := client.EnqueueContext(ctx, task,
		asynq.Queue(queueDefault),
		asynq.MaxRetry(10),
		asynq.Timeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("enqueue send_http_webhook: %w", err)
	}
	return info, nil
}

// ServeMuxConfig holds runtime dependencies for building the default serve mux.
type ServeMuxConfig struct {
	DB             *gorm.DB
	InvoiceSvc     invsvc.Service
	WebhookSecret  string
}

func NewDefaultServeMux() *asynq.ServeMux {
	return NewServeMuxWithConfig(ServeMuxConfig{})
}

// NewServeMuxWithConfig builds the handler mux, injecting real service dependencies.
func NewServeMuxWithConfig(cfg ServeMuxConfig) *asynq.ServeMux {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskTypeRuntimeProbe, func(_ context.Context, task *asynq.Task) error {
		slog.Info("runtime probe processed", slog.String("payload", string(task.Payload())))
		return nil
	})
	mux.HandleFunc(TaskTypeRetryProbe, func(_ context.Context, task *asynq.Task) error {
		slog.Info("retry probe processed", slog.String("payload", string(task.Payload())))
		return nil
	})

	// Invoice handlers
	if cfg.InvoiceSvc != nil {
		mux.HandleFunc(handlers.TaskTypeFinalizeInvoice, handlers.HandleFinalizeInvoice(cfg.InvoiceSvc))
	}
	if cfg.DB != nil {
		mux.HandleFunc(handlers.TaskTypeMarkPaymentOverdue, handlers.HandleMarkPaymentOverdue(cfg.DB))
	}

	// Event handler (stub)
	mux.HandleFunc(handlers.TaskTypeValidateEvents, handlers.HandleValidateEvents())

	// Payment handler (stub)
	mux.HandleFunc(handlers.TaskTypeCreatePayment, handlers.HandleCreatePayment())

	// Webhook handler
	if cfg.DB != nil {
		mux.HandleFunc(handlers.TaskTypeSendHTTPWebhook, handlers.HandleSendHTTPWebhook(cfg.DB, cfg.WebhookSecret))
	}

	return mux
}
