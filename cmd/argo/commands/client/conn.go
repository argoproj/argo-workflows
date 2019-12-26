package client

import (
	"log"

	"google.golang.org/grpc"
)

func GetClientConn(server string) *grpc.ClientConn {
	conn, err := grpc.Dial(server, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	return conn
}
