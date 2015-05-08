package restapi

import (
	"strconv"
	"strings"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
)

// UnixToRFC3339 converts a Unix timestamp to a human readable format.
func UnixToRFC3339(ts int64) string {
	if ts > 0 {
		return time.Unix(ts, 0).UTC().Format(time.RFC3339)
	}
	return ""
}

func parseListQuery(r *rest.Request) models.ListParams {
	q := r.URL.Query()
	l := models.ListParams{}
	// Filters
	items := strings.Split(q.Get("filters"), ",")
	l.Filters = make(map[string]string, len(items))
	for _, item := range items {
		p := strings.SplitN(item, ":", 2)
		if len(p) == 2 {
			l.Filters[p[0]] = p[1]
		}
	}
	// Fields
	l.Fields = strings.Split(q.Get("fields"), ",")
	// Sort
	items = strings.Split(q.Get("sort"), ",")
	l.Sort = make(map[string]string, len(items))
	for _, item := range items {
		p := strings.SplitN(item, ":", 1)
		if len(p) == 2 {
			l.Sort[p[0]] = p[1]
		}
	}
	// Page
	var err error
	l.Page, err = strconv.Atoi(q.Get("page"))
	if err != nil || l.Page < 1 {
		l.Page = 1
	}
	// Limit
	l.Limit, err = strconv.Atoi(q.Get("limit"))
	if err != nil || l.Limit > 100 {
		l.Limit = 100
	}
	return l
}
