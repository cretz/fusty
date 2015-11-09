package model

import "strings"

func sanitizeRegex(regex string) string {
	// We implicitly prepend .* if ^ isn't there and same at end for lack of $
	// TODO: maybe this should check that the first char isn't a regex char instead?
	if !strings.HasPrefix(regex, "^") {
		regex = ".*" + regex
	}
	if !strings.HasSuffix(regex, "$") {
		regex += ".*"
	}
	return regex
}
