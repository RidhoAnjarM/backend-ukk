package utils

import (
	"fmt"
	"time"
)

func TimeAgo(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	if duration > 7*24*time.Hour {
		return t.Format("02 Jan 2006") 
	}

	switch {
	case duration < time.Minute:
		return "Baru saja"
	case duration < time.Hour:
		return formatDuration(duration.Minutes(), "menit")
	case duration < 24*time.Hour:
		return formatDuration(duration.Hours(), "jam")
	case duration < 7*24*time.Hour:
		return formatDuration(duration.Hours()/24, "hari")
	}
	return ""
}

func formatDuration(value float64, unit string) string {
	intValue := int(value)
	if intValue == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %s", intValue, unit)
}
