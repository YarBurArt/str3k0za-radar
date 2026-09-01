package handler

import (
	"context"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/yarburart/str3k0za-radar/internal/application"
)

type UserUIState struct {
	SelectedAPTGroups map[string]bool
	APTPage           int
	SelectedCountries map[string]bool
	CountryPage       int
}

type Router struct {
	bot *bot.Bot
	// TODO: connect generate digest service
	userService   *application.UserService
	digestService *application.DigestService
	// state before save per user, without concurrent map writes
	mu      sync.Mutex
	uiState map[int64]*UserUIState
}

func NewRouter(b *bot.Bot, userService *application.UserService, digestService *application.DigestService) *Router {
	r := &Router{
		bot:           b,
		uiState:       make(map[int64]*UserUIState),
		userService:   userService,
		digestService: digestService,
	}
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, r.Start)
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/digest", bot.MatchTypeExact, r.Digest)
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/enable", bot.MatchTypeExact, r.EnableDigest)
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/disable", bot.MatchTypeExact, r.DisableDigest)
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/settime", bot.MatchTypePrefix, r.SetTime)

	r.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "apt:", bot.MatchTypePrefix, r.APTCallback)
	r.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "country:", bot.MatchTypePrefix, r.APTSourceCountryCallback)

	return r
}

func (r *Router) getState(chatID int64) *UserUIState {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.uiState[chatID]
	if state == nil {
		state = &UserUIState{
			SelectedAPTGroups: make(map[string]bool),
			APTPage:           0,
			SelectedCountries: make(map[string]bool),
			CountryPage:       0,
		}
		r.uiState[chatID] = state
	}
	return state
}

// drops transient data after a successful save to free
func (r *Router) clearState(chatID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.uiState, chatID)
}

func (r *Router) EchoFallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "command not found",
	})
}
