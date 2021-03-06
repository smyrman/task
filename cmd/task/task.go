package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-task/task"

	"github.com/spf13/pflag"
)

var (
	version = "master"
)

func main() {
	log.SetFlags(0)

	pflag.Usage = func() {
		fmt.Println(`task [target1 target2 ...]: Runs commands under targets like make.

Example: 'task hello' with the following 'Taskfile.yml' file will generate
an 'output.txt' file.
'''
hello:
  cmds:
    - echo "I am going to write a file named 'output.txt' now."
    - echo "hello" > output.txt
  generates:
    - output.txt
'''
`)
		pflag.PrintDefaults()
	}

	var (
		versionFlag bool
		init        bool
		force       bool
		watch       bool
	)

	pflag.BoolVar(&versionFlag, "version", false, "show Task version")
	pflag.BoolVarP(&init, "init", "i", false, "creates a new Taskfile.yml in the current folder")
	pflag.BoolVarP(&force, "force", "f", false, "forces execution even when the task is up-to-date")
	pflag.BoolVarP(&watch, "watch", "w", false, "enables watch of the given task")
	pflag.Parse()

	if versionFlag {
		log.Printf("Task version: %s\n", version)
		return
	}

	if init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := task.InitTaskfile(wd); err != nil {
			log.Fatal(err)
		}
		return
	}

	e := task.Executor{
		Force: force,
		Watch: watch,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := e.ReadTaskfile(); err != nil {
		log.Fatal(err)
	}

	args := pflag.Args()
	if len(args) == 0 {
		log.Println("task: No argument given, trying default task")
		args = []string{"default"}
	}

	if err := e.Run(args...); err != nil {
		log.Fatal(err)
	}
}
