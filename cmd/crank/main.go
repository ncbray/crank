package main

import (
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/ncbray/cmdline"
	"github.com/ncbray/crank/task"
	"github.com/ncbray/crank/watch"
	"github.com/ncbray/crank/workgraph"
	"log"
	"os"
	"path/filepath"
)

type PathMatch struct {
	Glob   string
	Invert bool
}

type CascadingPathMatch struct {
	Matches []PathMatch
}

func (m *CascadingPathMatch) Match(path string) bool {
	ok := false
	for _, match := range m.Matches {
		// TODO check error
		matched, _ := doublestar.Match(match.Glob, path)
		if matched {
			ok = !match.Invert
		}
	}
	return ok
}

type FileManager struct {
	Graph *workgraph.WorkGraph
	Tasks []*TaskWrapper
}

func (fm *FileManager) FileChanged(path string) bool {
	matched := false
	for _, task := range fm.Tasks {
		if task.Match.Match(path) {
			fm.Graph.Invalidate(task.Node)
			matched = true
		}
	}
	return matched
}

type TaskWrapper struct {
	Task  task.TaskDecl
	Log   task.TaskLog
	Node  *workgraph.Node
	Match *CascadingPathMatch
}

func (w *TaskWrapper) Invalidated() {
}

func (w *TaskWrapper) Run() bool {
	return w.Task.Run(w.Log)
}

type IncrementalTaskRunner struct {
	FileManager *FileManager
	Graph       *workgraph.WorkGraph
}

func (runner *IncrementalTaskRunner) Run() {
	fmt.Println("Running...")
	runner.Graph.Run()
	fmt.Println("Done...")
	fmt.Println()
}

func createWorkGraph(subpath string, logger task.TaskLog) *IncrementalTaskRunner {
	// TODO be sensitive to directory renames and deletetion.
	// TODO ignore .git/

	all_go := &CascadingPathMatch{
		Matches: []PathMatch{
			{Glob: "**/*.go"},
		},
	}

	all_go_no_tests := &CascadingPathMatch{
		Matches: []PathMatch{
			{Glob: "**/*.go"},
			{Glob: "**/*_test.go", Invert: true},
		},
	}

	g := &workgraph.WorkGraph{}
	tasks := []*TaskWrapper{}

	attach := func(g *workgraph.WorkGraph, wrapper *TaskWrapper) *TaskWrapper {
		wrapper.Node = g.CreateNode(wrapper)
		tasks = append(tasks, wrapper)
		return wrapper
	}

	vet := attach(g, &TaskWrapper{
		Task:  task.Command("go", "vet", subpath),
		Log:   logger,
		Match: all_go,
	})
	test := attach(g, &TaskWrapper{
		Task:  task.Command("go", "test", subpath),
		Log:   logger,
		Match: all_go,
	})
	install := attach(g, &TaskWrapper{
		Task:  task.Command("go", "install", subpath),
		Log:   logger,
		Match: all_go_no_tests,
	})
	g.CreateEdge(vet.Node, test.Node, true)
	g.CreateEdge(test.Node, install.Node, false)
	g.MarkLive(install.Node)

	return &IncrementalTaskRunner{
		FileManager: &FileManager{
			Graph: g,
			Tasks: tasks,
		},
		Graph: g,
	}
}

func (runner *IncrementalTaskRunner) Begin() {
	runner.Run()
}

func (runner *IncrementalTaskRunner) FileChanged(path string) bool {
	path = filepath.ToSlash(path)

	// Do not watch git files.
	is_git, _ := doublestar.Match("**/.git/**", path)
	if is_git {
		return false
	}

	fmt.Println("changed", path)
	return runner.FileManager.FileChanged(path)
}

func (runner *IncrementalTaskRunner) Idle() {
	runner.Run()
}

func doGoWorkflow(workspaceDir string, packageRoot string) {
	packageDir := filepath.Join("src", packageRoot)

	subpath := filepath.Join(packageRoot, "...")

	err := watch.WatchFiles(
		filepath.Join(packageDir, "..."),
		watch.Rel(workspaceDir, createWorkGraph(subpath, task.MakeConsoleLog())),
	)
	if err != nil {
		panic(err)
	}
}

func main() {
	workspace_dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	goPkg := &cmdline.FilePath{
		Root:      "src",
		MustExist: true,
	}

	var pkg string

	app := cmdline.MakeApp("crank_worker")
	app.RequiredArgs([]*cmdline.Argument{
		{
			Name:  "package",
			Value: goPkg.Set(&pkg),
		},
	})
	app.Run(os.Args[1:])

	doGoWorkflow(workspace_dir, pkg)
}
