package main

import (
	"fmt"
	"github.com/ncbray/cmdline"
	"github.com/ncbray/crank/watch"
	"os"
	"os/exec"
	"strings"
)

type stayfresh struct {
	executable string
	args       []string
	cmd        *exec.Cmd
}

func (s *stayfresh) printableCmd() string {
	return strings.Join(append([]string{s.executable}, s.args...), " ")
}

func (s *stayfresh) run() {
	fmt.Println("stayfresh run:", s.printableCmd())
	fmt.Println()
	s.cmd = exec.Command(s.executable, s.args...)
	s.cmd.Stdin = os.Stdin
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	s.cmd.Start()
}

func (s *stayfresh) Begin() {
	s.run()
}

func (s *stayfresh) FileChanged(path string) bool {
	return true
}

func (s *stayfresh) Idle() {
	s.cmd.Process.Kill()
	s.cmd.Wait()
	fmt.Println()
	fmt.Println("stayfresh kill:", s.printableCmd())
	s.run()
}

func main() {
	var executable string
	args := []string{}

	app := cmdline.MakeApp("stayfresh")
	executableFile := &cmdline.FilePath{
		MustExist: true,
	}
	app.RequiredArgs([]*cmdline.Argument{
		{
			Name:  "executable",
			Value: executableFile.Set(&executable),
		},
	})
	app.ExcessArguments(&cmdline.Argument{
		Name: "args",
		Value: cmdline.String.Call(func(value string) {
			args = append(args, value)
		}),
	})
	app.Run(os.Args[1:])

	err := watch.WatchFiles(executable, &stayfresh{
		executable: executable,
		args:       args,
	})

	if err != nil {
		panic(nil)
	}
}
