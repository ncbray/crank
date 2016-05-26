package task

import (
	"os/exec"
	"strings"
)

func RunCommand(args []string, log TaskLog) bool {
	log.LogInfo("Running: %s", strings.Join(args, " "))

	cmd := exec.Command(args[0], args[1:]...)
	// TODO control the environment
	cmd.Stdout, cmd.Stderr = log.BeginCapture()
	err := cmd.Run()
	log.EndCapture()
	if err == nil {
		return true
	} else {
		log.LogError("Command failed: %s", err)
		return false
	}
}

type CommandTask struct {
	Args []string
}

func (task *CommandTask) Run(log TaskLog) bool {
	return RunCommand(task.Args, log)
}
