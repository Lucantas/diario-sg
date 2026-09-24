package domain

import (
	"fmt"
	"strconv"
)

func FormatBRL(cents int64) string {
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "." + whole[i:]
	}
	return fmt.Sprintf("R$ %s,%02d", whole, cents%100)
}
