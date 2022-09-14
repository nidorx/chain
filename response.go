package chain

import (
	"errors"
	"net/http"
)

// AlreadySentError Error raised when trying to modify or send an already sent response
var AlreadySentError = errors.New("the response was already sent")

type ResponseWriterSpy struct {
	http.ResponseWriter
	wrote               bool
	beforeSendCallbacks []func()
}

func (w *ResponseWriterSpy) WriteHeader(status int) {
	w.runBeforeSend()
	w.ResponseWriter.WriteHeader(status)
}

func (w *ResponseWriterSpy) Write(b []byte) (int, error) {
	w.runBeforeSend()
	return w.ResponseWriter.Write(b)
}

// registerBeforeSend Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (w *ResponseWriterSpy) registerBeforeSend(callback func()) error {
	if w.wrote {
		return AlreadySentError
	}
	w.beforeSendCallbacks = append(w.beforeSendCallbacks, callback)
	return nil
}

func (w *ResponseWriterSpy) runBeforeSend() {
	if w.wrote {
		return
	}
	w.wrote = true
	if w.beforeSendCallbacks != nil {
		for i := len(w.beforeSendCallbacks) - 1; i >= 0; i-- {
			w.beforeSendCallbacks[i]()
		}
	}
	w.beforeSendCallbacks = nil
}
