package logger

import (
	"bufio"
	"io"
	"log"
	"os"
)

// Logger wraps the standard logger and file handle
type Logger struct {
	*log.Logger
	file   *os.File
	writer *bufio.Writer
}

// Close properly flushes and closes the log file
func (l *Logger) Close() error {
	if err := l.Flush(); err != nil {
		return err
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Flush() error {
	if l.writer != nil {
		return l.writer.Flush()
	}
	return nil
}

func NewLogger(filename string) (*Logger, error) {
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}

	bufferedWriter := bufio.NewWriter(logFile)

	// Write log to both stdout and log file
	multiWriter := io.MultiWriter(os.Stdout, bufferedWriter)

	logger := log.New(multiWriter, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	return &Logger{Logger: logger, file: logFile, writer: bufferedWriter}, nil
}
