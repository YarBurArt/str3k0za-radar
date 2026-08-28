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
	"github.com/go-telegram/bot/models"
	"golang.org/x/net/proxy"
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
		bot.WithDefaultHandler(handler),
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

	b.Start(ctx)
}

// handle any message , response same text to source chatid
func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
	if err != nil {
		log.Printf("failed to send message: %v", err)
	}
}
