package logging

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

type Log struct {
	path, env string
	file      *os.File
}

const chunkSize = 50

var (
	dateForm = regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}(.*?)$`)
)

func NewLog(path, env string) (l *Log, err error) {
	l = &Log{
		path: path,
		env:  env,
	}
	err = l.Write("initialising log", "INFO")
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Log) Write(message, level string) (err error) {
	err = l.openLogForWrite()
	if err != nil {
		return err
	}
	defer l.file.Close()
	_, err = l.file.Write(l.logMessage(level, message))
	return err
}

func (l *Log) logMessage(level, message string) []byte {
	return []byte(
		fmt.Sprintf(
			"[%s] [%s.%s] %s\n",
			time.Now().UTC().Format(time.RFC3339),
			l.env,
			level,
			message,
		),
	)
}

// GetLog returns lines of the log
func (l *Log) GetLog(lines uint) (result []string, err error) {
	err = l.openLogForRead()
	if err != nil {
		return result, err
	}
	defer l.file.Close()
	stat, err := l.file.Stat()
	if err != nil {
		return result, err
	}
	l.readChunks(int64(lines), stat.Size(), &result)
	return result, err
}

func (l *Log) readChunks(lines, fileSize int64, result *[]string) error {
	size := chunkSize * lines
	chunk := fileSize - size
	count := int64(1)
	for {
		r, err := l.iterateReadChunks(fileSize, int(lines), &size, &chunk, &count)
		if err != nil {
			return err
		}
		if r != nil {
			*result = r
			break
		}
		if count > 2000 {
			return fmt.Errorf("timeout")
		}
	}
	return nil
}

func (l *Log) iterateReadChunks(fileSize int64, lines int, size, chunk, count *int64) ([]string, error) {
	r := make([]string, 0)
	b := make([]byte, *size)
	_, err := l.file.ReadAt(b, *chunk)
	if err != nil {
		return nil, err
	}
	split := strings.Split(string(b), "\n")
	for i := len(split) - 1; i > 0; i-- {
		if dateForm.Match([]byte(split[i])) {
			r = append(r, split[i])
		}
	}
	if len(r) >= lines {
		return r[0:lines], nil
	}
	*count++
	*size = chunkSize * int64(lines) * *count
	*chunk = fileSize - *size
	return nil, nil
}

func (l *Log) ErrLog(e error, fatal bool) {
	if fatal {
		l.Write(e.Error(), "FATAL")
		l.file.Close()
		log.Fatal(e)
		return
	}
	l.Write(e.Error(), "ERROR")
	log.Println(e)
}

func (l *Log) openLogForWrite() error {
	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil && os.IsNotExist(err) {
		_, err = os.Create(l.path)
		if err != nil {
			return err
		}
		return l.openLogForWrite()
	}
	l.file = file
	return err
}

func (l *Log) openLogForRead() error {
	file, err := os.OpenFile(l.path, os.O_RDONLY, os.ModeDevice)
	if err != nil && os.IsNotExist(err) {
		_, err = os.Create(l.path)
		if err != nil {
			return err
		}
		return l.openLogForRead()
	}
	l.file = file
	return err
}
