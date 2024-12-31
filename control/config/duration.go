package config

import (
	"encoding/json"
	"time"
)

// inspired by:
// - https://gist.github.com/ulexxander/a678baa2ae3454f9516a1cd7450ed6be
// - https://stackoverflow.com/questions/48050945/how-to-unmarshal-json-into-durations
// Duration is a wrapper around time.Duration that implements custom JSON marshaling
type Duration time.Duration

// MarshalJSON implements the json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	val, err := time.ParseDuration(str)
	*d = Duration(val)
	return err
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return time.Duration(d).String()
}
