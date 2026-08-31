package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/yarburart/str3k0za-radar/internal/domain"
)

func (r *Router) Start(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	text := "Bot is started.\n\n" +
		"Available commands:\n" +
		"- /enable-digest  : enable daily digest\n" +
		"- /disable-digest : disable daily digest\n" +
		"- /settime HH:MM  : set delivery time (like /settime 14:30)\n" +
		"- /digest : Generate report manually with random TTP, CWE and latest CVE. This based on DFIR reports."

	newUser, err := r.userService.NewUserAutoReg(ctx, update.Message.Chat.ID, update.Message.From.Username)
	if err != nil {
		log.Printf("user register error: %v", err)
	} else {
		log.Printf("User registered/loaded with ID: %d", newUser.ID)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.Printf("failed to send message to %d: %v", update.Message.Chat.ID, err)
	}
}

func (r *Router) Digest(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// TODO: inject digest service
	text := "sorry bro, not implemented"

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func (r *Router) EnableDigest(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	enabled := true
	err := r.userService.UpdateDigestSettings(ctx, update.Message.Chat.ID, &enabled, nil)

	msg := "Daily digest enabled."
	if err != nil {
		msg = "Failed to enable digest."
		log.Printf("enable digest error: %v", err)
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}

func (r *Router) DisableDigest(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	enabled := false
	err := r.userService.UpdateDigestSettings(ctx, update.Message.Chat.ID, &enabled, nil)

	msg := "Daily digest disabled."
	if err != nil {
		msg = "Failed to disable digest."
		log.Printf("disable digest error: %v", err)
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}

func (r *Router) SetTime(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	parts := strings.Split(update.Message.Text, " ")
	if len(parts) != 2 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Usage: /settime HH:MM (like /settime 14:30)",
		})
		return
	}

	timeParts := strings.Split(parts[1], ":")
	if len(timeParts) != 2 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Invalid time format. Use HH:MM.",
		})
		return
	}

	hour, err1 := strconv.ParseInt(timeParts[0], 10, 16)
	minute, err2 := strconv.ParseInt(timeParts[1], 10, 16)

	if err1 != nil || err2 != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Invalid numbers in time format.",
		})
		return
	}

	deliveryTime, err := domain.NewTimeOfDay(int16(hour), int16(minute))
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Invalid time: %v", err),
		})
		return
	}

	err = r.userService.UpdateDigestSettings(ctx, update.Message.Chat.ID, nil, &deliveryTime)
	if err != nil {
		log.Printf("set time error: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to save delivery time.",
		})
		return
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Delivery time set to %02d:%02d", deliveryTime.Hour, deliveryTime.Minute),
	})
}
