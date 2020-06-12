package log

import (
	"fmt"
	"os"
	"strings"

	"github.com/FleekHQ/space/core/env"
	"github.com/sirupsen/logrus"
)

var (
	log *logger
)

type logger struct {
	log *logrus.Logger
}

func init() {
	log = new("")
}

func New(env env.SpaceEnv) *logger {
	// TODO: check for log level in config and pass it to new
	return new("")
}

func new(logLevel string) *logger {
	logLevelConf := "Debug"
	level, err := logrus.ParseLevel(logLevelConf)
	if err != nil {
		level = logrus.DebugLevel
	}
	log = &logger{
		log: &logrus.Logger{
			Level:     level,
			Out:       os.Stdout,
			Formatter: &logrus.TextFormatter{},
		}}

	return log
}

// METHODS

func (l *logger) Info(msg string, tags ...string) {
	if l.log.Level < logrus.InfoLevel {
		return
	}

	l.log.WithFields(parseFields(tags...)).Info(msg)
}

func (l *logger) Printf(msg string, args ...interface{}) {
	if l.log.Level < logrus.InfoLevel {
		return
	}

	l.log.Printf(msg, args...)
}

func (l *logger) Debug(msg string, tags ...string) {
	if l.log.Level < logrus.DebugLevel {
		return
	}

	// l.log.WithFields(parseFields(tags...)).Debug(msg)
	l.log.Debug(msg, tags)
}

func (l *logger) Error(msg string, err error, tags ...string) {
	if l.log.Level < logrus.ErrorLevel {
		return
	}
	msg = fmt.Sprintf("%s -- ERROR -- %v", msg, err)
	// l.log.WithFields(parseFields(tags...)).Error(msg)
	l.log.Error(msg, tags)
}

func (l *logger) Fatal(err error) {
	l.Error(err.Error(), err)
}

// Functions

func Info(msg string, tags ...string) {
	log.Info(msg, tags...)
}

func Printf(msg string, args ...interface{}) {
	log.Printf(msg, args...)
}

func Debug(msg string, tags ...string) {
	log.Debug(msg, tags...)
}

func Error(msg string, err error, tags ...string) {
	log.Error(msg, err, tags...)
}

func Fatal(err error) {
	log.Fatal(err)
}

func parseFields(tags ...string) logrus.Fields {
	result := make(logrus.Fields, len(tags))

	for _, tag := range tags {
		els := strings.Split(tag, ":")
		if len(els) > 1 {
			result[strings.TrimSpace(els[0])] = strings.TrimSpace(els[1])
		}
	}
	return result
}
