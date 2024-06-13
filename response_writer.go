package chain

import (
	"errors"
	"log/slog"
	"net/http"
)

// ErrAlreadySent Error raised when trying to modify or send an already sent response
var ErrAlreadySent = errors.New("the response was already sent")

type ResponseWriterSpy struct {
	http.ResponseWriter
	writeStarted           bool
	writeCalled            bool
	writeHeaderCalled      bool
	beforeWriteHeaderHooks []func()
	afterWriteHooks        []func()
}

func (w *ResponseWriterSpy) WriteHeader(status int) {
	w.writeHeaderCalled = true
	w.execBeforeWriteHeaderHooks()
	w.ResponseWriter.WriteHeader(status)
}

func (w *ResponseWriterSpy) Write(b []byte) (int, error) {
	w.writeCalled = true
	if !w.writeStarted {
		w.execBeforeWriteHeaderHooks()
	}
	return w.ResponseWriter.Write(b)
}

// beforeWriteHeader Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (w *ResponseWriterSpy) beforeWriteHeader(callback func()) error {
	if w.writeStarted {
		return ErrAlreadySent
	}
	w.beforeWriteHeaderHooks = append(w.beforeWriteHeaderHooks, callback)
	return nil
}

// afterWrite Registers a callback to be invoked before the response is sent.
//
// Callbacks are invoked in the reverse order they are defined (callbacks defined first are invoked last).
func (w *ResponseWriterSpy) afterWrite(callback func()) error {
	if w.writeStarted {
		return ErrAlreadySent
	}
	w.afterWriteHooks = append(w.afterWriteHooks, func() {
		defer func() {
			// no panic
			if r := recover(); r != nil {
				slog.Warn("[chain] panic occured in a after write hook", slog.Any("panic", r))
			}
		}()
		callback()
	})
	return nil
}

func (w *ResponseWriterSpy) execBeforeWriteHeaderHooks() {
	if w.writeStarted {
		return
	}
	w.writeStarted = true
	if w.beforeWriteHeaderHooks != nil {
		for i := len(w.beforeWriteHeaderHooks) - 1; i >= 0; i-- {
			w.beforeWriteHeaderHooks[i]()
		}
	}
	w.beforeWriteHeaderHooks = nil
}

// execAfterWriteHooksCalledByRouter called by router.ServeHTTP
func (w *ResponseWriterSpy) execAfterWriteHooksCalledByRouter() {
	if w.afterWriteHooks != nil {
		for i := len(w.afterWriteHooks) - 1; i >= 0; i-- {
			w.afterWriteHooks[i]()
		}
	}
	w.afterWriteHooks = nil
}
