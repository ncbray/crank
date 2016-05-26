package task

type TaskDecl interface {
	Run(log TaskLog) bool
}
