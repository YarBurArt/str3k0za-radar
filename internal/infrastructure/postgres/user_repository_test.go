package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yarburart/str3k0za-radar/internal/domain"
)

// testRepo returns a repository on a scratch database, or skips. The database
// name must contain "test" because the fixtures truncate the tables.
func testRepo(t *testing.T) *UserRepository {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if !strings.Contains(dsn, "test") {
		t.Fatalf("TEST_DATABASE_URL must point at a database whose name contains \"test\", got %q", dsn)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `TRUNCATE users, preferences RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return NewUserRepository(pool)
}

func seedUser(t *testing.T, repo *UserRepository, telegramID int64, enabled bool, deliveryTime *domain.TimeOfDay) {
	t.Helper()

	ctx := context.Background()
	if _, err := repo.CreateUser(ctx, telegramID, "tester"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := repo.UpdateDigestSettings(ctx, telegramID, &enabled, deliveryTime); err != nil {
		t.Fatalf("update digest settings: %v", err)
	}
}

func tod(t *testing.T, hour, minute int16) *domain.TimeOfDay {
	t.Helper()

	v, err := domain.NewTimeOfDay(hour, minute)
	if err != nil {
		t.Fatalf("build time %d:%d: %v", hour, minute, err)
	}
	return &v
}

// dbNowMinute reads the database clock, which is what the delivery window is
// matched against.
func dbNowMinute(t *testing.T, repo *UserRepository) domain.TimeOfDay {
	t.Helper()

	var hour, minute int16
	row := repo.pool.QueryRow(context.Background(),
		`SELECT EXTRACT(HOUR FROM date_trunc('minute', NOW() AT TIME ZONE 'UTC'))::int,
		        EXTRACT(MINUTE FROM date_trunc('minute', NOW() AT TIME ZONE 'UTC'))::int`)
	if err := row.Scan(&hour, &minute); err != nil {
		t.Fatalf("read db clock: %v", err)
	}
	return domain.TimeOfDay{Hour: hour, Minute: minute}
}

func telegramIDs(targets []DeliveryTarget) map[int64]bool {
	got := make(map[int64]bool, len(targets))
	for _, target := range targets {
		got[target.TelegramID] = true
	}
	return got
}

func TestListUsersForDeliveryWindow(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	now := dbNowMinute(t, repo)
	prev := domain.TimeOfDay{}
	if now.Minute == 0 {
		prev = domain.TimeOfDay{Hour: 23, Minute: 59}
	} else {
		prev = domain.TimeOfDay{Hour: now.Hour, Minute: now.Minute - 1}
	}
	future := domain.TimeOfDay{Hour: (now.Hour + 2) % 24, Minute: now.Minute}

	seedUser(t, repo, 1, true, &now)
	// prev is the exclusive lower bound: it was served by the previous tick and
	// must not be served twice
	seedUser(t, repo, 2, true, &prev)
	seedUser(t, repo, 3, false, &now)
	seedUser(t, repo, 4, true, nil)
	seedUser(t, repo, 5, true, &future)

	targets, err := repo.ListUsersForDelivery(ctx)
	if err != nil {
		t.Fatalf("list users for delivery: %v", err)
	}
	got := telegramIDs(targets)

	if !got[1] {
		t.Error("user 1 should be due (enabled, delivery_time is the current minute)")
	}
	for _, id := range []int64{2, 3, 4, 5} {
		if got[id] {
			t.Errorf("user %d should not be due", id)
		}
	}
}

// TestListUsersForDeliveryWholeMinute is the regression guard for the old
// `delivery_time = CURRENT_TIME(0)` predicate, which matched only during the
// first second of a minute because CURRENT_TIME(0) keeps its seconds.
func TestListUsersForDeliveryWholeMinute(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	seedUser(t, repo, 1, true, nil)
	seed := func() domain.TimeOfDay {
		t.Helper()
		m := dbNowMinute(t, repo)
		if err := repo.UpdateDigestSettings(ctx, 1, nil, &m); err != nil {
			t.Fatalf("set delivery time: %v", err)
		}
		return m
	}

	cur := seed()
	const samples = 20
	hits := 0
	for i := range samples {
		// a rollover moves the window past the seeded value, so re-seed
		if m := dbNowMinute(t, repo); m != cur {
			cur = seed()
		}

		targets, err := repo.ListUsersForDelivery(ctx)
		if err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
		if len(targets) == 1 {
			hits++
		}
		time.Sleep(100 * time.Millisecond)
	}

	if hits < samples*3/4 {
		t.Errorf("delivery window matched %d/%d samples, want the whole minute", hits, samples)
	}
}

func TestUpdateDigestSettingsIsPartial(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	seedUser(t, repo, 1, true, tod(t, 9, 30))
	if err := repo.UpdateAptGroupsOnly(ctx, 1, []string{"G0001"}); err != nil {
		t.Fatalf("update apt groups: %v", err)
	}

	no := false
	if err := repo.UpdateDigestSettings(ctx, 1, &no, nil); err != nil {
		t.Fatalf("disable: %v", err)
	}

	user, err := repo.GetUserByTelegramID(ctx, 1)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Prefs.DigestEnabled {
		t.Error("digest should be disabled")
	}
	if !user.Prefs.HasDeliveryTime || user.Prefs.DeliveryTime != *tod(t, 9, 30) {
		t.Errorf("delivery_time should stay 09:30, got %+v (has=%v)", user.Prefs.DeliveryTime, user.Prefs.HasDeliveryTime)
	}
	if len(user.Prefs.APTGroups) != 1 {
		t.Errorf("apt_groups should survive a digest update, got %v", user.Prefs.APTGroups)
	}

	if err := repo.UpdateDigestSettings(ctx, 1, nil, tod(t, 21, 5)); err != nil {
		t.Fatalf("set time: %v", err)
	}
	user, err = repo.GetUserByTelegramID(ctx, 1)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Prefs.DigestEnabled {
		t.Error("setting only the time must not re-enable the digest")
	}
	if user.Prefs.DeliveryTime != *tod(t, 21, 5) {
		t.Errorf("delivery_time should be 21:05, got %+v", user.Prefs.DeliveryTime)
	}
}

func TestUpdateAptGroupsOnlyPreservesDigestSettings(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	seedUser(t, repo, 1, true, tod(t, 7, 15))

	if err := repo.UpdateAptGroupsOnly(ctx, 1, nil); err != nil {
		t.Fatalf("update apt groups: %v", err)
	}

	user, err := repo.GetUserByTelegramID(ctx, 1)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !user.Prefs.DigestEnabled || user.Prefs.DeliveryTime != *tod(t, 7, 15) {
		t.Errorf("digest settings changed, got %+v", user.Prefs)
	}
	if user.Prefs.APTGroups == nil || len(user.Prefs.APTGroups) != 0 {
		t.Errorf("apt_groups should be an empty non-nil slice, got %#v", user.Prefs.APTGroups)
	}
}

func TestUnknownUserIsNotFound(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	if err := repo.UpdateAptGroupsOnly(ctx, 404, []string{"G0001"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateAptGroupsOnly: got %v, want ErrNotFound", err)
	}
	if err := repo.UpdateDigestSettings(ctx, 404, nil, nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateDigestSettings: got %v, want ErrNotFound", err)
	}
	if _, err := repo.GetUserByTelegramID(ctx, 404); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetUserByTelegramID: got %v, want ErrNotFound", err)
	}
}
