package types

import "fmt"

// ErrCommandTopic is returned when an event is emitted on a command topic.
var ErrCommandTopic = fmt.Errorf("brainkit: topic is a command topic, not an event topic")
