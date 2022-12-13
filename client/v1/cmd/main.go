package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"mahjong-goserver/client/v1"
	pb "mahjong-goserver/services/mahjong/v1"
	"time"
)

var (
	playerName string
	address    string
	port       int
	timeout    int
)

func parseFlags() {
	flag.StringVar(&playerName, "playerName", "player", "player name")
	flag.StringVar(&address, "address", "127.0.0.1", "server address")
	flag.IntVar(&port, "port", 7777, "port")
	flag.IntVar(&timeout, "timeout", 5, "seconds for timeout")
	flag.Parse()
}

// unaryInterceptor 一个简单的 unary interceptor 示例。
func unaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// pre-processing
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...) // invoking RPC method
	// post-processing
	end := time.Now()
	usedTime := end.Sub(start)
	log.Printf("RPC: %s, used time: %s", method, usedTime)
	return err
}

func main() {
	parseFlags()
	fmt.Println("Hello World!")
	tcpAddr := fmt.Sprintf("%s:%d", address, port)

	conn, err := grpc.Dial(tcpAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(unaryInterceptor))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	//pb.NewMahjongClient(conn)

	MahjongClient := pb.NewMahjongClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	c := v1.NewMahjongClient(ctx, playerName, MahjongClient)

	// Login
	err = c.Login()
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	// ping
	err = c.Ping()
	if err != nil {
		log.Fatalf("could not ping: %v", err)
	}

	// RefreshRoom
	err = c.RefreshRoom("")
	if err != nil {
		log.Fatalf("could not refresh room: %v", err)
	}

	// CreateRoom
	err = c.CreateRoom("room1")
	if err != nil {
		log.Fatalf("could not CreateRoom: %v", err)
	}

	// RefreshRoom
	err = c.RefreshRoom("")
	if err != nil {
		log.Fatalf("could not refresh room: %v", err)
	}

	// Logout
	err = c.Logout()
	if err != nil {
		log.Fatalf("could not logout: %v", err)
	}
}
