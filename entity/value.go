package entity

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Value wraps a field value and provides type conversion helpers.
type Value struct {
	Raw any
}

// String returns the value as a string.
func (v Value) String() string {
	if v.Raw == nil {
		return ""
	}
	return fmt.Sprintf("%v", v.Raw)
}

// Int returns the value as an int.
func (v Value) Int() (int, error) {
	i, ok := v.Raw.(int64)
	if !ok {
		return 0, errors.Errorf("value is not an int64: %T", v.Raw)
	}
	return int(i), nil
}

// Float returns the value as a float64.
func (v Value) Float() (float64, error) {
	f, ok := v.Raw.(float64)
	if !ok {
		return 0, errors.Errorf("value is not a float64: %T", v.Raw)
	}
	return f, nil
}

// Bool returns the value as a bool.
func (v Value) Bool() (bool, error) {
	b, ok := v.Raw.(bool)
	if !ok {
		return false, errors.Errorf("value is not a bool: %T", v.Raw)
	}
	return b, nil
}

// Time returns the value as a time.Time.
func (v Value) Time() (time.Time, error) {
	t, ok := v.Raw.(time.Time)
	if !ok {
		return time.Time{}, errors.Errorf("value is not a time.Time: %T", v.Raw)
	}
	return t, nil
}

// Line represents a single log entry as an ordered list of values.
// The order corresponds to the fields returned by Store.Fields().
type Line []Value
