package handler

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ISO codes to flag emojis
var countryEmojis = map[string]string{
	"RU": "🇷🇺", "CN": "🇨🇳", "IR": "🇮🇷", "KP": "🇰🇵", "US": "🇺🇸",
	"GB": "🇬🇧", "FR": "🇫🇷", "DE": "🇩🇪", "IN": "🇮🇳", "PK": "🇵🇰",
	"VN": "🇻🇳", "KR": "🇰🇷", "JP": "🇯🇵", "BR": "🇧🇷", "IL": "🇮🇱",
	"UA": "🇺🇦", "TR": "🇹🇷", "SA": "🇸🇦", "EG": "🇪🇬", "NG": "🇳🇬",
	"BE": "🇧🇪", "CO": "🇨🇴", "LB": "🇱🇧", "RO": "🇷🇴", "UK": "🇬🇧",
}

func (r *Router) buildCountryKeyboard(chatID int64) *models.InlineKeyboardMarkup {
	state := r.getState(chatID)
	countries := r.userService.GetAvailableCountries(context.Background())

	const cols = 3
	const itemsPerPage = 15

	totalPages := (len(countries) + itemsPerPage - 1) / itemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if state.CountryPage >= totalPages {
		state.CountryPage = 0
	}

	startIdx := state.CountryPage * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > len(countries) {
		endIdx = len(countries)
	}

	pageItems := endIdx - startIdx
	rowsCount := (pageItems + cols - 1) / cols
	keyboard := make([][]models.InlineKeyboardButton, 0, rowsCount+2)

	for i := startIdx; i < endIdx; i += cols {
		row := make([]models.InlineKeyboardButton, 0, cols)
		for j := 0; j < cols && i+j < endIdx; j++ {
			c := countries[i+j]
			icon := "[ ]"
			if state.SelectedCountries[c] {
				icon = "[+]"
			}

			// flag only if it exists in map
			label := fmt.Sprintf("%s %s", icon, c)
			if emoji, ok := countryEmojis[c]; ok {
				label = fmt.Sprintf("%s %s %s", icon, c, emoji)
			}

			row = append(row, models.InlineKeyboardButton{
				Text:         label,
				CallbackData: "country:toggle:" + c,
			})
		}
		keyboard = append(keyboard, row)
	}

	var navRow []models.InlineKeyboardButton
	if state.CountryPage > 0 {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "Prev",
			CallbackData: "country:page:prev",
		})
	}

	navRow = append(navRow, models.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", state.CountryPage+1, totalPages),
		CallbackData: "country:page:noop",
	})

	if state.CountryPage < totalPages-1 {
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
			Text:        "Select APT source countries to track",
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
		state := r.getState(chatID)
		if parts[2] == "next" {
			state.CountryPage++
		} else if parts[2] == "prev" {
			state.CountryPage--
		}

		_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: r.buildCountryKeyboard(chatID),
		})
		if err != nil {
			log.Printf("failed to edit page markup: %v", err)
		}

	case "done":
		state := r.getState(chatID)
		var selectedCountries []string
		for country, isSelected := range state.SelectedCountries {
			if isSelected {
				selectedCountries = append(selectedCountries, country)
			}
		}

		err := r.userService.UpdateCountries(ctx, chatID, selectedCountries)
		if err != nil {
			log.Printf("failed to save country preferences: %v", err)
		} else {
			// clear state from memory after save
			r.clearState(chatID)
		}

		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Country preferences saved",
		})

		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "Country preferences saved. Use /digest to fetch",
		})
	}
}
