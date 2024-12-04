package main

import (
	"context"
	"fmt"
	// "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

func main() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		// handle error!
	}
	resp, err := cli.Get(cli.Ctx(), "mykey2")
	if err != nil {
		switch err {
		case context.Canceled:
			log.Fatalf("ctx is canceled by another routine: %v", err)
		case context.DeadlineExceeded:
			log.Fatalf("ctx is attached with a deadline is exceeded: %v", err)
		default:
			log.Fatalf("bad cluster endpoints, wich are not etcd servers: %v", err)
		}
	} else {
		// get the value as string
		// check length to avoid panic
		if len(resp.Kvs) == 0 {
			fmt.Printf("key not found\n")
		} else {
			s := string(resp.Kvs[0].Value)
			fmt.Printf("get value: %s\n", s)
		}
	}
	defer cli.Close()
}
