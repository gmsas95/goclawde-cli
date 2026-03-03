package store

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON is a custom type for storing JSON in PostgreSQL that properly handles
// scanning from TEXT columns into json.RawMessage
type JSON json.RawMessage

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case string:
		*j = JSON(v)
	case []byte:
		*j = JSON(v)
	default:
		return errors.New("cannot scan type into JSON")
	}

	return nil
}

// MarshalJSON implements json.Marshaler
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON implements json.Unmarshaler
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		*j = JSON(data)
		return nil
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// ToRawMessage converts JSON to json.RawMessage
func (j JSON) ToRawMessage() json.RawMessage {
	return json.RawMessage(j)
}

// FromRawMessage converts json.RawMessage to JSON
func FromRawMessage(rm json.RawMessage) JSON {
	return JSON(rm)
}
