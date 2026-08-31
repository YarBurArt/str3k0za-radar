package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/proxy"

	"github.com/yarburart/str3k0za-radar/internal/application"
	"github.com/yarburart/str3k0za-radar/internal/handler"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/mitre"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

// test bot echo
func main() {
	// modern GDPR problems need modern sollutions, bad opsec btw
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, proxy.Direct)
	if err != nil {
		log.Fatalf("failed to create SOCKS5 proxy dialer: %v", err)
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
		},
	}
	httpClient := &http.Client{
		Transport: transport,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// WithHTTPClient for proxy in censored countries
	opts := []bot.Option{
		bot.WithHTTPClient(15*time.Second, httpClient),
	}
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN env var is not set for current process")
	}
	b, err := bot.New(botToken, opts...)
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("cant connect to db: %v\n", err)
	}
	defer pool.Close()
	userRepo := postgres.NewUserRepository(pool)

	loader := mitre.NewLoader(
		"data/enterprise-attack.json",
		"data/threat-groups.json",
	)
	attackGraph, err := loader.Load()
	if err != nil {
		log.Fatalf("failed to load attack graph: %v", err)
	}
	log.Printf("Attack graph loaded: %d APTs, %d TTPs", len(attackGraph.APTs), len(attackGraph.TTPs))

	userProfileService := application.NewUserService(userRepo, attackGraph)
	router := handler.NewRouter(b, userProfileService)
	b.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypeExact, router.EchoFallback)

	b.Start(ctx)
}
