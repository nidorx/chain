package chain

import (
	"errors"
	"net/http"
)

// AlreadySentError Error raised when trying to modify or send an already sent response
var AlreadySentError = errors.New("the response was already sent")

type ResponseWriterSpy struct {
	http.ResponseWriter
	wrote           bool
	writeCalled     bool
	hooksBeforeSend []func()
	hooksAfterSend  []func()
}

func (w *ResponseWriterSpy) WriteHeader(status int) {
	w.runBeforeHook()
	w.ResponseWriter.WriteHeader(status)
	if !w.writeCalled {
		w.runAfterHook()
	}
}

func (w *ResponseWriterSpy) Write(b []byte) (int, error) {
	w.writeCalled = true
	w.runBeforeHook()
	i, err := w.ResponseWriter.Write(b)
	w.runAfterHook()
	return i, err
}

// beforeSend Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (w *ResponseWriterSpy) beforeSend(callback func()) error {
	if w.wrote {
		return AlreadySentError
	}
	w.hooksBeforeSend = append(w.hooksBeforeSend, callback)
	return nil
}

// beforeSend Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (w *ResponseWriterSpy) afterSend(callback func()) error {
	if w.wrote {
		return AlreadySentError
	}
	w.hooksAfterSend = append(w.hooksAfterSend, callback)
	return nil
}

func (w *ResponseWriterSpy) runBeforeHook() {
	if w.wrote {
		return
	}
	w.wrote = true
	if w.hooksBeforeSend != nil {
		for i := len(w.hooksBeforeSend) - 1; i >= 0; i-- {
			w.hooksBeforeSend[i]()
		}
	}
	w.hooksBeforeSend = nil
}

func (w *ResponseWriterSpy) runAfterHook() {
	if w.hooksAfterSend != nil {
		for i := len(w.hooksAfterSend) - 1; i >= 0; i-- {
			w.hooksAfterSend[i]()
		}
	}
	w.hooksAfterSend = nil
}
