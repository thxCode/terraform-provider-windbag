package log

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	_ = iota
	lTrace
	lDebug
	lInfo
	lWarn
	lError
	lFatal
)

type logger struct {
	l *log.Logger
	v int
}

func (lg *logger) Traceln(message string) {
	if lg.v <= lTrace {
		lg.Println("\n\t[TRACE] ", message)
	}
}

func (lg *logger) Debugln(message string) {
	if lg.v <= lDebug {
		lg.Println("\n\t[DEBUG] ", message)
	}
}

func (lg *logger) Infoln(message string) {
	if lg.v <= lInfo {
		lg.Println("\n\t[INFO] ", message)
	}
}

func (lg *logger) Warnln(message string) {
	if lg.v <= lWarn {
		lg.Println("\n\t[WARN] ", message)
	}
}

func (lg *logger) Errorln(message string) {
	if lg.v <= lError {
		lg.Println("\n\t[ERROR] ", message)
	}
}

func (lg *logger) Fatalln(message string) {
	if lg.v <= lFatal {
		lg.Println("\n\t[FATAL] ", message)
		os.Exit(1)
	}
}

func (lg *logger) Println(message ...string) {
	lg.l.Println(message)
}

var l *logger

func init() {
	var v int
	switch strings.ToLower(os.Getenv("WINDBAG_LOG")) {
	case "fatal":
		v = lFatal
	case "error":
		v = lError
	case "warn", "warning":
		v = lWarn
	case "debug":
		v = lDebug
	case "trace":
		v = lTrace
	default:
		v = lInfo
	}
	l = &logger{
		l: log.New(os.Stderr, "", 0),
		v: v,
	}
}

func Tracef(format string, args ...interface{}) {
	l.Traceln(fmt.Sprintf(format, args...))
}

func Debugf(format string, args ...interface{}) {
	l.Debugln(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...interface{}) {
	l.Infoln(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...interface{}) {
	l.Warnln(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...interface{}) {
	l.Errorln(fmt.Sprintf(format, args...))
}

func Warn(message string) {
	l.Warnln(message)
}

func Fatal(message string) {
	l.Fatalln(message)
}
