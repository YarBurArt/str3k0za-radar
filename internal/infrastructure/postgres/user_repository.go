package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yarburart/str3k0za-radar/internal/domain"
)

var ErrNotFound = errors.New("entity not found")

type DeliveryTarget struct {
	TelegramID int64
	APTGroups  []string
}

type UserRepository struct {
	pool *pgxpool.Pool
	q    *Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
		q:    New(pool),
	}
}

// fetches a user and their preferences.
func (r *UserRepository) GetUserByTelegramID(ctx context.Context, telegramID int64) (domain.User, error) {
	sqlcUser, err := r.q.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}

	prefs, err := r.q.GetPreferencesByUserID(ctx, sqlcUser.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, fmt.Errorf("get preferences: %w", err)
	}
	domainUser, err := mapToDomainUser(sqlcUser, prefs)
	if err != nil {
		return domain.User{}, fmt.Errorf("digest pref time conversion: %w", err)
	}
	return domainUser, nil
}

// registers a new TelegramID user with default preferences
func (r *UserRepository) CreateUser(ctx context.Context, telegramID int64, username string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
			log.Printf("rollback failed with: %v", rollbackErr)
		}
	}()

	qtx := r.q.WithTx(tx)

	sqlcUser, err := qtx.CreateUser(ctx, CreateUserParams{
		TelegramID: telegramID,
		Username:   pgtype.Text{String: username, Valid: username != ""},
	})
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	_, err = qtx.CreatePreferences(ctx, CreatePreferencesParams{
		UserID:        sqlcUser.ID,
		AptGroups:     []string{},
		DigestEnabled: false,                     // enable by /enable
		DeliveryTime:  pgtype.Time{Valid: false}, // NULL until /settime or /enable default
	})
	if err != nil {
		return domain.User{}, fmt.Errorf("create preferences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("commit tx: %w", err)
	}

	return domain.User{
		ID:         sqlcUser.ID,
		TelegramID: sqlcUser.TelegramID,
		Username:   sqlcUser.Username.String,
		Prefs: domain.Preferences{
			APTGroups:       []string{},
			DigestEnabled:   false,
			DeliveryTime:    domain.TimeOfDay{},
			HasDeliveryTime: false,
		},
	}, nil
}

// a filter edit must not reset the digest schedule
func (r *UserRepository) UpdateAptGroupsOnly(ctx context.Context, telegramID int64, groups []string) error {
	userID, err := r.userIDByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}
	if groups == nil {
		groups = []string{}
	}
	if _, err := r.q.UpdateAptGroupsOnly(ctx, UpdateAptGroupsOnlyParams{
		UserID:    userID,
		AptGroups: groups,
	}); err != nil {
		return fmt.Errorf("update apt groups: %w", err)
	}
	return nil
}

// nil pointers fall through COALESCE, so one statement covers all three commands
func (r *UserRepository) UpdateDigestSettings(ctx context.Context, telegramID int64, enabled *bool, deliveryTime *domain.TimeOfDay) error {
	userID, err := r.userIDByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}

	params := UpdateDigestSettingsParams{UserID: userID}
	if enabled != nil {
		params.Enabled = pgtype.Bool{Bool: *enabled, Valid: true}
	}
	if deliveryTime != nil {
		params.DeliveryTime = toSQLTime(*deliveryTime)
	}

	if _, err := r.q.UpdateDigestSettings(ctx, params); err != nil {
		return fmt.Errorf("update digest settings: %w", err)
	}
	return nil
}

func (r *UserRepository) userIDByTelegramID(ctx context.Context, telegramID int64) (int64, error) {
	user, err := r.q.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("get user: %w", err)
	}
	return user.ID, nil
}

func (r *UserRepository) ListUsersForDelivery(ctx context.Context) ([]DeliveryTarget, error) {
	rows, err := r.q.ListUsersForDelivery(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users for delivery: %w", err)
	}

	targets := make([]DeliveryTarget, len(rows))
	for i, row := range rows {
		groups := row.AptGroups
		if groups == nil {
			groups = []string{}
		}
		targets[i] = DeliveryTarget{
			TelegramID: row.TelegramID,
			APTGroups:  groups,
		}
	}

	return targets, nil
}

func mapToDomainUser(u User, p Preference) (domain.User, error) {
	prefs, err := mapToDomainPrefs(p)
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:         u.ID,
		TelegramID: u.TelegramID,
		Username:   u.Username.String,
		Prefs:      prefs,
	}, nil
}

func mapToDomainPrefs(p Preference) (domain.Preferences, error) {
	groups := p.AptGroups
	if groups == nil {
		groups = []string{}
	}
	dtime, err := toDomainTime(p.DeliveryTime)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("convert delivery time: %w", err)
	}
	return domain.Preferences{
		APTGroups:       groups,
		DigestEnabled:   p.DigestEnabled,
		DeliveryTime:    dtime,
		HasDeliveryTime: p.DeliveryTime.Valid,
	}, nil
}

func toDomainTime(t pgtype.Time) (domain.TimeOfDay, error) {
	if !t.Valid {
		return domain.TimeOfDay{}, nil
	}

	// pgtype.Time stores microseconds since midnight
	totalSeconds := t.Microseconds / 1_000_000

	hour := totalSeconds / 3600
	// check on G115: integer overflow
	if hour > math.MaxInt16 {
		return domain.TimeOfDay{}, fmt.Errorf("hour overflow: %d", hour)
	}

	remainder := totalSeconds % 3600
	minute := remainder / 60
	if minute > math.MaxInt16 {
		return domain.TimeOfDay{}, fmt.Errorf("minute overflow: %d", minute)
	}

	return domain.TimeOfDay{
		//nolint:gosec // G115: fp, bounds are explicitly checked above
		Hour: int16(hour),
		//nolint:gosec // G115: fp, bounds are explicitly checked above
		Minute: int16(minute),
	}, nil
}

func toSQLTime(t domain.TimeOfDay) pgtype.Time {
	usec := (int64(t.Hour)*3600 + int64(t.Minute)*60) * 1_000_000
	return pgtype.Time{Microseconds: usec, Valid: true}
}
