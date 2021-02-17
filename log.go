package anserpc

import (
	"os"
	"sync"

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

var (
	_xlog Logger
	once  sync.Once
)

type logLvl int

type Logger interface {
	Crit(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
}

func newLogger(path string, filterLvl logLvl, silent bool) Logger {
	l := log.New()

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

func newSafeLogger(opt *logOpt) {
	once.Do(func() {
		if opt.logger != nil {
			_xlog = opt.logger
		} else if _xlog == nil {
			_xlog = newLogger(opt.path, opt.filterLvl, opt.silent)
		}
	})
}
