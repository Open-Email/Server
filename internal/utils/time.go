package utils

import (
	"os"
	"time"
)

func TimestampNow() time.Time {
	return time.Now().UTC()
}

func ToRFC3339String(t time.Time) string {
	return t.Format(time.RFC3339)
}

func ParseRFC3339Time(timeStr string) (*time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, err
	}
	return &parsedTime, nil
}

func SetFileModificationTime(filePath string, t time.Time) error {
	err := os.Chtimes(filePath, t, t)
	if err != nil {
		return err
	}
	return nil
}

func RelativeTime(t time.Time) string {
	now := time.Now()

	// If the date is today, display just the time (HH:MM)
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}

	// If within the last 6 days, display the day of the week (e.g., Tue)
	if now.Sub(t) <= 6*24*time.Hour {
		return t.Format("Monday")
	}

	// If beyond 6 days, display the date as "DD.MM.YY"
	return t.Format("02.01.06")
}
