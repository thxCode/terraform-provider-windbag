package utils

import (
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"
)

// HandleCrash handles the crash with a default log handler.
func HandleCrash(additionalHandlers ...func(interface{})) {
	if r := recover(); r != nil {
		logPanic(r)
		for _, fn := range additionalHandlers {
			fn(r)
		}
	}
}

// HandleCrashSilent handles the crash without the default log handler.
func HandleCrashSilent(additionalHandlers ...func(interface{})) {
	if r := recover(); r != nil {
		for _, fn := range additionalHandlers {
			fn(r)
		}
	}
}

// logPanic logs the caller tree when a panic occurs (except in the special case of http.ErrAbortHandler).
func logPanic(r interface{}) {
	if r == http.ErrAbortHandler {
		// honor the http.ErrAbortHandler sentinel panic value:
		//   ErrAbortHandler is a sentinel panic value to abort a handler.
		//   While any panic from ServeHTTP aborts the response to the client,
		//   panicking with ErrAbortHandler also suppresses logging of a stack trace to the server's error log.
		return
	}

	// Same as stdlib http server code. Manually allocate stack trace buffer size
	// to prevent excessively large logs
	const size = 64 << 10
	stacktrace := make([]byte, size)
	stacktrace = stacktrace[:runtime.Stack(stacktrace, false)]
	if _, ok := r.(string); ok {
		logrus.Errorf("Observed a panic: %s\n%s", r, stacktrace)
	} else {
		logrus.Errorf("Observed a panic: %#v (%v)\n%s", r, r, stacktrace)
	}
}
