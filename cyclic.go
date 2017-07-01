package task

import (
	"strings"
)

// HasCyclicDep checks if a task tree has any cyclic dependency
func (e *Executor) HasCyclicDep() bool {
	visits := make(map[string]struct{}, len(e.Tasks))

	var checkCyclicDep func(string, *Task) bool
	checkCyclicDep = func(name string, t *Task) bool {
		if _, ok := visits[name]; ok {
			return false
		}
		visits[name] = struct{}{}
		defer delete(visits, name)

		for _, d := range t.Deps {
			if strings.HasPrefix(d, "^") {
				call, err := ParseCall(d)
				if err != nil {
					return true
				}
				d = call.Name
			}
			if !checkCyclicDep(d, e.Tasks[d]) {
				return false
			}
		}
		return true
	}

	for k, v := range e.Tasks {
		if !checkCyclicDep(k, v) {
			return true
		}
	}
	return false
}
