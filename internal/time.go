package re

import (
	"time"
)

func ReviewDue(requestedAt time.Time) time.Time {
	due := requestedAt.Add(24 * time.Hour)

	if due.Weekday() == time.Saturday {
		due = due.Add(48 * time.Hour)
	}
	if due.Weekday() == time.Sunday {
		due = due.Add(24 * time.Hour)
	}
	return due
}
