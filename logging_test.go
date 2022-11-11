package logging

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

type logTest struct {
	filePath, env string
	l             *Log
}

var (
	l           *logTest
	testLog     string = "test write for get log one\nwith a few\nnewlines"
	testLogTwo  string = "test write for get log two\nplus this line"
	testLogPath string
)

func TestMain(m *testing.M) {
	initialiseTest()
	code := m.Run()
	tearDownTest()
	os.Exit(code)
}

func TestErrorLog(t *testing.T) {
	l.l.ErrLog(errors.New("test error"), false)
	checkError(t)
}

func TestLog(t *testing.T) {
	l.l.Write("test write", INFO)
	checkWrite(t, INFO, "test write")
}

func TestLevels(t *testing.T) {
	l.l.Success("success")
	checkWrite(t, SUCCESS, "success")
	l.l.Debug("debug")
	checkWrite(t, DEBUG, "debug")
	l.l.Error("error")
	checkWrite(t, ERROR, "error")
	l.l.Warning("warning")
	checkWrite(t, WARNING, "warning")
	l.l.Info("info")
	checkWrite(t, INFO, "info")
}

func TestLevelsf(t *testing.T) {
	args := []interface{}{"testing", 12, 3.44534, false}
	msg := fmt.Sprintf("%s, %d, %2.f, %t", args...)
	l.l.Successf("success %s, %d, %2.f, %t", "testing", 12, 3.44534, false)
	checkWrite(t, SUCCESS, fmt.Sprintf("success %s", msg))
	l.l.Debugf("debug %s, %d, %2.f, %t", "testing", 12, 3.44534, false)
	checkWrite(t, DEBUG, fmt.Sprintf("debug %s", msg))
	l.l.Errorf("error %s, %d, %2.f, %t", "testing", 12, 3.44534, false)
	checkWrite(t, ERROR, fmt.Sprintf("error %s", msg))
	l.l.Warningf("warning %s, %d, %2.f, %t", "testing", 12, 3.44534, false)
	checkWrite(t, WARNING, fmt.Sprintf("warning %s", msg))
	l.l.Infof("info %s, %d, %2.f, %t", "testing", 12, 3.44534, false)
	checkWrite(t, INFO, fmt.Sprintf("info %s", msg))
}

func TestGetLog(t *testing.T) {
	err := l.l.Write(testLog, "INFO")
	if err != nil {
		t.Fatal(err)
	}
	err = l.l.Write(testLogTwo, "INFO")
	if err != nil {
		t.Fatal(err)
	}
	getLogOne(t)
	getLogTwo(t)
}

func getLogOne(t *testing.T) {
	result, err := l.l.GetLog(2)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 2 {
		t.Errorf("expected get log to contain two results, got %d", len(result))
		return
	}
	if !strings.Contains(result[0], testLogTwo) {
		t.Errorf("expected log result to contain '%s', got %s", testLogTwo, result[0])
	}
	if !strings.Contains(result[1], "test write for get log one") {
		t.Errorf("expected log result to contain '%s', got %s", testLog, result[1])
	}
}

func getLogTwo(t *testing.T) {
	result, err := l.l.GetLog(200) // should get the whole log
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) < 2 {
		t.Errorf("expected get log to contain at least two results, got %d", len(result))
		return
	}
	hasinit := false
	for _, v := range result {
		if strings.Contains(v, "initialising log") {
			hasinit = true
			break
		}
	}
	if !hasinit {
		t.Errorf("expected get log to return all lines including the initialising line, got %s", strings.Join(result, "\n"))
	}
}

func TestLogPath(t *testing.T) {
	if l.l.Path() != testLogPath {
		t.Errorf("expected test log path to be %s, got %s", testLogPath, l.l.Path())
	}
}

func TestReporting(t *testing.T) {
	err := spinTestLog(LEVEL_ERROR, REPORT_LEVEL_DEBUG)
	if err != nil {
		t.Fatal(err)
	}
	msg := fmt.Sprintf("%s: %d", "expects a report here but not a log", time.Now().Unix())
	l.l.Debug(msg)
	content, err := getFileContent()
	if err != nil {
		t.Fatal(err)
	}
	contentSplit := strings.Split(strings.Trim(string(content), "\n"), "\n")
	lastLog := contentSplit[len(contentSplit)-1]
	if strings.Contains(lastLog, msg) {
		t.Errorf("expected message '%s' to be omitted from the log", msg)
	}
}

func TestCustomLevel(t *testing.T) {
	err := spinTestLog(LEVEL_ERROR, REPORT_LEVEL_NONE) // Lowest possible sensitivity
	if err != nil {
		t.Fatal(err)
	}
	msg := fmt.Sprintf("%s: %d", "expects a report here and a log because we always log custom levels", time.Now().Unix())
	l.l.Write(msg, "CUSTOMLEVEL")
	checkWrite(t, "CUSTOMLEVEL", msg)
}

func initialiseTest() {
	err := spinTestLog(LEVEL_INFO, REPORT_LEVEL_INFO)
	if err != nil {
		os.Remove(l.filePath)
		panic(err)
	}
}

func tearDownTest() {
	defer l.l.file.Close()
	err := os.Remove(l.l.path)
	if err != nil {
		log.Println(err)
	}
}

func spinTestLog(logLevel, reportLevel int) error {
	testLogPath = fmt.Sprintf("%d__tmp_test_log.log", time.Now().Unix())
	l = &logTest{
		filePath: testLogPath,
		env:      "TEST",
	}
	var err error
	l.l, err = NewLog(l.filePath, l.env, logLevel, reportLevel)
	return err
}

func checkError(t *testing.T) {
	content, err := getFileContent()
	if err != nil {
		t.Fatal(err)
	}
	contentSplit := strings.Split(strings.Trim(string(content), "\n"), "\n")
	if len(contentSplit) < 2 {
		t.Fatalf("expected at least two logs to have been written, got %d", len(contentSplit))
	}
	lastLog := contentSplit[len(contentSplit)-1]
	if !strings.Contains(lastLog, "[TEST.ERROR] test error") {
		t.Errorf("expected last log to contain '%s', got '%s'", "test error", lastLog)
	}
}

func checkWrite(t *testing.T, level, message string) {
	content, err := getFileContent()
	if err != nil {
		t.Fatal(err)
	}
	contentSplit := strings.Split(strings.Trim(string(content), "\n"), "\n")
	if len(contentSplit) < 2 {
		t.Fatalf("expected at least two logs to have been written, got %d", len(contentSplit))
	}
	lastLog := contentSplit[len(contentSplit)-1]
	if !strings.Contains(lastLog, fmt.Sprintf("[TEST.%s] %s", level, message)) {
		t.Errorf("expected last log to contain '%s', got '%s'", fmt.Sprintf("[TEST.%s] %s", level, message), lastLog)
	}
}

func getFileContent() ([]byte, error) {
	file, err := os.OpenFile(l.filePath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	b := make([]byte, stat.Size())
	_, err = file.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
