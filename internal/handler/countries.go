package handler

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// buildCountryKeyboard generates a 3x5 grid of countries with pagination
func (r *Router) buildCountryKeyboard(chatID int64) *models.InlineKeyboardMarkup {
	state := r.getState(chatID)

	// Mock, TODO: load this from infrastructure/mitre/apt_group_loader.go
	countries := []struct {
		code  string
		emoji string
	}{
		{"RU", "🇷🇺"},
		{"CN", "🇨🇳"},
		{"IR", "🇮🇷"},
		{"KP", "🇰🇵"},
		{"US", "🇺🇸"},
		{"GB", "🇬🇧"},
		{"FR", "🇫🇷"},
		{"DE", "🇩🇪"},
		{"IN", "🇮🇳"},
		{"PK", "🇵🇰"},
		{"VN", "🇻🇳"},
		{"KR", "🇰🇷"},
		{"JP", "🇯🇵"},
		{"BR", "🇧🇷"},
		{"IL", "🇮🇱"},
		{"UA", "🇺🇦"},
		{"TR", "🇹🇷"},
		{"SA", "🇸🇦"},
		{"EG", "🇪🇬"},
		{"NG", "🇳🇬"},
		{"IDK", "🏳️"}, // Grey/white flag for unknown origins
	}

	const rows = 5
	const cols = 3
	const itemsPerPage = rows * cols

	totalPages := (len(countries) + itemsPerPage - 1) / itemsPerPage
	if state.CurrentPage >= totalPages {
		state.CurrentPage = 0
	}

	startIdx := state.CurrentPage * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > len(countries) {
		endIdx = len(countries)
	}

	pageItems := endIdx - startIdx
	rowsCount := (pageItems + cols - 1) / cols
	keyboard := make([][]models.InlineKeyboardButton, 0, rowsCount+2)

	// 3x5 grid
	for i := startIdx; i < endIdx; i += cols {
		row := make([]models.InlineKeyboardButton, 0, cols)
		for j := 0; j < cols && i+j < endIdx; j++ {
			c := countries[i+j]
			icon := "[ ]"
			if state.SelectedCountries[c.code] {
				icon = "[+]"
			}
			row = append(row, models.InlineKeyboardButton{
				Text:         fmt.Sprintf("%s %s %s", icon, c.code, c.emoji),
				CallbackData: "country:toggle:" + c.code,
			})
		}
		keyboard = append(keyboard, row)
	}

	// pagination, there is too many APT)
	var navRow []models.InlineKeyboardButton
	if state.CurrentPage > 0 {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "Prev",
			CallbackData: "country:page:prev",
		})
	}

	navRow = append(navRow, models.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", state.CurrentPage+1, totalPages),
		CallbackData: "country:page:noop", // callback workaround to display text
	})

	if state.CurrentPage < totalPages-1 {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "Next",
			CallbackData: "country:page:next",
		})
	}
	keyboard = append(keyboard, navRow)

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: "Save & Done", CallbackData: "country:done"},
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func (r *Router) APTSourceCountryCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	if update.CallbackQuery.Message.Type != models.MaybeInaccessibleMessageTypeMessage {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "This message is no longer available.",
			ShowAlert:       true,
		})
		return
	}

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

	switch action {
	case "configure":
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        "Select source countries to track.\n\nTap a country to toggle it. Use Next/Prev to paginate.",
			ReplyMarkup: r.buildCountryKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit message: %v", err)
		}

	case "toggle":
		if len(parts) != 3 {
			return
		}
		country := parts[2]

		state := r.getState(chatID)
		state.SelectedCountries[country] = !state.SelectedCountries[country]

		status := "disabled"
		if state.SelectedCountries[country] {
			status = "enabled"
		}

		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            country + " filter " + status,
		})

		_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: r.buildCountryKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit country markup: %v", err)
		}

	case "page":
		if len(parts) != 3 {
			return
		}
		direction := parts[2]
		state := r.getState(chatID)

		switch direction {
		case "next":
			state.CurrentPage++
		case "prev":
			state.CurrentPage--
		}
		// "noop" is intentionally ignored

		_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: r.buildCountryKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit page markup: %v", err)
		}

	case "done":
		// TODO: quiery APT from data/ by state.SelectedCountries, then save to DB via UpdateProfile

		_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Country preferences saved",
		})
		if err != nil {
			log.Printf("failed to answer done callback: %v", err)
		}

		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "Country preferences saved. Use /digest to fetch your tailored report.",
		})
		if err != nil {
			log.Printf("failed to edit message on done: %v", err)
		}
	}
}
