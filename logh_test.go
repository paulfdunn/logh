package logh

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type testPrint struct {
	level  LoghLevel
	format string
	msg    string
}

var (
	loggerName = "testlogger"
	testLog    string
	testPrints = []testPrint{
		{Debug, "", "this is a debug print"},
		{Info, "", "this is a info print"},
		{Warning, "", "this is a warning print"},
		{Audit, "", "this is a audit print"},
		{Error, "", "this is a error print"}}
)

func init() {
	t := testing.T{}
	testLog = filepath.Join(t.TempDir(), "log.txt")
}

func (tp *testPrint) Println(t *testing.T) {
	Map[loggerName].Println(tp.level, tp.msg)
}

// TestDefaultOutput tests that non-file logging does go to the defaultLogger
func TestDefaultOutput(t *testing.T) {
	rf, wf, err := os.Pipe()
	if err != nil {
		t.Errorf("Cannot make pipe, error: %v", err)
	}

	defaultOutput = wf
	testSetup(t)
	err = New(loggerName, "", DefaultLevels, Debug, 0, 10, 1000)
	if err != nil {
		t.Errorf("error with New, error: %v", err)
	}

	Map[loggerName].Println(0, "Sending data to defaultOutput")
	buf := make([]byte, 1000)
	n, err := rf.Read(buf)
	if err != nil {
		t.Errorf("Cannot read pipe, error: %v", err)
	}
	out := string(buf[0:n])
	fmt.Printf("%s", out)
	if out != "debug: Sending data to defaultOutput\n" {
		t.Errorf("Incorrect output for defaultOutput, received: %s", out)
	}

	Map[loggerName].Shutdown()
}

// TestLineNumbers is used to verify the Output calldepth parameter is the correct
// value. That does mean that changes to this file required updating the test
// for new line numbers.
func TestLineNumbers(t *testing.T) {
	testSetup(t)
	err := New(loggerName, testLog, DefaultLevels, Debug, log.Lshortfile, 10, 1000)
	if err != nil {
		t.Errorf("error with New, error: %v", err)
	}

	Map[loggerName].Printf(0, "this is the Printf call")
	Map[loggerName].Println(0, "this is the Println call")
	// Shutdown to flush output.
	Map[loggerName].Shutdown()
	logString, _ := readTestLog(testLog, 0)
	fmt.Println(logString)
	if !strings.Contains(logString, "logh_test.go:79: this is the Printf call") ||
		!strings.Contains(logString, "logh_test.go:80: this is the Println call") {
		t.Errorf("Output calldepth problem")
	}
}
func TestRotate(t *testing.T) {
	testSetup(t)
	err := New(loggerName, testLog, DefaultLevels, Debug, 0, 1, 70)
	if err != nil {
		t.Errorf("error with New, error: %v", err)
	}

	var out string
	var log0ShouldContain, log1ShouldContain []int
	subTest := 0
	fmt.Printf("\n\nsubtest: %d, Partially write log .0\n", subTest)
	out = fmt.Sprintf("%d-12345678901234567890123456789012345678901234567890", subTest)
	Map[loggerName].Println(0, out)
	log0String, _ := readTestLog(testLog, 0)
	fmt.Printf("log0\n%s\n", log0String)
	log1String, _ := readTestLog(testLog, 1)
	fmt.Printf("log1\n%s\n", log1String)
	log0ShouldContain = []int{subTest}
	log1ShouldContain = []int{}
	if len(log0String) == 0 || len(log1String) > 0 {
		t.Errorf("rotate subtest %d failed", subTest)
	}
	shouldContainCheck(t, log0String, log1String, log0ShouldContain, log1ShouldContain)

	subTest++
	fmt.Printf("\n\nsubtest: %d, Fill log .0\n", subTest)
	out = fmt.Sprintf("%d-12345678901234567890123456789012345678901234567890", subTest)
	Map[loggerName].Println(0, out)
	log0String, _ = readTestLog(testLog, 0)
	fmt.Printf("log0\n%s\n", log0String)
	log1String, _ = readTestLog(testLog, 1)
	fmt.Printf("log1\n%s\n", log1String)
	log0ShouldContain = append(log0ShouldContain, subTest)
	if len(log0String) == 0 || len(log1String) > 0 {
		t.Errorf("rotate test %d failed", subTest)
	}
	shouldContainCheck(t, log0String, log1String, log0ShouldContain, log1ShouldContain)

	subTest++
	fmt.Printf("\n\nsubtest: %d, Partially write log .1\n", subTest)
	out = fmt.Sprintf("%d-12345678901234567890123456789012345678901234567890", subTest)
	Map[loggerName].Println(0, out)
	log0String, _ = readTestLog(testLog, 0)
	fmt.Printf("log0\n%s\n", log0String)
	log1String, _ = readTestLog(testLog, 1)
	fmt.Printf("log1\n%s\n", log1String)
	log1ShouldContain = append(log1ShouldContain, subTest)
	if len(log0String) == 0 || len(log1String) == 0 {
		t.Errorf("rotate test %d failed", subTest)
	}
	shouldContainCheck(t, log0String, log1String, log0ShouldContain, log1ShouldContain)

	subTest++
	fmt.Printf("\n\nsubtest: %d, Fill log .1; log .0 will now be opened/cleared as it was rotated in.\n", subTest)
	out = fmt.Sprintf("%d-12345678901234567890123456789012345678901234567890", subTest)
	Map[loggerName].Println(0, out)
	log0String, _ = readTestLog(testLog, 0)
	fmt.Printf("log0\n%s\n", log0String)
	log1String, _ = readTestLog(testLog, 1)
	fmt.Printf("log1\n%s\n", log1String)
	log0ShouldContain = []int{}
	log1ShouldContain = append(log1ShouldContain, subTest)
	if len(log0String) > 0 || len(log1String) == 0 {
		t.Errorf("rotate test %d failed", subTest)
	}
	shouldContainCheck(t, log0String, log1String, log0ShouldContain, log1ShouldContain)

	subTest++
	fmt.Printf("\n\nsubtest: %d, Partially write log .0.\n", subTest)
	out = fmt.Sprintf("%d-12345678901234567890123456789012345678901234567890", subTest)
	Map[loggerName].Println(0, out)
	log0String, _ = readTestLog(testLog, 0)
	fmt.Printf("log0\n%s\n", log0String)
	log1String, _ = readTestLog(testLog, 1)
	fmt.Printf("log1\n%s\n", log1String)
	log0ShouldContain = append(log0ShouldContain, subTest)
	if len(log0String) == 0 || len(log1String) == 0 {
		t.Errorf("rotate test %d failed", subTest)
	}
	shouldContainCheck(t, log0String, log1String, log0ShouldContain, log1ShouldContain)

	Map[loggerName].Shutdown()
}

// TestShowOutput can be used with the -v parameter to just demo the output. This is not an
// Example because the logger output is not seen by the tests on STDOUT.
func TestShowOutput(t *testing.T) {
	rf, wf, err := os.Pipe()
	if err != nil {
		t.Errorf("Cannot make pipe, error: %v", err)
	}

	defaultOutput = wf
	testSetup(t)

	aLog := "app"
	checkLogSize := 10 // every 10 entries, check log size and rotate if size exceeds maxLogSize.
	maxLogSize := int64(10000)
	err = New(aLog, "", DefaultLevels, Debug, DefaultFlags, checkLogSize, maxLogSize)
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
	defer ShutdownAll()
	// Re-define alias with New log.
	lp = Map[aLog].Println
	if err != nil {
		t.Errorf("error with New, error: %v", err)
	}
	lp(Debug, "This is a debug level print, but will not output with warning level logging.")
	lp(Warning, "Warning and higher do print")

	// Change back to debug level  logging
	err = New(aLog, "", DefaultLevels, Debug, DefaultFlags, checkLogSize, maxLogSize)
	if err != nil {
		t.Errorf("error with New, error: %v", err)
	}
	lp(Debug, "And this debug print is now output.")

	buf := make([]byte, 1000)
	n, err := rf.Read(buf)
	if err != nil {
		t.Errorf("Cannot read pipe, error: %v", err)
	}
	out := string(buf[0:n])
	fmt.Printf("%s", out)
}

// TestLevels tests that the proper number of lines are included in output for the specified
// Level.
func TestLevels(t *testing.T) {
	testSetup(t)
	for lvl := 0; lvl <= 3; lvl++ {
		err := New(loggerName, testLog, DefaultLevels, LoghLevel(lvl), DefaultFlags, 10, 10000)
		if err != nil {
			t.Errorf("error with New, error: %v", err)
		}

		for _, v := range testPrints {
			v.Println(t)
		}

		logString, err := readTestLog(testLog, 0)
		fmt.Printf("\nTestLevels lvl:%d\n%s", lvl, logString)
		if err != nil {
			t.Errorf("Error reading log file, error: %+v", err)
		}
		lines := strings.Split(string(logString), "\n")
		switch lvl {
		case 0:
			// len(DefaultLevels) +1; final line ends with \n
			if len(lines) != len(DefaultLevels)+1 {
				t.Errorf("Wrong number of lines with level %d, lines: %d", lvl, len(lines))
			}

			if !strings.Contains(logString, DefaultLevels[0]) ||
				!strings.Contains(logString, DefaultLevels[1]) ||
				!strings.Contains(logString, DefaultLevels[2]) ||
				!strings.Contains(logString, DefaultLevels[3]) ||
				!strings.Contains(logString, DefaultLevels[4]) {
				t.Errorf("Log missing level: %d", lvl)
			}
		case 1:
			if len(lines) != len(DefaultLevels) {
				t.Errorf("Wrong number of lines with level %d, lines: %d", lvl, len(lines))
			}

			if strings.Contains(logString, DefaultLevels[0]) {
				t.Errorf("Log contains level that should have been filtered, level: %d", lvl)
			}

			if !strings.Contains(logString, DefaultLevels[1]) ||
				!strings.Contains(logString, DefaultLevels[2]) ||
				!strings.Contains(logString, DefaultLevels[3]) ||
				!strings.Contains(logString, DefaultLevels[4]) {
				t.Errorf("Log missing level: %d", lvl)
			}
		case 2:
			if len(lines) != len(DefaultLevels)-1 {
				t.Errorf("Wrong number of lines with level %d, lines: %d", lvl, len(lines))
			}

			if strings.Contains(logString, DefaultLevels[0]) ||
				strings.Contains(logString, DefaultLevels[1]) {
				t.Errorf("Log contains level that should have been filtered, level: %d", lvl)
			}

			if !strings.Contains(logString, DefaultLevels[2]) ||
				!strings.Contains(logString, DefaultLevels[3]) ||
				!strings.Contains(logString, DefaultLevels[4]) {
				t.Errorf("Log missing level: %d", lvl)
			}
		case 3:
			if len(lines) != len(DefaultLevels)-2 {
				t.Errorf("Wrong number of lines with level %d, lines: %d", lvl, len(lines))
			}

			if strings.Contains(logString, DefaultLevels[0]) ||
				strings.Contains(logString, DefaultLevels[1]) ||
				strings.Contains(logString, DefaultLevels[2]) {
				t.Errorf("Log contains level that should have been filtered, level: %d", lvl)
			}

			if !strings.Contains(logString, DefaultLevels[3]) ||
				!strings.Contains(logString, DefaultLevels[4]) {
				t.Errorf("Log missing level: %d", lvl)
			}
		case 4:
			if len(lines) != len(DefaultLevels)-3 {
				t.Errorf("Wrong number of lines with level %d, lines: %d", lvl, len(lines))
			}

			if strings.Contains(logString, DefaultLevels[0]) ||
				strings.Contains(logString, DefaultLevels[1]) ||
				strings.Contains(logString, DefaultLevels[2]) ||
				strings.Contains(logString, DefaultLevels[3]) {
				t.Errorf("Log contains level that should have been filtered, level: %d", lvl)
			}

			if !strings.Contains(logString, DefaultLevels[4]) {
				t.Errorf("Log missing level: %d", lvl)
			}
		}

		testSetup(t)
	}

	Map[loggerName].Shutdown()
}

// Test2Logs tests writing to 2 independent logs
func Test2Logs(t *testing.T) {
	testLog1 := filepath.Join(t.TempDir(), "log1.txt")
	testLog2 := filepath.Join(t.TempDir(), "log2.txt")

	err := New("testLog1", testLog1, DefaultLevels, Debug, DefaultFlags, 10, 10000)
	if err != nil {
		removeLogs(testLog1, t)
		t.Errorf("error with New, error: %v", err)
	}
	err = New("testLog2", testLog2, DefaultLevels, Debug, DefaultFlags, 10, 10000)
	if err != nil {
		removeLogs(testLog1, t)
		removeLogs(testLog2, t)
		t.Errorf("error with New, error: %v", err)
	}

	Map["testLog1"].Println(Debug, "log1")
	Map["testLog2"].Println(Debug, "log2")

	l1, err := readTestLog(testLog1, 0)
	fmt.Printf("log1:%s", l1)
	if err != nil {
		removeLogs(testLog1, t)
		removeLogs(testLog2, t)
		t.Errorf("error reading, error: %v", err)
	}
	l2, err := readTestLog(testLog2, 0)
	fmt.Printf("log2:%s", l2)
	if err != nil {
		removeLogs(testLog1, t)
		removeLogs(testLog2, t)
		t.Errorf("error reading, error: %v", err)
	}

	if !(strings.Contains(l1, "log1") && strings.Contains(l2, "log2")) {
		t.Errorf("independent logs not working, l1:%s, l2:%s", l1, l2)
		return
	}

	removeLogs(testLog1, t)
	removeLogs(testLog2, t)
}

func readTestLog(filepath string, rotation int) (string, error) {
	b, err := ioutil.ReadFile(filepath + "." + strconv.Itoa(rotation))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func removeLogs(filepath string, t *testing.T) {
	for i := 0; i < maxRotations; i++ {
		err := os.Remove(filepath + "." + strconv.Itoa(i))
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("error removing log file, error: %+v", err)
		}
	}
}

func shouldContainCheck(t *testing.T, log0String string, log1String string,
	log0ShouldContain []int, log1ShouldContain []int) {

	for _, v := range log0ShouldContain {
		expected := fmt.Sprintf(" %d-", v)
		if !strings.Contains(log0String, expected) {
			t.Errorf("rotate subtest failed, missing contents: %s", expected)
		}
	}

	for _, v := range log1ShouldContain {
		expected := fmt.Sprintf(" %d-", v)
		if !strings.Contains(log1String, expected) {
			t.Errorf("rotate subtest failed, missing contents: %s", expected)
		}
	}
}

func testSetup(t *testing.T) {
	removeLogs(testLog, t)
}
