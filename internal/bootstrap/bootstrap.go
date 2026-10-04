package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-telegram/bot"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yarburart/str3k0za-radar/internal/domain"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/cwe"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/mitre"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/telegram"
)

const (
	envBotToken = "BOT_TOKEN"
	envDBURL    = "DB_URL"
	envProxy    = "SOCKS5_PROXY"
	// modern GDPR problems need modern sollutions, bad opsec btw
	defaultProxyAddr = "127.0.0.1:9050"

	attackDataPath  = "data/enterprise-attack.json"
	threatGroupPath = "data/threat-groups.json"
	cweDataPath     = "data/cwe-1000.csv"
)

func Pool(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv(envDBURL)
	if dbURL == "" {
		return nil, fmt.Errorf("%s env var is not set for current process", envDBURL)
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	return pool, nil
}

func Bot() (*bot.Bot, error) {
	botToken := os.Getenv(envBotToken)
	if botToken == "" {
		return nil, fmt.Errorf("%s env var is not set for current process", envBotToken)
	}

	proxyAddr := os.Getenv(envProxy)
	if proxyAddr == "" {
		proxyAddr = defaultProxyAddr
	}
	return telegram.NewBot(botToken, proxyAddr)
}

// 45 MB ATT&CK dump, parsed once per startup since both binaries need it
func KnowledgeBase() (*domain.AttackGraph, []domain.CWE, error) {
	attackGraph, err := mitre.NewLoader(attackDataPath, threatGroupPath).Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load attack graph: %w", err)
	}
	log.Printf("Attack graph loaded: %d APTs, %d TTPs", len(attackGraph.APTs), len(attackGraph.TTPs))

	_, cweData, err := cwe.LoadCWEdata(cweDataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load cwe dataset: %w", err)
	}
	return attackGraph, cweData, nil
}
