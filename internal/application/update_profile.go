package application

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/yarburart/str3k0za-radar/internal/domain"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

type UserService struct {
	userRepo *postgres.UserRepository
	graph    *domain.AttackGraph // for user preferences setup
}

func NewUserService(userRepo *postgres.UserRepository, attackGraph *domain.AttackGraph) *UserService {
	return &UserService{
		userRepo: userRepo,
		graph:    attackGraph,
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

type APTKeyboardData struct {
	MitreID    string
	Name       string
	IsSelected bool
}

func (s *UserService) GetAPTKeyboardData(ctx context.Context, telegramID int64) ([]APTKeyboardData, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("get user for apt keyboard: %w", err)
	}
	// optimize selection checks by map
	selectedMap := make(map[string]struct{}, len(user.Prefs.APTGroups))
	for _, id := range user.Prefs.APTGroups {
		selectedMap[id] = struct{}{}
	}

	var result []APTKeyboardData
	if s.graph != nil {
		// keys to slice for stable sorting, for pagination
		ids := make([]string, 0, len(s.graph.APTs))
		for id := range s.graph.APTs {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		result = make([]APTKeyboardData, 0, len(ids))
		for _, id := range ids {
			apt := s.graph.APTs[id]
			_, isSelected := selectedMap[apt.MitreID]
			result = append(result, APTKeyboardData{
				MitreID:    apt.MitreID,
				Name:       apt.Name,
				IsSelected: isSelected,
			})
		}
	}
	return result, nil
}

// extracts unique countries present in the loaded graph
func (s *UserService) GetAvailableCountries(ctx context.Context) []string {
	if s.graph == nil {
		return []string{}
	}

	unique := make(map[string]struct{})
	for _, apt := range s.graph.APTs {
		// TODO: separate field for unknown
		if apt.SourceCountry != "" && apt.SourceCountry != "UNKNOWN" {
			unique[apt.SourceCountry] = struct{}{}
		}
	}

	result := make([]string, 0, len(unique))
	for c := range unique {
		// 2 A-Z country code validation
		if len(c) == 2 && c[0] >= 'A' && c[0] <= 'Z' && c[1] >= 'A' && c[1] <= 'Z' {
			result = append(result, c)
		}
	}
	sort.Strings(result) // for UI pagination
	return result
}

func (s *UserService) UpdateAPTGroups(ctx context.Context, telegramID int64, selectedAPTIDs []string) error {
	// empty slice for consistent unmarshaling
	if selectedAPTIDs == nil {
		selectedAPTIDs = []string{}
	}
	// a filter edit must not reset the digest schedule
	return s.userRepo.UpdateAptGroupsOnly(ctx, telegramID, selectedAPTIDs)
}

func (s *UserService) UpdateCountries(ctx context.Context, telegramID int64, selectedCountries []string) error {
	var resolvedAPTIDs []string
	if s.graph != nil && len(selectedCountries) > 0 {
		countrySet := make(map[string]struct{})
		for _, c := range selectedCountries {
			countrySet[c] = struct{}{}
		}

		// resolve country choices into APT mitre IDs
		for _, apt := range s.graph.APTs {
			if apt.SourceCountry != "UNKNOWN" {
				if _, ok := countrySet[apt.SourceCountry]; ok {
					resolvedAPTIDs = append(resolvedAPTIDs, apt.MitreID)
				}
			}
		}
	}

	// a filter edit must not reset the digest schedule
	return s.userRepo.UpdateAptGroupsOnly(ctx, telegramID, resolvedAPTIDs)
}

// returns the time now in effect so /enable can confirm it without a re-read
func (s *UserService) UpdateDigestSettings(ctx context.Context, telegramID int64, enabled *bool, deliveryTime *domain.TimeOfDay) (domain.TimeOfDay, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return domain.TimeOfDay{}, fmt.Errorf("user not found: %w", err)
	}

	if enabled == nil && deliveryTime == nil {
		return user.Prefs.DeliveryTime, nil
	}

	effective := user.Prefs.DeliveryTime
	hasTime := user.Prefs.HasDeliveryTime
	switch {
	case deliveryTime != nil:
		effective, hasTime = *deliveryTime, true
	case enabled != nil && *enabled && !hasTime:
		// enabling without a time would otherwise never be scheduled
		effective, hasTime = domain.DefaultDeliveryTime(), true
	}

	// a nil pointer keeps delivery_time NULL, so disabling a user who never picked
	// a time does not invent 00:00 and make it look deliberate later
	var timeArg *domain.TimeOfDay
	if hasTime {
		timeArg = &effective
	}

	if err := s.userRepo.UpdateDigestSettings(ctx, telegramID, enabled, timeArg); err != nil {
		return domain.TimeOfDay{}, err
	}
	return effective, nil
}

// just wrapper for layer
func (s *UserService) GetPreferences(ctx context.Context, telegramID int64) (domain.User, error) {
	return s.userRepo.GetUserByTelegramID(ctx, telegramID)
}
