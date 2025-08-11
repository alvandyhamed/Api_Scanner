package functions

import (
	"strings"
	"time"
)

func itoa(i int) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.TrimPrefix(strings.TrimSuffix(time.Duration(i).String(), "ns"), "0"), " ", ""))
}
