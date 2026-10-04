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

const helpText = "Bot is started.\n\n" +
	"Available commands:\n" +
	"- /enable  : enable the daily digest\n" +
	"- /disable : disable the daily digest\n" +
	"- /settime HH:MM  : set the delivery time (like /settime 18:30)\n" +
	"- /digest : generate a report right now, same content as the daily one\n" +
	"- /help : this message\n\n" +
	"Times are UTC. /settime 18:30 means 18:30 UTC, whatever timezone you are in."

func (r *Router) Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	r.sendHelp(ctx, b, update.Message.Chat.ID, nil)
}

func (r *Router) sendHelp(ctx context.Context, b *bot.Bot, chatID int64, keyboard models.ReplyMarkup) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        helpText,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.Printf("failed to send help to %d: %v", chatID, err)
	}
}

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

	newUser, err := r.userService.NewUserAutoReg(ctx, update.Message.Chat.ID, update.Message.From.Username)
	if err != nil {
		log.Printf("user register error: %v", err)
	} else {
		log.Printf("User registered/loaded with ID: %d", newUser.ID)
	}

	r.sendHelp(ctx, b, update.Message.Chat.ID, keyboard)
}

func (r *Router) Digest(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	text, err := r.digestService.GenerateDigestHTML(ctx, update.Message.Chat.ID)
	if err != nil {
		log.Printf("failed to generate /digest for user %d : %v", update.Message.Chat.ID, err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Sorry, some error in generating digest.\n Devs don't even know about that :) ",
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

func (r *Router) EnableDigest(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	enabled := true
	deliveryTime, err := r.userService.UpdateDigestSettings(ctx, update.Message.Chat.ID, &enabled, nil)

	msg := "Failed to enable digest."
	if err != nil {
		log.Printf("enable digest error: %v", err)
	} else {
		msg = fmt.Sprintf("Daily digest enabled, it will arrive at %02d:%02d UTC. Change with /settime HH:MM",
			deliveryTime.Hour, deliveryTime.Minute)
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
	if _, err := r.userService.UpdateDigestSettings(ctx, update.Message.Chat.ID, &enabled, nil); err != nil {
		log.Printf("disable digest error: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to disable digest, sorry",
		})
		return
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Daily digest disabled.",
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
			Text:   "Usage: /settime HH:MM in UTC (like /settime 18:30)",
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

	if _, err := r.userService.UpdateDigestSettings(ctx, update.Message.Chat.ID, nil, &deliveryTime); err != nil {
		log.Printf("set time error: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to save delivery time.",
		})
		return
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Delivery time set to %02d:%02d UTC", deliveryTime.Hour, deliveryTime.Minute),
	})
}
