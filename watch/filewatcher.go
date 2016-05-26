package watch

import (
	"github.com/rjeczalik/notify"
	"path/filepath"
	"time"
)

type FileObserver interface {
	Begin()
	FileChanged(path string) bool
	Idle()
}

type relWrapper struct {
	basepath string
	child    FileObserver
}

func (w *relWrapper) Begin() {
	w.child.Begin()
}

func (w *relWrapper) FileChanged(path string) bool {
	path, err := filepath.Rel(w.basepath, path)
	if err != nil {
		return false
	}
	return w.child.FileChanged(path)
}

func (w *relWrapper) Idle() {
	w.child.Idle()
}

func Rel(basepath string, child FileObserver) FileObserver {
	return &relWrapper{basepath: basepath, child: child}
}

func debounce() (chan<- bool, <-chan bool) {
	beginDebounce := make(chan bool, 1)
	endDebounce := make(chan bool, 1)

	debounceDuration := time.Duration(1) * time.Second

	go func() {
		for {
			<-beginDebounce
			quiet := time.After(debounceDuration)
			waiting := true
			for waiting {
				select {
				case <-beginDebounce:
					quiet = time.After(debounceDuration)
				case <-quiet:
					waiting = false
				}
			}
			endDebounce <- true
		}
	}()
	return beginDebounce, endDebounce
}

func WatchFiles(path string, observer FileObserver) error {
	fileWatcher := make(chan notify.EventInfo, 1)
	err := notify.Watch(path, fileWatcher, notify.All)
	if err != nil {
		return err
	}
	defer notify.Stop(fileWatcher)
	beginDebounce, endDebounce := debounce()
	observer.Begin()
	for {
		select {
		case evt := <-fileWatcher:
			if observer.FileChanged(evt.Path()) {
				beginDebounce <- true
			}
		case <-endDebounce:
			observer.Idle()
		}
	}
}
