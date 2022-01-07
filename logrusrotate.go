// This package adds the log rotation functionality for the apps which use logrus,
// and which need to run for a long time (daemons).

package logrusrotate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type Logrotate struct {
	// only perform rotation when we're active
	active bool

	// arguments to the ctor
	log      *logrus.Logger
	basePath string
	interval int64
	sizeMb   int64 // TODO: currently unused
	format   string

	// when true (that is, format contains '8'), don't use time format as a suffix.
	// instead, use an incrementing number
	useIncrement      bool
	incrementedSuffix int64

	// handle to the actual file
	f *os.File

	// the periodical ticker channel
	ticker        *time.Ticker
	appDoneCh     chan int
	forceRotateCh chan time.Time

	// add to the log file messages about the scheduled (or forced) rotations
	verbose bool
}

const (
	Second int64 = 1
	Minute       = Second * 60
	Hour         = Minute * 60
	Day          = Hour * 24

	parsedIncrementFormat = 8
)

func New(log *logrus.Logger, basePath string, interval, sizeMb int64, format string) (*Logrotate, error) {

	// minimal path checks
	if fi, err := os.Stat(filepath.Dir(basePath)); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("invalid base path")
	}

	if fi, err := os.Stat(basePath); err == nil && fi.IsDir() {
		return nil, fmt.Errorf("log base path is a directory")
	}

	l := &Logrotate{
		log:           log,
		basePath:      basePath,
		interval:      interval,
		sizeMb:        sizeMb,
		format:        format,
		ticker:        time.NewTicker(time.Duration(interval) * time.Second),
		appDoneCh:     make(chan int),
		forceRotateCh: make(chan time.Time),
	}

	// check if the format is the increment (not a time's Layout)
	if intFmt, err := strconv.ParseInt(format, 10, 32); err == nil && intFmt == parsedIncrementFormat {
		l.useIncrement = true
	}

	os.Remove(l.basePath)

	go l.rotate()
	l.Start()

	return l, nil
}

// return suffix for the log file. depending on the
func (l *Logrotate) getSuffix(t time.Time) (result string) {
	if t == time.UnixMicro(0) {
		t = time.Now()
	}
	if l.useIncrement {
		result = fmt.Sprintf("%d", l.incrementedSuffix)
		l.incrementedSuffix++
	} else {
		result = t.Format(l.format)
	}
	return
}

func (l *Logrotate) Start() (err error) {
	if l.f, err = os.OpenFile(l.basePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		l.log.SetOutput(l.f)

		// we immediately hardlink 'base.log' to 'base.log.<suffix>'; rotate() does the same
		pathWithSuffix := l.basePath + "." + l.getSuffix(time.Now())
		if err = os.Link(l.basePath, pathWithSuffix); err != nil {
			return fmt.Errorf("error linking log file: %s", err)
		}

		// l.ticker = time.NewTicker(time.Duration(l.interval) * time.Second)
		l.ticker.Reset(time.Duration(l.interval) * time.Second)
		l.active = true
	}
	return
}

// pause the rotation
func (l *Logrotate) Stop() {
	// l.ticker.Stop()
	l.active = false
}

func (l *Logrotate) SetVerbose(v bool) {
	l.verbose = v
}

func (l *Logrotate) rotateLog(t time.Time) error {
	if !l.active {
		return nil
	}

	pathWithSuffix := l.basePath + "." + l.getSuffix(t)

	newF, err := os.OpenFile(pathWithSuffix, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("log rotation error, cannot create new file: %s", err)
	}

	log.SetOutput(newF)
	l.f.Close()
	l.f = newF
	os.Remove(l.basePath)
	os.Link(pathWithSuffix, l.basePath)
	return nil
}

func (l *Logrotate) rotate() {
	l.ticker = time.NewTicker(time.Duration(l.interval) * time.Second)
	for {
		select {
		case <-l.appDoneCh:
			return
		case t := <-l.ticker.C:
			if l.verbose {
				l.log.Info("scheduled log rotation")
			}
			if err := l.rotateLog(t); err != nil {
				return
			}
		case t := <-l.forceRotateCh:
			if l.verbose {
				l.log.Info("forced log rotation")
			}
			if err := l.rotateLog(t); err != nil {
				return
			}
		}
	}
}

// force rotation
func (l *Logrotate) Rotate() (err error) {
	if l.verbose {
		l.log.Info("forced log rotation")
	}
	err = l.rotateLog(time.UnixMicro(0))
	return
}
