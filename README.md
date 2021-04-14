# logh
logh is a GO (GOLANG) package for leveled logging, for multiple simultaneous logs, with log rotation.
Key features:
* Multiple simultaneous log files, each with their own log level, are supported.
* Log rotation is supported.
* Default levels are provided, but the user can provide user defined levels on a per log basis.
* Supports logging to a file, or STDOUT.
    * When logging to a file, 2 log rotations are managed, to the file size specified by the caller.
* Log output is only written if the called logger is at or higher than the specified logging level.
* The logging level can be changed at runtime; Shutdown and start at a new logging level.

Example setup and use:
```
aLog := "app"
checkLogSize := 10 // every 10 entries, check log size and rotate if size exceeds maxLogSize.
maxLogSize := int64(10000)
err = New(aLog, "", DefaultLevels, Debug, DefaultFlags, checkLogSize, maxLogSize)
defer ShutdownAll()
if err != nil {
    t.Errorf("error with New, error: %v", err)
}

// Define an alias to use to keep print statements short.
lp := Map[aLog].Println
lp(Debug, "This is a debug level print; debug level logging.")
lp(Info, "This is a info level print; debug level logging.")
lp(Warning, "This is a warning level print; debug level logging.")
lp(Audit, "This is a audit level print; debug level logging.")
lp(Error, "This is a error level print; debug level logging.")

// Change to warning level logging
err = New(aLog, "", DefaultLevels, Warning, DefaultFlags, checkLogSize, maxLogSize)
// Re-define alias with New log.
lp = Map[aLog].Println
if err != nil {
    t.Errorf("error with New, error: %v", err)
}
lp(Debug, "This is a debug level print, but will not output with warning level logging.")
lp(Warning, "Warning and higher do print")
```

Example output:
```
debug: 2021/04/01 15:43:24.617769 logh_test.go:194: This is a debug level print; debug level logging.
info: 2021/04/01 15:43:24.617778 logh_test.go:195: This is a info level print; debug level logging.
warning: 2021/04/01 15:43:24.617783 logh_test.go:196: This is a warning level print; debug level logging.
audit: 2021/04/01 15:43:24.617787 logh_test.go:197: This is a audit level print; debug level logging.
error: 2021/04/01 15:43:24.617792 logh_test.go:198: This is a error level print; debug level logging.
# Note - debug level did not print with warning level logging.
warning: 2021/04/01 15:43:24.617803 logh_test.go:206: Warning and higher do print
```