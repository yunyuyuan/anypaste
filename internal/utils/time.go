package utils

import (
	"time"
)

func TimestampToTime(ms *int64) *time.Time {
	if ms == nil {
		return nil
	}
	t := time.UnixMilli(*ms)
	return &t
}

func TimeToTimestamp(time *time.Time) *int64 {
	var expiredAt *int64
	if time != nil {
		v := time.UnixMilli()
		expiredAt = &v
	}
	return expiredAt
}
