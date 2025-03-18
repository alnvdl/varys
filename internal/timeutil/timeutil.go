// Package timeutil provides time utility functions.
package timeutil

import "time"

func Now() int64 {
	return time.Now().Unix()
}

func HoursAgo(ts int64, h int) int64 {
	return ts - int64(h*3600)
}
