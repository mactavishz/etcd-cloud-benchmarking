package runner

import (
	"context"
	"time"

	status "google.golang.org/grpc/status"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

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
