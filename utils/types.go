package utils

import "regexp"

// ParseStatus describes current status of the parsing
type ParseStatus uint8

// enums for parsing
const (
	StatusNotReady ParseStatus = iota
	StatusOk
	StatusWarn
	StatusFail
)

func (ps ParseStatus) String() string {
	switch ps {
	case StatusNotReady:
		return "not ready"
	case StatusOk:
		return "ok"
	case StatusWarn:
		return "with errors"
	case StatusFail:
		return "failed"
	default:
		return "unknown"
	}
}

var filterBashDialogsRegex = regexp.MustCompile(`(?m)^([\w\d]+:\s*)(.+)$`)

func filterBashDialog(s string) string {
	return filterBashDialogsRegex.ReplaceAllString(s, "$2")
}
