package logging

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Log struct {
	path, env string
	file      *os.File
}

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
	err = l.getLog()
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
			"[%s] [%s.%s] %s\n\n",
			time.Now().UTC().Format(time.RFC3339),
			l.env,
			level,
			message,
		),
	)
}

func (l *Log) getLog() error {
	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil && os.IsNotExist(err) {
		_, err = os.Create(l.path)
		if err != nil {
			return err
		}
		return l.getLog()
	}
	l.file = file
	return err
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
