package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/yarburart/str3k0za-radar/internal/application"
	"github.com/yarburart/str3k0za-radar/internal/bootstrap"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
	"github.com/yarburart/str3k0za-radar/internal/job"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b, err := bootstrap.Bot()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := bootstrap.Pool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	userRepo := postgres.NewUserRepository(pool)

	attackGraph, cweData, err := bootstrap.KnowledgeBase()
	if err != nil {
		log.Fatal(err)
	}

	schedule, err := job.EveryMinuteSchedule()
	if err != nil {
		log.Fatal(err)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			// modest concurrency: Telegram rate limits
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: job.NewWorkers(
			job.NewDispatchDigestsWorker(userRepo),
			job.NewSendDigestWorker(userRepo, application.NewDigestService(userRepo, attackGraph, cweData), b),
		),
		PeriodicJobs: []*river.PeriodicJob{
			job.DispatchDigestsPeriodicJob(schedule),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("failed to start river client: %v", err)
	}
	log.Println("river worker started")

	<-ctx.Done()
	log.Println("shutting down river worker...")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	if err := riverClient.Stop(stopCtx); err != nil {
		log.Printf("river client stop error: %v", err)
	}
}
