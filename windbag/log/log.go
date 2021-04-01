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

func (lg *logger) Tracef(format string, args ...interface{}) {
	if lg.v <= lTrace {
		lg.l.Printf("[TRACE] %s\n", fmt.Sprintf(format, args...))
	}
}

func (lg *logger) Debugf(format string, args ...interface{}) {
	if lg.v <= lDebug {
		lg.l.Printf("[DEBUG] %s\n", fmt.Sprintf(format, args...))
	}
}

func (lg *logger) Infof(format string, args ...interface{}) {
	if lg.v <= lInfo {
		lg.l.Printf("[INFO] %s\n", fmt.Sprintf(format, args...))
	}
}

func (lg *logger) Warnf(format string, args ...interface{}) {
	if lg.v <= lWarn {
		lg.l.Printf("[WARN] %s\n", fmt.Sprintf(format, args...))
	}
}

func (lg *logger) Errorf(format string, args ...interface{}) {
	if lg.v <= lError {
		lg.l.Printf("[ERROR] %s\n", fmt.Sprintf(format, args...))
	}
}

func (lg *logger) Fatalf(format string, args ...interface{}) {
	if lg.v <= lFatal {
		lg.l.Printf("[FATAL] %s\n", fmt.Sprintf(format, args...))
	}
}

func (lg *logger) Traceln(v ...interface{}) {
	if lg.v <= lTrace {
		lg.l.Printf("[TRACE] %s", fmt.Sprintln(v...))
	}
}

func (lg *logger) Debugln(v ...interface{}) {
	if lg.v <= lDebug {
		lg.l.Printf("[DEBUG] %s", fmt.Sprintln(v...))
	}
}

func (lg *logger) Infoln(v ...interface{}) {
	if lg.v <= lInfo {
		lg.l.Printf("[INFO] %s", fmt.Sprintln(v...))
	}
}

func (lg *logger) Warnln(v ...interface{}) {
	if lg.v <= lWarn {
		lg.l.Printf("[WARN] %s", fmt.Sprintln(v...))
	}
}

func (lg *logger) Errorln(v ...interface{}) {
	if lg.v <= lError {
		lg.l.Printf("[ERROR] %s", fmt.Sprintln(v...))
	}
}

func (lg *logger) Fatalln(v ...interface{}) {
	if lg.v <= lFatal {
		lg.l.Printf("[FATAL] %s", fmt.Sprintln(v...))
		os.Exit(1)
	}
}

var l *logger

func init() {
	// configure default logger
	log.SetFlags(0)
	log.SetPrefix("[sdk] ")

	// configure system logger
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
		l: log.New(os.Stderr, "[plg] ", 0),
		v: v,
	}
}

func Tracef(format string, args ...interface{}) {
	l.Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	l.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	l.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	l.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	l.Fatalf(format, args...)
}

func Traceln(v ...interface{}) {
	l.Traceln(v...)
}

func Debugln(v ...interface{}) {
	l.Debugln(v...)
}

func Infoln(v ...interface{}) {
	l.Infoln(v...)
}

func Warnln(v ...interface{}) {
	l.Warnln(v...)
}

func Errorln(v ...interface{}) {
	l.Errorln(v...)
}

func Fatalln(v ...interface{}) {
	l.Fatalln(v...)
}
