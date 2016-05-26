package task

func Command(args ...string) *CommandTask {
	return &CommandTask{Args: args}
}
