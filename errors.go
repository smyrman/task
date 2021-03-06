package task

import (
	"errors"
	"fmt"
)

var (
	// ErrCyclicDependencyDetected is returned when a cyclic dependency was found in the Taskfile
	ErrCyclicDependencyDetected = errors.New("task: cyclic dependency detected")
	// ErrTaskfileAlreadyExists is returned on creating a Taskfile if one already exists
	ErrTaskfileAlreadyExists = errors.New("task: A Taskfile already exists")
)

type taskFileNotFound struct {
	taskFile string
}

func (err taskFileNotFound) Error() string {
	return fmt.Sprintf(`task: No task file found (is it named "%s"?). Use "task --init" to create a new one`, err.taskFile)
}

type taskNotFoundError struct {
	taskName string
}

func (err *taskNotFoundError) Error() string {
	return fmt.Sprintf(`task: Task "%s" not found`, err.taskName)
}

type taskRunError struct {
	taskName string
	err      error
}

func (err *taskRunError) Error() string {
	return fmt.Sprintf(`task: Failed to run task "%s": %v`, err.taskName, err.err)
}

type cyclicDepError struct {
	taskName string
}

func (err *cyclicDepError) Error() string {
	return fmt.Sprintf(`task: Cyclic dependency of task "%s" detected`, err.taskName)
}

type cantWatchNoSourcesError struct {
	taskName string
}

func (err *cantWatchNoSourcesError) Error() string {
	return fmt.Sprintf(`task: Can't watch task "%s" because it has no specified sources`, err.taskName)
}
