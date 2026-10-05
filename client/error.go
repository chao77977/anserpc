package client

import (
	"encoding/json"
	"fmt"
)

// Error is a server-reported JSON-RPC error. It mirrors the server's error
// object (code / message / data) and implements the same accessor methods the
// server uses, so callers can type-assert on it.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("anserpc: error code %d", e.Code)
	}
	return fmt.Sprintf("anserpc: %s (code %d)", e.Message, e.Code)
}

// ErrorCode returns the JSON-RPC error code.
func (e *Error) ErrorCode() int { return e.Code }

// ErrorMessage returns the error message.
func (e *Error) ErrorMessage() string { return e.Message }

// ErrorData decodes the optional error data payload into v (which should be a
// pointer). It returns false if there is no data to decode.
func (e *Error) ErrorData(v interface{}) (bool, error) {
	if len(e.Data) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(e.Data, v); err != nil {
		return true, fmt.Errorf("anserpc client: decode error data: %w", err)
	}
	return true, nil
}

// AsError extracts a *Error from err if present, like errors.As.
func AsError(err error) (*Error, bool) {
	e, ok := err.(*Error)
	return e, ok
}
