package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/execext"

	"golang.org/x/sync/errgroup"
)

const (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile"
)

// Executor executes a Taskfile
type Executor struct {
	Tasks Tasks
	Dir   string
	Force bool
	Watch bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	watchingFiles map[string]struct{}
}

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Cmds      []string
	Deps      []string
	Desc      string
	Sources   []string
	Generates []string
	Status    []string
	Dir       string
	Vars      map[string]string
	Set       string
	Env       map[string]string
}

// Run runs Task
func (e *Executor) Run(args ...string) error {
	if e.HasCyclicDep() {
		return ErrCyclicDependencyDetected
	}

	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}

	// check if given tasks exist
	for _, a := range args {
		if _, ok := e.Tasks[a]; !ok {
			// FIXME: move to the main package
			e.printExistingTasksHelp()
			return &taskNotFoundError{taskName: a}
		}
	}

	if e.Watch {
		if err := e.watchTasks(args...); err != nil {
			return err
		}
		return nil
	}

	for _, a := range args {
		if err := e.RunTask(context.Background(), &Call{Name: a}); err != nil {
			return err
		}
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call *Call) error {
	t, ok := e.Tasks[call.Name]
	if !ok {
		return &taskNotFoundError{call.Name}
	}

	if err := e.runDeps(ctx, call); err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := e.isTaskUpToDate(ctx, call)
		if err != nil {
			return err
		}
		if upToDate {
			e.printfln(`task: Task "%s" is up to date`, call.Name)
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.runCommand(ctx, call, i); err != nil {
			return &taskRunError{call.Name, err}
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, call *Call) error {
	g, ctx := errgroup.WithContext(ctx)
	t := e.Tasks[call.Name]

	for _, d := range t.Deps {
		dep := d

		g.Go(func() error {
			dep, err := e.ReplaceVariables(call, dep)
			if err != nil {
				return err
			}

			if strings.HasPrefix(dep, "^") {
				call, err := ParseCall(dep)
				if err != nil {
					return err
				}
				return e.RunTask(ctx, call)
			}
			return e.RunTask(ctx, &Call{Name: dep})
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (e *Executor) isTaskUpToDate(ctx context.Context, call *Call) (bool, error) {
	t := e.Tasks[call.Name]

	if len(t.Status) > 0 {
		return e.isUpToDateStatus(ctx, call)
	}
	return e.isUpToDateTimestamp(ctx, call)
}

func (e *Executor) isUpToDateStatus(ctx context.Context, call *Call) (bool, error) {
	t := e.Tasks[call.Name]

	environ, err := e.getEnviron(call)
	if err != nil {
		return false, err
	}
	dir, err := e.getTaskDir(call)
	if err != nil {
		return false, err
	}

	for _, s := range t.Status {
		err = execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: s,
			Dir:     dir,
			Env:     environ,
		})
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (e *Executor) isUpToDateTimestamp(ctx context.Context, call *Call) (bool, error) {
	t := e.Tasks[call.Name]

	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false, nil
	}

	dir, err := e.getTaskDir(call)
	if err != nil {
		return false, err
	}

	sources, err := e.ReplaceSliceVariables(call, t.Sources)
	if err != nil {
		return false, err
	}
	generates, err := e.ReplaceSliceVariables(call, t.Generates)
	if err != nil {
		return false, err
	}

	sourcesMaxTime, err := getPatternsMaxTime(dir, sources)
	if err != nil || sourcesMaxTime.IsZero() {
		return false, nil
	}

	generatesMinTime, err := getPatternsMinTime(dir, generates)
	if err != nil || generatesMinTime.IsZero() {
		return false, nil
	}

	return generatesMinTime.After(sourcesMaxTime), nil
}

func (e *Executor) runCommand(ctx context.Context, call *Call, i int) error {
	t := e.Tasks[call.Name]

	c, err := e.ReplaceVariables(call, t.Cmds[i])
	if err != nil {
		return err
	}

	if strings.HasPrefix(c, "^") {
		call, err := ParseCall(c)
		if err != nil {
			return err
		}
		return e.RunTask(ctx, call)
	}

	dir, err := e.getTaskDir(call)
	if err != nil {
		return err
	}

	envs, err := e.getEnviron(call)
	if err != nil {
		return err
	}
	opts := &execext.RunCommandOptions{
		Context: ctx,
		Command: c,
		Dir:     dir,
		Env:     envs,
		Stdin:   e.Stdin,
		Stderr:  e.Stderr,
	}

	if t.Set == "" {
		e.println(c)
		opts.Stdout = e.Stdout
		if err = execext.RunCommand(opts); err != nil {
			return err
		}
	} else {
		buff := bytes.NewBuffer(nil)
		opts.Stdout = buff
		if err = execext.RunCommand(opts); err != nil {
			return err
		}
		os.Setenv(t.Set, strings.TrimSpace(buff.String()))
	}
	return nil
}

func (e *Executor) getTaskDir(call *Call) (string, error) {
	t := e.Tasks[call.Name]

	exeDir, err := e.ReplaceVariables(call, e.Dir)
	if err != nil {
		return "", err
	}
	taskDir, err := e.ReplaceVariables(call, t.Dir)
	if err != nil {
		return "", err
	}

	return filepath.Join(exeDir, taskDir), nil
}

func (e *Executor) getEnviron(call *Call) ([]string, error) {
	t := e.Tasks[call.Name]

	if t.Env == nil {
		return nil, nil
	}

	envs := os.Environ()

	for k, v := range t.Env {
		env, err := e.ReplaceVariables(call, fmt.Sprintf("%s=%s", k, v))
		if err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}
