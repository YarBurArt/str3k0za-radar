package handler

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const aptItemsPerPage = 20

// buildAPTKeyboard generates the inline keyboard based on the user's current selections.
func (r *Router) buildAPTKeyboard(chatID int64) *models.InlineKeyboardMarkup {
	state := r.getState(chatID)

	// fetch current DB state of filters
	apts, err := r.userService.GetAPTKeyboardData(context.Background(), chatID)
	if err != nil {
		log.Printf("failed to get apt keyboard data: %v", err)
		return nil
	}

	// sync state with DB defaults on first menu open
	for _, apt := range apts {
		if _, exists := state.SelectedAPTGroups[apt.MitreID]; !exists {
			state.SelectedAPTGroups[apt.MitreID] = apt.IsSelected
		}
	}

	totalPages := (len(apts) + aptItemsPerPage - 1) / aptItemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if state.APTPage >= totalPages {
		state.APTPage = 0
	}

	startIdx := state.APTPage * aptItemsPerPage
	endIdx := startIdx + aptItemsPerPage
	if endIdx > len(apts) {
		endIdx = len(apts)
	}

	rowsCount := (endIdx - startIdx + 1) / 2
	keyboard := make([][]models.InlineKeyboardButton, 0, rowsCount+2)

	for i := startIdx; i < endIdx; i += 2 {
		row := make([]models.InlineKeyboardButton, 0, 2)

		apt1 := apts[i]
		icon1 := "[ ]"
		if state.SelectedAPTGroups[apt1.MitreID] {
			icon1 = "[+]"
		}
		row = append(row, models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s (%s)", icon1, apt1.Name, apt1.MitreID),
			CallbackData: "apt:toggle:" + apt1.MitreID,
		})

		if i+1 < endIdx {
			apt2 := apts[i+1]
			icon2 := "[ ]"
			if state.SelectedAPTGroups[apt2.MitreID] {
				icon2 = "[+]"
			}
			row = append(row, models.InlineKeyboardButton{
				Text:         fmt.Sprintf("%s %s (%s)", icon2, apt2.Name, apt2.MitreID),
				CallbackData: "apt:toggle:" + apt2.MitreID,
			})
		}
		keyboard = append(keyboard, row)
	}

	// pagination navigation row
	var navRow []models.InlineKeyboardButton
	if state.APTPage > 0 {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "Prev",
			CallbackData: "apt:page:prev",
		})
	}

	navRow = append(navRow, models.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", state.APTPage+1, totalPages),
		CallbackData: "apt:page:noop", // update ui without full callback logic
	})

	if state.APTPage < totalPages-1 {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "Next",
			CallbackData: "apt:page:next",
		})
	}
	keyboard = append(keyboard, navRow)

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: "Save & Done", CallbackData: "apt:done"},
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func (r *Router) APTCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}

	msg := update.CallbackQuery.Message.Message
	chatID := msg.Chat.ID
	messageID := msg.ID
	data := update.CallbackQuery.Data

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	switch action {
	case "configure":
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        "Select APT groups to track in your threat model",
			ReplyMarkup: r.buildAPTKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit message: %v", err)
		}

	case "toggle":
		if len(parts) != 3 {
			return
		}
		groupID := parts[2]
		state := r.getState(chatID)
		state.SelectedAPTGroups[groupID] = !state.SelectedAPTGroups[groupID]

		_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: r.buildAPTKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit message markup: %v", err)
		}

	case "page":
		if len(parts) != 3 {
			return
		}
		state := r.getState(chatID)
		if parts[2] == "next" {
			state.APTPage++
		} else if parts[2] == "prev" {
			state.APTPage--
		}

		_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: r.buildAPTKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit page markup: %v", err)
		}

	case "done":
		var finalSelected []string
		state := r.getState(chatID)
		for aptID, isSelected := range state.SelectedAPTGroups {
			if isSelected {
				finalSelected = append(finalSelected, aptID)
			}
		}

		err := r.userService.UpdateAPTGroups(ctx, chatID, finalSelected)
		if err != nil {
			log.Printf("failed to save apt groups: %v", err)
		} else {
			// next open fetches a fresh state from DB
			r.clearState(chatID)
		}

		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Preferences saved",
		})

		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "APT group preferences saved. Use /digest to fetch",
		})
	}
}
