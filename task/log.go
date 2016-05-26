package task

import (
	"fmt"
	"github.com/mattn/go-colorable"
	"github.com/mgutz/ansi"
	"io"
	"strings"
	"time"
)

type TaskLog interface {
	LogInfo(format string, args ...interface{})
	LogError(format string, args ...interface{})
	BeginCapture() (io.Writer, io.Writer)
	EndCapture()
	CreateSubtask(name string) TaskLog
	Begin(t time.Time)
	End(t time.Time, d time.Duration)
}

type NullLog struct {
}

func (log *NullLog) LogInfo(format string, args ...interface{}) {
}

func (log *NullLog) LogError(format string, args ...interface{}) {
}

func (log *NullLog) BeginCapture() (io.Writer, io.Writer) {
	return NullWriter, NullWriter
}

func (log *NullLog) EndCapture() {
}

func (log *NullLog) CreateSubtask(name string) TaskLog {
	return log
}

func (log *NullLog) Begin(t time.Time) {
}

func (log *NullLog) End(t time.Time, d time.Duration) {
}

type FlatTextLogPrinter struct {
	Stdout io.Writer
	Stderr io.Writer
	Info   io.Writer
	Error  io.Writer
}

type FlatTextLog struct {
	Parent  TaskLog
	Path    []string
	Printer *FlatTextLogPrinter
}

func (log *FlatTextLog) LogInfo(format string, args ...interface{}) {
	// TODO check error?
	fmt.Fprintf(log.Printer.Info, format+"\n", args...)
}

func (log *FlatTextLog) LogError(format string, args ...interface{}) {
	// TODO check error?
	fmt.Fprintf(log.Printer.Error, format+"\n", args...)
}

func (log *FlatTextLog) BeginCapture() (io.Writer, io.Writer) {
	return log.Printer.Stdout, log.Printer.Stderr
}

func (log *FlatTextLog) EndCapture() {
}

func (log *FlatTextLog) CreateSubtask(name string) TaskLog {
	return &FlatTextLog{Parent: log, Path: append(log.Path, name), Printer: log.Printer}
}

func (log *FlatTextLog) Begin(t time.Time) {
	log.LogInfo(">>> %s", strings.Join(log.Path, "/"))
}

func (log *FlatTextLog) End(t time.Time, d time.Duration) {
	log.LogInfo("<<< %s %s", strings.Join(log.Path, "/"), d)
	log.LogInfo("")
}

func MakeAnsiColorWriter(child io.Writer, color string) io.Writer {
	return MakeWrappedWriter(child, color, ansi.Reset)
}

func MakeConsoleLog() TaskLog {
	// TODO detect if this is actually a console.
	stdout := colorable.NewColorableStdout()
	stderr := colorable.NewColorableStderr()
	return &FlatTextLog{
		Printer: &FlatTextLogPrinter{
			Stdout: stdout,
			Stderr: MakeAnsiColorWriter(stderr, ansi.Yellow),
			Info:   MakeAnsiColorWriter(stdout, ansi.Green),
			Error:  MakeAnsiColorWriter(stderr, ansi.Red),
		},
	}
}

type MultiLog struct {
	Children []TaskLog
}

func (log *MultiLog) LogInfo(format string, args ...interface{}) {
	for _, child := range log.Children {
		child.LogInfo(format, args...)
	}
}

func (log *MultiLog) LogError(format string, args ...interface{}) {
	for _, child := range log.Children {
		child.LogError(format, args...)
	}
}

func (log *MultiLog) BeginCapture() (io.Writer, io.Writer) {
	stdouts := make([]io.Writer, len(log.Children))
	stderrs := make([]io.Writer, len(log.Children))
	for i, child := range log.Children {
		stdouts[i], stderrs[i] = child.BeginCapture()
	}
	return MakeMultiWriter(stdouts...), MakeMultiWriter(stderrs...)
}

func (log *MultiLog) EndCapture() {
	for _, child := range log.Children {
		child.EndCapture()
	}
}

func (log *MultiLog) CreateSubtask(name string) TaskLog {
	new_children := make([]TaskLog, len(log.Children))
	for i, child := range log.Children {
		new_children[i] = child.CreateSubtask(name)
	}
	return &MultiLog{Children: new_children}
}

func (log *MultiLog) Begin(t time.Time) {
	for _, child := range log.Children {
		child.Begin(t)
	}
}

func (log *MultiLog) End(t time.Time, d time.Duration) {
	for _, child := range log.Children {
		child.End(t, d)
	}
}

func MakeMultiLog(children ...TaskLog) TaskLog {
	return &MultiLog{Children: children}
}
