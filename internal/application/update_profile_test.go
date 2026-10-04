package application

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yarburart/str3k0za-radar/internal/domain"
	"github.com/yarburart/str3k0za-radar/internal/infrastructure/postgres"
)

// the name must contain "test" cuz the fixtures truncate the tables
func testService(t *testing.T) (*UserService, *postgres.UserRepository) {
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

	repo := postgres.NewUserRepository(pool)
	return NewUserService(repo, nil), repo
}

// a user who never picked a time has delivery_time NULL, so /disable must leave
// it NULL: materializing 00:00 makes the next /enable treat midnight as a choice
func TestDisableKeepsUnsetDeliveryTimeNull(t *testing.T) {
	svc, repo := testService(t)
	ctx := context.Background()

	const telegramID = int64(555001)
	if _, err := repo.CreateUser(ctx, telegramID, "notime"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	disabled := false
	if _, err := svc.UpdateDigestSettings(ctx, telegramID, &disabled, nil); err != nil {
		t.Fatalf("disable: %v", err)
	}

	user, err := repo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if user.Prefs.HasDeliveryTime {
		t.Fatalf("delivery_time = %02d:%02d, want still NULL after /disable",
			user.Prefs.DeliveryTime.Hour, user.Prefs.DeliveryTime.Minute)
	}
}

// the regression: /disable then /enable reported midnight instead of the default
func TestEnableAfterDisableUsesDefaultTime(t *testing.T) {
	svc, repo := testService(t)
	ctx := context.Background()

	const telegramID = int64(555002)
	if _, err := repo.CreateUser(ctx, telegramID, "cycle"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	disabled := false
	if _, err := svc.UpdateDigestSettings(ctx, telegramID, &disabled, nil); err != nil {
		t.Fatalf("disable: %v", err)
	}

	enabled := true
	effective, err := svc.UpdateDigestSettings(ctx, telegramID, &enabled, nil)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}

	want := domain.DefaultDeliveryTime()
	if effective != want {
		t.Fatalf("effective time = %02d:%02d, want default %02d:%02d",
			effective.Hour, effective.Minute, want.Hour, want.Minute)
	}

	user, err := repo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !user.Prefs.DigestEnabled {
		t.Fatal("digest_enabled = false, want true")
	}
	if user.Prefs.DeliveryTime != want {
		t.Fatalf("persisted time = %02d:%02d, want %02d:%02d",
			user.Prefs.DeliveryTime.Hour, user.Prefs.DeliveryTime.Minute, want.Hour, want.Minute)
	}
}

// an explicit time is a real choice and must survive a disable/enable cycle
func TestDisablePreservesChosenTime(t *testing.T) {
	svc, repo := testService(t)
	ctx := context.Background()

	const telegramID = int64(555003)
	if _, err := repo.CreateUser(ctx, telegramID, "chosen"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	chosen, err := domain.NewTimeOfDay(21, 45)
	if err != nil {
		t.Fatalf("build time: %v", err)
	}
	if _, err := svc.UpdateDigestSettings(ctx, telegramID, nil, &chosen); err != nil {
		t.Fatalf("settime: %v", err)
	}

	disabled := false
	if _, err := svc.UpdateDigestSettings(ctx, telegramID, &disabled, nil); err != nil {
		t.Fatalf("disable: %v", err)
	}

	enabled := true
	effective, err := svc.UpdateDigestSettings(ctx, telegramID, &enabled, nil)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if effective != chosen {
		t.Fatalf("effective time = %02d:%02d, want the chosen %02d:%02d",
			effective.Hour, effective.Minute, chosen.Hour, chosen.Minute)
	}
}

// apt edits are on their own statement and must not disturb the schedule
func TestUpdateAPTGroupsKeepsDigestSchedule(t *testing.T) {
	svc, repo := testService(t)
	ctx := context.Background()

	const telegramID = int64(555004)
	if _, err := repo.CreateUser(ctx, telegramID, "filter"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	enabled := true
	if _, err := svc.UpdateDigestSettings(ctx, telegramID, &enabled, nil); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if err := svc.UpdateAPTGroups(ctx, telegramID, []string{"G0016"}); err != nil {
		t.Fatalf("set apt groups: %v", err)
	}

	user, err := repo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !user.Prefs.DigestEnabled {
		t.Fatal("digest_enabled = false, want true after a filter edit")
	}
	if user.Prefs.DeliveryTime != domain.DefaultDeliveryTime() {
		t.Fatalf("delivery time = %02d:%02d, want the default %02d:%02d",
			user.Prefs.DeliveryTime.Hour, user.Prefs.DeliveryTime.Minute,
			domain.DefaultDeliveryTime().Hour, domain.DefaultDeliveryTime().Minute)
	}
	if len(user.Prefs.APTGroups) != 1 || user.Prefs.APTGroups[0] != "G0016" {
		t.Fatalf("apt_groups = %v, want [G0016]", user.Prefs.APTGroups)
	}
}
