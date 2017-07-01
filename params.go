package task

import (
	"errors"
	"strings"
)

// Params represents a task params
type Params map[string]string

// Call represents a task call from another task
type Call struct {
	Name   string
	Params Params
}

var (
	// ErrParamDoesntContainsEqual is returned when a param doesn't have an equal (=) sign
	ErrParamDoesntContainsEqual = errors.New(`task: task param doesn't contains "="`)
)

// ParseCall parses the task call string format
// The format is the following:
//     ^task-name PARAM=value ANOTHER_PARAM=another-value
func ParseCall(source string) (c *Call, err error) {
	c = &Call{
		Name:   "",
		Params: Params{},
	}
	c.Name = strings.TrimPrefix(source, "^")
	spaceIdx := strings.IndexRune(c.Name, ' ')
	if spaceIdx != -1 {
		c.Name = c.Name[:spaceIdx]
	}

	source = strings.TrimPrefix(source, "^"+c.Name)
	source = strings.TrimPrefix(source, " ")
	if source == "" {
		return
	}

	// FIXME: need to handle if a param have a space in it
	paramsWithValues := strings.Split(source, " ")
	for _, pv := range paramsWithValues {
		equalIdx := strings.IndexRune(pv, '=')
		if equalIdx == -1 {
			err = ErrParamDoesntContainsEqual
			return
		}
		param, value := pv[:equalIdx], pv[equalIdx+1:]
		c.Params[param] = value
	}
	return
}
