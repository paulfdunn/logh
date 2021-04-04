// Package logh is a GO package for leveled logging.
// Key features:
//   Levels are user definable.
//   Multiple logs are supported.
//   Supports logging to a file, or STDOUT.
//       When logging to a file, 2 log rotations are managed, to the file size specified by the caller.
//   Log output is only written if the called logger is at or higher than the specified logging level.
//   The logging level can be changed at runtime; Shutdown and start at a new logging level.
package logh

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type LoghLevel int

// Constants for use with DefaultLevels.
const (
	Debug LoghLevel = iota
	Info
	Warning
	Audit
	Error
)

type Logger struct {
	checkLogSize           int
	flags                  int
	level                  LoghLevel
	levels                 []string
	levelMaxWidth          int
	loggers                []*log.Logger
	file                   *os.File
	filePath               string
	maxLogSize             int64
	rotation               int
	writesSinceCheckRotate int
}

const (
	// DefaultFlags are the default/recommended flags.
	DefaultFlags = log.LUTC | log.Ltime | log.Lmicroseconds | log.Ldate | log.Lshortfile

	// This package has only been tested with 2 rotations, but more *should* work.
	maxRotations = 2
)

var (
	DefaultLevels = []string{"debug", "info", "warning", "audit", "error"}

	// Map holds key/value pairs of named Loggers, created with New.
	// This pattern has the advantage, compared to just returning the logger from New,
	// of allowing a main function to configure loggers, and libraries or other functions
	// can just try to logger to a specific named logger, without concern for log size or if
	// the named logger even exists.
	Map = map[string]*Logger{}

	defaultOutput = os.Stdout
)

// New adds a new logger. This logger supports rotation of 2 files; suffix
// .0 and suffix .1.
// 	 name - is the name of this logger, accessed as logh.Map[name]
// 	 filePath - fully qualified file path to which to log.
// 	 levels - log levels, priority order (low to high). The strings are used for log prefixes.
// 	 level - index into levels specifying the current log level.
// 	 checkLogSize, maxLogSize - Every checkLogSize number of calls, the log file size is
//     checked, and if it exceeds maxLogSize, the file is rotated.
//     High(er) values for checkLogSize will improve performance due to reduced calls to get the
//     file size, but will allow the actual file size to overshoot maxLogSize.
//     Low(er) values of checkLogSize will insure less overshoot on actual log size, but will
//     incur the penalty of checking file size more frequently.
func New(name string, filePath string, levels []string, level LoghLevel, flags int,
	checkLogSize int, maxLogSize int64) error {

	// Shutdown and delete any existing loggers at this name.
	if _, ok := Map[name]; ok {
		Map[name].Shutdown()
	}
	delete(Map, name)

	lg := Logger{
		checkLogSize: checkLogSize,
		flags:        flags,
		level:        level,
		levels:       levels,
		filePath:     filePath,
		maxLogSize:   maxLogSize,
	}
	logger := &lg

	if level < 0 || int(level) >= len(levels) {
		return fmt.Errorf("input level was outside range, level:%d, len(levels)-1:%d", level, len(levels)-1)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("creating log file directory, error:%v", err)
	}

	if err := logger.initializeRotation(); err != nil {
		return err
	}

	if err := logger.openFileAndInitialize(); err != nil {
		return err
	}

	// initialize levelMaxWidth, used to format output so the prefix is constant length
	// for the various Levels.
	for _, v := range logger.levels {
		if len(v) > logger.levelMaxWidth {
			logger.levelMaxWidth = len(v)
		}
	}

	Map[name] = logger
	return nil
}

// Printf wraps the log.Printf in order to rotate the file.
func (l *Logger) Printf(level LoghLevel, format string, v ...interface{}) {
	l.printCommon(level, format, v...)
}

// Println wraps the log.Println in order to rotate the file.
func (l *Logger) Println(level LoghLevel, v ...interface{}) {
	l.printCommon(level, "%s", v...)
}

// Shutdown shuts down loggers and closes the file.
func (l *Logger) Shutdown() error {
	for i := range l.loggers {
		l.loggers[i] = nil
	}
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("closing log file, error:%v", err)
		}
	}
	return nil
}

// ShutdownAll is a convenience function to shutdown all running loggers and clear Map.
func ShutdownAll() error {
	var errOut error
	for k := range Map {
		err := Map[k].Shutdown()
		if err != nil {
			errOut = fmt.Errorf("error: %v, prior errors: %v", err, errOut)
		}
	}
	Map = map[string]*Logger{}
	return errOut
}

func (l *Logger) checkSizeAndRotate() error {
	if l.filePath == "" {
		return nil
	}

	l.writesSinceCheckRotate = 0
	var err error
	var fi os.FileInfo
	if fi, err = os.Stat(l.filePath + "." + strconv.Itoa(l.rotation)); err != nil {
		return err
	}

	if fi.Size() > l.maxLogSize {
		l.rotation++
		if l.rotation >= maxRotations {
			l.rotation = 0
		}
		if err := os.Remove(l.filePath + "." + strconv.Itoa(l.rotation)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := l.openFileAndInitialize(); err != nil {
			return err
		}
	}

	return nil
}

func (l *Logger) initializeLoggers() {
	l.loggers = make([]*log.Logger, len(l.levels))
	for i, v := range l.levels {
		l.loggers[i] = log.New(l.file, v+": ", l.flags)
	}
}

// initializeRotation will find the first available rotation that is less than maxLogSize.
func (l *Logger) initializeRotation() error {
	for i := 0; i < maxRotations; i++ {
		fp := l.filePath + "." + strconv.Itoa(i)
		fi, err := os.Stat(fp)
		if err != nil {
			// File does not exist; should be os.IsNotExist(err)
			l.rotation = i
			return nil
		}
		if fi.Size() < l.maxLogSize {
			// Add to existing file.
			l.rotation = i
			return nil
		}
	}

	// All files are >= maxLogSize, clear and use rotation 0
	l.rotation = 0
	return os.Remove(l.filePath + ".0")
}

// openFileAndInitialize opens the file and assigns loggers. On error, which can happen
// at startup or during file rotations, errors will result in the defaultOutput being
// used for logging.
func (l *Logger) openFileAndInitialize() error {
	var err, errors error
	l.writesSinceCheckRotate = 0
	if l.filePath == "" {
		l.file = defaultOutput
	} else {
		if l.file != nil {
			// When calling due to rotation, Shutdown running logger.
			if err := l.Shutdown(); err != nil {
				errors = fmt.Errorf("closing log file, error:%v", err)
			}
		}
		fp := l.filePath + "." + strconv.Itoa(l.rotation)
		l.file, err = os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			l.file = defaultOutput
			errors = fmt.Errorf("%v, opening log file, error:%v", errors, err)
		}
	}

	l.initializeLoggers()

	return errors
}

// printCommon is a separate function so the call stack is the same from Printf
// and Println. (This could have been in Printf, and Println call Printf. But then
// the call stack is different, and the argument to Output would need to change
// depending on the caller.)
func (l *Logger) printCommon(level LoghLevel, format string, v ...interface{}) {
	if l == nil {
		return
	}

	if int(level) >= len(l.levels) {
		fmt.Printf("input level was outside range, level:%d, len(levels)-1:%d", level, len(l.levels)-1)
		return
	}

	if level >= l.level {
		l.loggers[level].Output(3, fmt.Sprintf(format, v...))
	}

	if l.filePath == "" {
		return
	}
	l.writesSinceCheckRotate++
	if l.writesSinceCheckRotate >= l.checkLogSize {
		l.checkSizeAndRotate()
	}
}
