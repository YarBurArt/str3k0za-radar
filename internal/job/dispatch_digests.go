package job

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

type DispatchDigestsArgs struct{}

func (DispatchDigestsArgs) Kind() string { return "dispatch_digests" }

type DispatchDigestsWorker struct {
	river.WorkerDefaults[DispatchDigestsArgs]
	userRepo *postgres.UserRepository
}

func NewDispatchDigestsWorker(userRepo *postgres.UserRepository) *DispatchDigestsWorker {
	return &DispatchDigestsWorker{userRepo: userRepo}
}

func (w *DispatchDigestsWorker) Work(ctx context.Context, job *river.Job[DispatchDigestsArgs]) error {
	targets, err := w.userRepo.ListUsersForDelivery(ctx)
	if err != nil {
		return fmt.Errorf("list users for delivery: %w", err)
	}
	if len(targets) == 0 {
		return nil
	}

	log.Printf("dispatch_digests: %d target(s) due", len(targets))

	client := river.ClientFromContext[pgx.Tx](ctx)
	for _, target := range targets {
		_, err := client.Insert(ctx, SendDigestArgs{TelegramID: target.TelegramID}, &river.InsertOpts{
			// prevents double-send on leader failover
			UniqueOpts: river.UniqueOpts{
				ByArgs:   true,
				ByPeriod: 24 * time.Hour,
			},
		})
		if err != nil {
			return fmt.Errorf("insert send_digest for user %d: %w", target.TelegramID, err)
		}
	}
	return nil
}
