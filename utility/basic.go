package utility

import (
	"strconv"
	"strings"
	"time"
)

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func GenerateQuery(modelID string) string {
	hashPrefix := modelID[:4]
	now := time.Now().UnixNano()
	timeComponent := now % (36 * 36)
	timeStr := strconv.FormatInt(timeComponent, 36)
	if len(timeStr) < 2 {
		timeStr = "0" + timeStr
	}
	return hashPrefix + timeStr
}
