package task

import (
	"io"
)

type nullWriter struct {
}

func (w *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var NullWriter = &nullWriter{}

type wrappedWriter struct {
	Child   io.Writer
	Prefix  []byte
	Postfix []byte
}

func (w *wrappedWriter) Write(p []byte) (n int, err error) {
	// TODO atomic write
	w.Child.Write(w.Prefix)
	n, err = w.Child.Write(p)
	w.Child.Write(w.Postfix)
	return
}

func MakeWrappedWriter(child io.Writer, prefix string, postfix string) io.Writer {
	return &wrappedWriter{Child: child, Prefix: []byte(prefix), Postfix: []byte(postfix)}
}

type multiWriter struct {
	Children []io.Writer
}

func (w *multiWriter) Write(p []byte) (n int, err error) {
	for _, child := range w.Children {
		child.Write(p)
	}
	// TODO how can we propagate errors?
	return len(p), nil
}

func MakeMultiWriter(children ...io.Writer) io.Writer {
	return &multiWriter{Children: children}
}
