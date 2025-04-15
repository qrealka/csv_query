package groupify

import (
	"fmt"
	"strings"
)

type TreeSize int

const (
	TreeSizeNone TreeSize = iota
	TreeSizeShort
	TreeSizeTall
	treeSizeCount
)

var _ fmt.Stringer = (*TreeSize)(nil)

// String returns the string representation of a Section
func (s TreeSize) String() string {
	switch s {
	case TreeSizeShort:
		return "short"
	case TreeSizeTall:
		return "tall"
	default:
		return ""
	}
}

// Parse parses a string key and returns the corresponding TreeSize
func ParseTreeSize(s string) TreeSize {
	if strings.EqualFold(s, "short") {
		return TreeSizeShort
	} else if strings.EqualFold(s, "tall") {
		return TreeSizeTall
	}
	return TreeSizeNone
}
