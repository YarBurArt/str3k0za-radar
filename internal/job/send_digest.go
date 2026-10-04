package job

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/riverqueue/river"

	"github.com/yarburart/str3k0za-radar/internal/application"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

type SendDigestArgs struct {
	TelegramID int64 `json:"telegram_id"`
}

func (SendDigestArgs) Kind() string { return "send_digest" }

type SendDigestWorker struct {
	river.WorkerDefaults[SendDigestArgs]
	userRepo      *postgres.UserRepository
	digestService *application.DigestService
	bot           *bot.Bot
}

func NewSendDigestWorker(
	userRepo *postgres.UserRepository,
	digestService *application.DigestService,
	b *bot.Bot,
) *SendDigestWorker {
	return &SendDigestWorker{
		userRepo:      userRepo,
		digestService: digestService,
		bot:           b,
	}
}

func (w *SendDigestWorker) Work(ctx context.Context, job *river.Job[SendDigestArgs]) error {
	telegramID := job.Args.TelegramID
	if telegramID == 0 {
		return river.JobCancel(fmt.Errorf("send_digest: empty telegram_id"))
	}

	// prefs may have changed between dispatch and this run
	user, err := w.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			log.Printf("send_digest: user %d not found, skipping", telegramID)
			return nil
		}
		return fmt.Errorf("load user %d: %w", telegramID, err)
	}
	if !user.Prefs.DigestEnabled {
		log.Printf("send_digest: user %d disabled digest, skipping", telegramID)
		return nil
	}

	text, err := w.digestService.GenerateDigestMessage(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("generate digest for user %d: %w", telegramID, err)
	}

	_, err = w.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: telegramID,
		Text:   text,
	})
	if err != nil {
		// if bot blocked or chat deleted
		if errors.Is(err, bot.ErrorForbidden) {
			return river.JobCancel(fmt.Errorf("send_digest user %d: %w", telegramID, err))
		}
		return fmt.Errorf("send digest to user %d: %w", telegramID, err)
	}

	log.Printf("send_digest: delivered to user %d", telegramID)
	return nil
}
