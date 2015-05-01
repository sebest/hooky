package restapi

import "time"

// UnixToRFC3339 converts a Unix timestamp to a human readable format.
func UnixToRFC3339(ts int64) string {
	if ts > 0 {
		return time.Unix(ts, 0).UTC().Format(time.RFC3339)
	}
	return ""
}
