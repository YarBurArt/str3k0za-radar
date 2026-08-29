package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type UserUIState struct {
	SelectedGroups    map[string]bool
	SelectedCountries map[string]bool
	CurrentPage       int
}

type Router struct {
	bot *bot.Bot
	// TODO: connect update profile and generate digest
	// temporal workaround for ui response
	uiState map[int64]*UserUIState
}

func NewRouter(b *bot.Bot) *Router {
	r := &Router{
		bot:     b,
		uiState: make(map[int64]*UserUIState),
	}
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, r.Start)
	r.bot.RegisterHandler(bot.HandlerTypeMessageText, "/digest", bot.MatchTypeExact, r.Digest)

	r.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "apt:", bot.MatchTypePrefix, r.APTCallback)
	r.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "country:", bot.MatchTypePrefix, r.APTSourceCountryCallback)

	return r
}

func (r *Router) getState(chatID int64) *UserUIState {
	state := r.uiState[chatID]
	if state == nil {
		state = &UserUIState{
			SelectedGroups:    make(map[string]bool),
			SelectedCountries: make(map[string]bool),
			CurrentPage:       0,
		}
		r.uiState[chatID] = state
	}
	return state
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
