package anserpc

import (
	"os"

	"github.com/chao77977/anserpc/util"
	log "github.com/inconshreveable/log15"
)

const (
	LvlCrit logLvl = iota
	LvlError
	LvlWarn
	LvlInfo
	LvlDebug
)

type logLvl int

type Logger interface {
	Crit(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
}

func newLogger(path string, filterLvl logLvl, silent bool) log.Logger {
	l := log.New("module", "anserpc")

	var h log.Handler
	if silent {
		if err := util.MakeFilePath(path); err != nil {
			panic(err)
		}

		h = log.Must.FileHandler(path, log.JsonFormat())
	} else {
		h = log.StreamHandler(os.Stderr, log.TerminalFormat())
	}

	l.SetHandler(log.LvlFilterHandler(log.Lvl(filterLvl), h))
	return l
}
