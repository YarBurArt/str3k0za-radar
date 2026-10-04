package telegram

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/net/proxy"
)

// little workaround for server location
func NewBot(token, proxyAddr string) (*bot.Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("bot token is empty")
	}

	httpClient := &http.Client{}
	if proxyAddr != "" {
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("create SOCKS5 proxy dialer: %w", err)
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("SOCKS5 dialer %T does not support DialContext", dialer)
		}
		httpClient = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return contextDialer.DialContext(ctx, network, addr)
				},
			},
		}
	}

	b, err := bot.New(token,
		bot.WithHTTPClient(15*time.Second, httpClient),
		// the router does not exist yet, so the fallback cannot be a method on it
		bot.WithDefaultHandler(EchoFallback),
	)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	return b, nil
}

// without this the library only logs the update, so the user gets silence
func EchoFallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Unknown command. Try /help",
	})
}
