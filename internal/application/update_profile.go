package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yarburart/str3k0za-radar/internal/domain"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

type UserService struct {
	userRepo *postgres.UserRepository
}

func NewUserService(userRepo *postgres.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// register on /start, tg gives chatID and telegramID
func (s *UserService) NewUserAutoReg(ctx context.Context, telegramID int64, username string) (domain.User, error) {
	oldUser, isExistUserErr := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	// /start is send again
	if isExistUserErr == nil {
		return oldUser, nil
	}
	if !errors.Is(isExistUserErr, postgres.ErrNotFound) {
		return domain.User{}, fmt.Errorf("unexpected db error: %w", isExistUserErr)
	}
	newUser, err := s.userRepo.CreateUser(ctx, telegramID, username)
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user error: %w", err)
	}
	return newUser, nil
}

func (s *UserService) UpdatePreferences(ctx context.Context, telegramID int64, prefs domain.Preferences) (domain.User, error) {
	_, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return domain.User{}, fmt.Errorf("idk this user: %w", err)
	}
	updatedPrefs, err := s.userRepo.UpdatePreferences(ctx, telegramID, prefs)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to update preferences: %w", err)
	}
	return domain.User{
		TelegramID: telegramID,
		Prefs:      updatedPrefs,
	}, nil
}

// just wrapper for layer
func (s *UserService) GetPreferences(ctx context.Context, telegramID int64) (domain.User, error) {
	return s.userRepo.GetUserByTelegramID(ctx, telegramID)
}
