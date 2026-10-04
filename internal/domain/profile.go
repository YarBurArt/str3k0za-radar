package domain

import "fmt"

// enabling without a time would otherwise never be scheduled
func DefaultDeliveryTime() TimeOfDay {
	return TimeOfDay{Hour: 18, Minute: 30}
}

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
	APTGroups       []string
	DigestEnabled   bool
	DeliveryTime    TimeOfDay
	HasDeliveryTime bool // delivery_time column is not NULL
}

type User struct {
	ID         int64
	TelegramID int64
	Username   string
	Prefs      Preferences
}
