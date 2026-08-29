package handler

import (
	"context"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// buildAPTKeyboard generates the inline keyboard based on the user's current selections.
func (r *Router) buildAPTKeyboard(chatID int64) *models.InlineKeyboardMarkup {
	state := r.getState(chatID)

	groups := []struct {
		id    string
		label string
	}{
		{"XDSpy", "XDSpy"},
		{"G0035", "G0035 (Dragonfly)"},
		{"APT41", "APT41 (Winnti)"},
		{"APT28", "APT28 (Fancy Bear)"},
		{"APT32", "APT32 (OceanLotus)"},
	}

	keyboard := make([][]models.InlineKeyboardButton, 0, len(groups)+1)
	for _, g := range groups {
		icon := "[ ]"
		if state.SelectedGroups[g.id] {
			icon = "[+]"
		}
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{Text: icon + "\t" + g.label, CallbackData: "apt:toggle:" + g.id},
		})
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: "Save & Done", CallbackData: "apt:done"},
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func (r *Router) APTCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	if update.CallbackQuery.Message.Type != models.MaybeInaccessibleMessageTypeMessage {
		// message is maybe deleted
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "This message is no longer available.",
			ShowAlert:       true,
		})
		return
	}
	// get callback message
	msg := update.CallbackQuery.Message.Message
	if msg == nil {
		return
	}

	chatID := msg.Chat.ID
	messageID := msg.ID
	data := update.CallbackQuery.Data

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return
	}

	action := parts[1]

	if action == "configure" {
		responseText := "Select APT groups to track in your threat model.\n\n" +
			"Tap a group to toggle itz inclusion in the filter."

		// static example, TODO: load by infrastructure/mitre/apt_group_loader.go , why not all in mitre btw
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        responseText,
			ReplyMarkup: r.buildAPTKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit message: %v", err)
		}
		return
	}

	if action == "toggle" && len(parts) == 3 {
		group := parts[2]

		state := r.getState(chatID)
		state.SelectedGroups[group] = !state.SelectedGroups[group]

		// TODO: inject UpdateProfile use case
		_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: r.buildAPTKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit message markup: %v", err)
		}
		return
	}

	if action == "done" {
		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Preferences saved",
		})
		if err != nil {
			log.Printf("failed to answer done callback: %v", err)
		}

		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "APT group preferences saved. Use /digest to fetch your tailored report.",
		})
		if err != nil {
			log.Printf("failed to edit message on done: %v", err)
		}
	}
}
