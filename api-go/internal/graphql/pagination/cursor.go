package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const cursorPrefix = "offset:"

func EncodeOffsetCursor(offset int) string {
	payload := fmt.Sprintf("%s%d", cursorPrefix, offset)
	return base64.StdEncoding.EncodeToString([]byte(payload))
}

func DecodeOffsetCursor(cursor *string) (int, error) {
	if cursor == nil || strings.TrimSpace(*cursor) == "" {
		return 0, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(*cursor)
	if err != nil {
		return 0, fmt.Errorf("decode cursor: %w", err)
	}

	decodedCursor := string(decoded)
	if !strings.HasPrefix(decodedCursor, cursorPrefix) {
		return 0, fmt.Errorf("invalid cursor prefix")
	}

	offsetString := strings.TrimPrefix(decodedCursor, cursorPrefix)
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		return 0, fmt.Errorf("parse cursor offset: %w", err)
	}

	if offset < 0 {
		return 0, fmt.Errorf("cursor offset cannot be negative")
	}

	return offset, nil
}
