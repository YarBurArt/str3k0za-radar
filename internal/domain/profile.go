package domain

import "fmt"

type TimeOfDay struct {
	Hour   int16
	Minute int16
}

func NewTimeOfDay(hour, minute int16) (TimeOfDay, error) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return TimeOfDay{},
			fmt.Errorf("invalid time: %d:%d", hour, minute)
	}
	return TimeOfDay{Hour: hour, Minute: minute}, nil
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
