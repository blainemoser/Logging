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

var l *logTest

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
	l.l.Write("test write", "INFO")
	checkWrite(t)
}

func TestGetLog(t *testing.T) {
	err := l.l.Write("test write for get log one", "INFO")
	if err != nil {
		t.Fatal(err)
	}
	err = l.l.Write("test write for get log two", "INFO")
	if err != nil {
		t.Fatal(err)
	}
	result, err := l.l.GetLog(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected get log to contain two results, got %d", len(result))
	}
	if !strings.Contains(result[0], "test write for get log two") {
		t.Errorf("expected log result to contain '%s', got %s", "test write for get log two", result[0])
	}
	if !strings.Contains(result[1], "test write for get log one") {
		t.Errorf("expected log result to contain '%s', got %s", "test write for get log one", result[1])
	}
}

func initialiseTest() {
	err := spinTestLog()
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

func spinTestLog() error {
	l = &logTest{
		filePath: fmt.Sprintf("%d__tmp_test_log.log", time.Now().Unix()),
		env:      "TEST",
	}
	var err error
	l.l, err = NewLog(l.filePath, l.env)
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

func checkWrite(t *testing.T) {
	content, err := getFileContent()
	if err != nil {
		t.Fatal(err)
	}
	contentSplit := strings.Split(strings.Trim(string(content), "\n"), "\n")
	if len(contentSplit) < 2 {
		t.Fatalf("expected at least two logs to have been written, got %d", len(contentSplit))
	}
	lastLog := contentSplit[len(contentSplit)-1]
	if !strings.Contains(lastLog, "[TEST.INFO] test write") {
		t.Errorf("expected last log to contain '%s', got '%s'", "test write", lastLog)
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
