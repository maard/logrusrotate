# logrusrotate

An utility module for [logrus](github.com/sirupsen/logrus).
Allows a long-running process (daemon) to safelystart writing to the next log file,
while existing gosubs finish writing into the previous file.

# Example

```go

// main()
import "logrusrotate"

func main() {
	log := logrus.New()
	logBasePath := "/path/to/daemon.log"
	rotateInterval := 60 * logrusrotate.Second // .Minute, .Hour, .Day
	rotateMb := 100
	// in addition to `time` formats, '8' is replaced with an integer value, which starts with 0,
	// and is incremented on each rotate
	suffixFormat := time.FRC3339

	rotate := logrusrotate.New(log, logBasePath, rotateInterval, rotateMb, suffixFormat)

	// if needed:
	// rotate.Stop() // disables the rotation
	// rotate.Start() // [re]starts the rotation
}

```