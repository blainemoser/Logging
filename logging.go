package logging

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	ERROR = "ERROR"

	WARNING = "WARNING"

	INFO = "INFO"

	SUCCESS = "SUCCESS"

	DEBUG = "DEBUG"
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

func (l *Log) Error(message string) (err error) {
	return l.Write(message, ERROR)
}

func (l *Log) Success(message string) (err error) {
	return l.Write(message, SUCCESS)
}

func (l *Log) Warning(message string) (err error) {
	return l.Write(message, WARNING)
}

func (l *Log) Debug(message string) (err error) {
	return l.Write(message, DEBUG)
}

func (l *Log) Info(message string) (err error) {
	return l.Write(message, INFO)
}

func (l *Log) Errorf(message string, vars ...interface{}) (err error) {
	return l.Error(fmt.Sprintf(message, vars...))
}

func (l *Log) Successf(message string, vars ...interface{}) (err error) {
	return l.Success(fmt.Sprintf(message, vars...))
}

func (l *Log) Warningf(message string, vars ...interface{}) (err error) {
	return l.Warning(fmt.Sprintf(message, vars...))
}

func (l *Log) Debugf(message string, vars ...interface{}) (err error) {
	return l.Debug(fmt.Sprintf(message, vars...))
}

func (l *Log) Infof(message string, vars ...interface{}) (err error) {
	return l.Info(fmt.Sprintf(message, vars...))
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

// Path returns the file path
func (l *Log) Path() string {
	return l.path
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
	var err error
	_, err = l.file.ReadAt(b, *chunk)
	if err != nil {
		if strings.Contains(err.Error(), "negative offset") {
			return l.wholeRead(fileSize)
		}
		return []string{}, err
	}
	split := strings.Split(string(b), "\n")
	l.iterateChunkSplit(split, &r)
	if len(r) >= lines {
		return r[0:lines], nil
	}
	*count++
	*size = chunkSize * int64(lines) * *count
	*chunk = fileSize - *size
	return nil, nil
}

func (l *Log) wholeRead(fileSize int64) ([]string, error) {
	b := make([]byte, fileSize)
	_, err := l.file.Read(b)
	if err != nil {
		return []string{}, err
	}
	splitLog := strings.Split(string(b), "\n")
	node := make([]string, 0)
	result := make([]string, 0)
	for i, v := range splitLog {
		if dateForm.MatchString(v) {
			if len(node) > 0 {
				result = append(result, strings.Trim(strings.Join(node, "\n"), " "))
			}
			node = []string{v}
			continue
		}
		node = append(node, strings.Trim(v, " "))
		if i == len(splitLog)-1 && len(node) > 0 {
			result = append(result, strings.Trim(strings.Join(node, "\n"), " "))
		}
	}
	return result, nil
}

func (l *Log) iterateChunkSplit(split []string, result *[]string) {
	node := make([]string, 0)
	for i := len(split) - 1; i > 0; i-- {
		if dateForm.MatchString(split[i]) {
			node = append(node, split[i])
			l.reverseNode(&node)
			*result = append(*result, strings.Trim(strings.Join(node, "\n"), " "))
			node = make([]string, 0)
			continue
		}
		node = append(node, strings.Trim(split[i], " "))
	}
}

func (l *Log) reverseNode(node *[]string) {
	if len(*node) < 2 {
		return
	}
	var place string
	for i := 0; i < len(*node)/2; i++ {
		place = (*node)[i]
		(*node)[i] = (*node)[len(*node)-1-i]
		(*node)[len(*node)-1-i] = place
	}
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
