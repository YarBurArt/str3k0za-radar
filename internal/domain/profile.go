package domain

import "fmt"

type TimeOfDay struct {
	hour   int16
	minute int16
}

func NewTimeOfDay(hour, minute int16) (TimeOfDay, error) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return TimeOfDay{}, fmt.Errorf("invalid time: %d:%d", hour, minute)
	}
	return TimeOfDay{hour: hour, minute: minute}, nil
}

func (t TimeOfDay) Hour() int16 {
	return t.hour
}

func (t TimeOfDay) Minute() int16 {
	return t.minute
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.hour, t.minute)
}

type Preferences struct {
	APTGroups     []string
	DigestEnabled bool
	DeliveryTime  TimeOfDay
}
type User struct {
	ID         int64
	TelegramID int64
	Username   string
	Prefs      Preferences
}
