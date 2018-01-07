package logging

import (
	"os"

	"github.com/op/go-logging"
)

const (
	DEBUG = iota
	WARNING
	INFO
	ERROR
)

var log *logging.Logger

var backend *logging.LogBackend
var fileBackend *logging.LogBackend

func init() {
	log = logging.MustGetLogger("TicketsCache")
	var format = logging.MustStringFormatter(
		`%{color}%{time:2006-01-02 04:05:06.000} %{shortfunc} %{level:.4s} %{color:reset} %{message}`,
	)
	backend := logging.NewLogBackend(os.Stdout, "", 0)
}

func Debug(msg string) {
	log.Debugf("")
}
