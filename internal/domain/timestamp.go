package domain

import (
	"fmt"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

// CustomTime wraps time.Time for JSON timestamp serialization in "YYYY-MM-DD HH:mm:ss" UTC format.
type CustomTime struct {
	time.Time
}

func NewCustomTime(t time.Time) CustomTime {
	return CustomTime{Time: t.UTC()}
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if ct.IsZero() {
		return []byte(`""`), nil
	}
	formatted := fmt.Sprintf("%q", ct.UTC().Format(TimeFormat))
	return []byte(formatted), nil
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == `""` || str == "null" {
		ct.Time = time.Time{}
		return nil
	}
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	parsed, err := time.ParseInLocation(TimeFormat, str, time.UTC)
	if err != nil {
		return err
	}
	ct.Time = parsed.UTC()
	return nil
}
