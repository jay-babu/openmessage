// Package sqliteuri formats filesystem paths as SQLite file URIs.
package sqliteuri

import (
	"net/url"
	"strings"
)

func File(path, query string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if len(path) >= 3 && path[1:3] == ":/" &&
		((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) {
		path = "/" + path
	} else if strings.HasPrefix(path, "//") {
		// Preserve UNC paths without using a SQLite URI authority.
		path = "//" + path
	}
	return (&url.URL{Scheme: "file", Path: path, RawQuery: query}).String()
}
