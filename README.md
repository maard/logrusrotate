# logrusrotate

This package can be used with [logrus](github.com/sirupsen/logrus) to add effortless log rotation support to your long-running daemon.

# Why [this package was created]

Task requirements:
- _yourapp.log_ should always refer to the actual log (which you should be able to tail using `tail -f yourapp.log`)
- periodically (or after size threshold is reached) this log file needs to be archived (copied to a name with a suffix)
- all currently executing _log.XXX_ calls should successfully finish writing to the file being rotated

# How [this module takes a different approach to log file 'rotation']

When creating a log file, `New()` also creates another entry (_logname.<suffix>_) as a hardlink to the _logname_.
Each time the rotation happens, a new file with the suffix is first created, `log.SetOutput` is called, and _logname_ is re-linked to point to the same file.

# Suffix format

This can either be one of the [time](https://pkg.go.dev/time#pkg-constants) formats, or a string, which must parse to integer 8 (you can figure out, why).

# Example

```go
// main.go
import "logrusrotate"

const logBaseName = "/path/to/yourapp.log"

var log *logrus.Logger

func startLogrotate(log *ruslog.Logger) {
	rotateInterval := 1 * logrusrotate.Hour // .Second, .Minute, .Hour, .Day
	rotateMb := 100 // not currently used
	suffixFormat := "20060102_150405"

	logrusrotate.New(log, logBaseName, rotateInterval, rotateMb, suffixFormat)
}

func main() {
	log = logrus.New()
	startLogrotate(log)
}
```

# API

## logrusrotate.New()

Creates a rotator instance, which runs `Start()` immediately.

```go
	rotator := logrusrotate.New(...)
```

## rotator.Start()

Re-starts a previously stopped rotation.

## rotator.Stop()

Disables the rotation

## rotator.Rotate(suffixTime time.Time)

Forces the immediate log rotation. The rotation timer is restarted. If the argument is 0 (e.g. `time.UnixMicro(0)`), `time.Now()` is used instead.

## rotator.SetVerbose(bool)

Verbosity adds a corresponding line to the log when the file is rotated periodically or forcefully.

# TODO

- rotation by size is not implemented
- sequential formats do not respect zero-prefixed suffixes (e.g. "0008" is treated the same as "8")
