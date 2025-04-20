package groupify

import (
	"fmt"
	"regexp"
	"strings"
)

type StreetName string

var (
	_             fmt.Stringer = (*StreetName)(nil)
	spaceSquasher              = regexp.MustCompile(`\s+`)
)

// String implements fmt.Stringer.
func (s StreetName) String() string {
	return string(s)
}

func ParseStreetName(s string) StreetName {
	// Replace multiple spaces with a single space
	s = spaceSquasher.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), " ")
	return StreetName(s)
}
