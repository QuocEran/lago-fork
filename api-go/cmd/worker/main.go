package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/getlago/lago/api-go/config"
	"github.com/getlago/lago/api-go/internal/jobs"
)

func main() {
	cfg := config.Load()

	redisOpt, err := jobs.ParseRedisConnOpt(cfg.Redis.URL, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		slog.Error("failed to parse redis config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	server := jobs.NewServer(redisOpt, jobs.ServerConfig{Concurrency: 20})
	scheduler := jobs.NewScheduler(redisOpt)
	mux := jobs.NewDefaultServeMux()

	if err := jobs.RegisterDefaultSchedules(scheduler); err != nil {
		slog.Error("failed to register schedules", slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		if runErr := scheduler.Run(); runErr != nil {
			slog.Error("scheduler exited", slog.String("error", runErr.Error()))
		}
	}()

	go func() {
		if runErr := server.Run(mux); runErr != nil {
			slog.Error("worker exited", slog.String("error", runErr.Error()))
		}
	}()

	slog.Info("asynq worker started", slog.String("redis", fmt.Sprintf("%T", redisOpt)))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	scheduler.Shutdown()
	server.Shutdown()
	slog.Info("asynq worker stopped")
}

var _ asynq.RedisConnOpt = asynq.RedisClientOpt{}
