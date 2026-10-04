package domain

import "testing"

func TestNewTimeOfDay(t *testing.T) {
	t.Parallel()

	valid := []struct {
		hour, minute int16
	}{
		{0, 0},
		{0, 59},
		{23, 59},
		{18, 30},
	}
	for _, tc := range valid {
		got, err := NewTimeOfDay(tc.hour, tc.minute)
		if err != nil {
			t.Errorf("NewTimeOfDay(%d, %d) unexpected error: %v", tc.hour, tc.minute, err)
		}
		if got != (TimeOfDay{Hour: tc.hour, Minute: tc.minute}) {
			t.Errorf("NewTimeOfDay(%d, %d) = %+v", tc.hour, tc.minute, got)
		}
	}

	invalid := []struct {
		hour, minute int16
	}{
		{-1, 0},
		{24, 0},
		{0, -1},
		{0, 60},
		{-1, -1},
	}
	for _, tc := range invalid {
		if _, err := NewTimeOfDay(tc.hour, tc.minute); err == nil {
			t.Errorf("NewTimeOfDay(%d, %d) should fail", tc.hour, tc.minute)
		}
	}
}

func TestDefaultDeliveryTime(t *testing.T) {
	t.Parallel()

	// caller cannot mutate it for everyone else
	first := DefaultDeliveryTime()
	first.Hour = 0

	if second := DefaultDeliveryTime(); second.Hour != 18 || second.Minute != 30 {
		t.Errorf("DefaultDeliveryTime() = %+v after caller mutation", second)
	}
}
