package config

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// time.Duration forces you to specify in millis, and does not support days
// see https://stackoverflow.com/questions/48050945/how-to-unmarshal-json-into-durations
type TTL time.Duration

func (l TTL) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(l).String())
}

func (l *TTL) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		if value == "" {
			*l = 0
			return nil
		}
		if before, ok := strings.CutSuffix(value, "d"); ok {
			days, err := strconv.Atoi(before)
			*l = TTL(time.Duration(days) * 24 * time.Hour)
			return err
		}
		if before, ok := strings.CutSuffix(value, "h"); ok {
			hours, err := strconv.Atoi(before)
			*l = TTL(time.Duration(hours) * time.Hour)
			return err
		}
		if before, ok := strings.CutSuffix(value, "m"); ok {
			minutes, err := strconv.Atoi(before)
			*l = TTL(time.Duration(minutes) * time.Minute)
			return err
		}
		if before, ok := strings.CutSuffix(value, "s"); ok {
			seconds, err := strconv.Atoi(before)
			*l = TTL(time.Duration(seconds) * time.Second)
			return err
		}
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*l = TTL(d)
		return nil
	default:
		return errors.New("invalid TTL")
	}
}
