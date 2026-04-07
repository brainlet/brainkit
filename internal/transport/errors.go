package transport

import (
	"errors"
	"fmt"
)

type decodeFailure struct {
	topic string
	err   error
}

func (e *decodeFailure) Error() string {
	return fmt.Sprintf("%s: %v", e.topic, e.err)
}

func (e *decodeFailure) Unwrap() error {
	return e.err
}

func NewDecodeFailure(topic string, err error) error {
	if err == nil {
		return nil
	}
	return &decodeFailure{topic: topic, err: err}
}

func IsDecodeFailure(err error) bool {
	var target *decodeFailure
	return errors.As(err, &target)
}
