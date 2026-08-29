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
	"golang.org/x/net/proxy"

	"github.com/yarburart/str3k0za-radar/internal/handler"
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
	router := handler.NewRouter(b)
	b.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypeExact, router.EchoFallback)

	b.Start(ctx)
}
