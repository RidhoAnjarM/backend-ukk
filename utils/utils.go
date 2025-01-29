package utils

import (
	"fmt"
	"time"

)

func TimeAgo(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	switch {
	case duration < time.Minute:
		return "Baru saja"
	case duration < time.Hour:
		return formatDuration(duration.Minutes(), "menit")
	case duration < 24*time.Hour:
		return formatDuration(duration.Hours(), "jam")
	case duration < 30*24*time.Hour:
		return formatDuration(duration.Hours()/24, "hari")
	case duration < 12*30*24*time.Hour:
		return formatDuration(duration.Hours()/(24*30), "bulan")
	default:
		return formatDuration(duration.Hours()/(24*365), "tahun")
	}
}

func formatDuration(value float64, unit string) string {
	intValue := int(value)
	if intValue == 1 {
		return "1 " + unit + " yang lalu"
	}
	return fmt.Sprintf("%d %s yang lalu", intValue, unit)
}
