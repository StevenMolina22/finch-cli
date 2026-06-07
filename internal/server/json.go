package server

import "encoding/json"

// jsonUnmarshal is a thin wrapper around encoding/json kept here so callers
// that only need parsing can import encoding/json through this package.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
