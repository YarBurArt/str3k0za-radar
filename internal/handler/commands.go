package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (r *Router) Start(
	ctx context.Context, b *bot.Bot, update *models.Update,
) {
	if update.Message == nil {
		return
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Setup APT filter", CallbackData: "apt:configure"},
				{Text: "Filter by Source Country", CallbackData: "country:configure"},
			},
		},
	}
	// TODO: help message
	text := "bot is started"
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
}

func (r *Router) Digest(
	ctx context.Context, b *bot.Bot, update *models.Update,
) {
	if update.Message == nil {
		return
	}
	// TODO:
	text := "Generation of digest ..."

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}
