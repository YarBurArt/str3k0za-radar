package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"

	"github.com/yarburart/str3k0za-radar/internal/application"
	"github.com/yarburart/str3k0za-radar/internal/bootstrap"
	"github.com/yarburart/str3k0za-radar/internal/handler"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
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

	router := handler.NewRouter(
		b,
		application.NewUserService(userRepo, attackGraph),
		application.NewDigestService(userRepo, attackGraph, cweData),
	)
	b.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypeExact, router.EchoFallback)

	b.Start(ctx)
}
