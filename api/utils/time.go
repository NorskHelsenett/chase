package utils

import (
	"fmt"
	"time"
)

func ParseFlexibleTime(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05 -0700",
		// Add any other time formats you encounter here
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, timeStr)
		if err == nil {
			return parsedTime, nil
		}
	}

	// If we've tried all formats and none worked, return the last error
	return time.Time{}, fmt.Errorf("unable to parse time '%s': %v", timeStr, err)
}
