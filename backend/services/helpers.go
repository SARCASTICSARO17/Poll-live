package services

import (
	"encoding/json"
	"strconv"
)

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func parseInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}