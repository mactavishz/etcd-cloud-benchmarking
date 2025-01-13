package runner

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"time"

	status "google.golang.org/grpc/status"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
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

func GetTimeoutCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.TODO(), timeout)
}

func GetErrInfo(err error) (int, string) {
	var statusCode int
	var statusText string
	if err == context.Canceled {
		// ctx is canceled by another routine
		statusCode = -1
		statusText = "Context canceled by another goroutine"
	} else if err == context.DeadlineExceeded {
		// ctx is attached with a deadline and it exceeded
		statusCode = -2
		statusText = "Request deadline exceeded"
	} else if statusErr, ok := err.(rpctypes.EtcdError); ok {
		// etcd client rpc error
		statusCode = int(statusErr.Code())
		statusText = statusErr.Error()
	} else if ev, ok := status.FromError(err); ok {
		// gRPC status error
		statusCode = int(ev.Code())
		statusText = ev.String()
	} else if clientv3.IsConnCanceled(err) {
		statusCode = -3
		statusText = "gRPC Client connection closed"
	} else {
		// bad cluster endpoints, which are not etcd servers
		statusCode = -4
		statusText = err.Error()
	}
	return statusCode, statusText
}
