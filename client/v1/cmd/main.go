package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"log"
	"mahjong-goserver/client/v1"
	"mahjong-goserver/osutils"
	pb "mahjong-goserver/services/mahjong/v1"
	"time"
)

var (
	playerName string
	address    string
	port       int
	timeout    int
	timeTicker int
)

func parseFlags() {
	flag.StringVar(&playerName, "playerName", "player2", "player name")
	flag.StringVar(&address, "address", "127.0.0.1", "server address")
	flag.IntVar(&port, "port", 7777, "port")
	flag.IntVar(&timeout, "timeout", 5, "seconds for timeout")
	flag.IntVar(&timeTicker, "timeTicker", 10, "seconds for time ticker")
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

	var kacp = keepalive.ClientParameters{
		Time:                time.Duration(timeTicker),
		Timeout:             time.Duration(timeout),
		PermitWithoutStream: true,
	}

	tcpAddr := fmt.Sprintf("%s:%d", address, port)
	conn, err := grpc.Dial(tcpAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(unaryInterceptor), grpc.WithKeepaliveParams(kacp))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	//pb.NewMahjongClient(conn)

	MahjongClient := pb.NewMahjongClient(conn)
	//ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	//defer cancel()
	ctx := context.Background()
	c := v1.NewMahjongClient(ctx, playerName, MahjongClient)

	// Login
	err = c.Login()
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	defer func(c *v1.MahjongClient) {
		err := c.Logout()
		if err != nil {
			log.Fatalf("Logout failed: %v", err)
		}
	}(c)

	// ping
	go func() {
		ticker := time.NewTicker(time.Duration(timeTicker) * time.Second)
		for {
			select {
			case <-ticker.C:
				err := c.Ping()
				if err != nil {
					log.Fatalf("Ping failed: %v", err)
				}
			}
		}
	}()

	// RefreshRoom
	err = c.RefreshRoom("")
	if err != nil {
		log.Fatalf("could not refresh room: %v", err)
	}

	// JoinRoom
	if len(c.RoomList) > 0 {
		err = c.JoinRoom(c.RoomList[0].RoomID.String())
		if err != nil {
			log.Printf("could not join room: %v", err)
		}
	}

	// CreateRoom
	if c.Room == nil {
		err = c.CreateRoom("room1")
		if err != nil {
			log.Fatalf("could not create room: %v", err)
		}
	}

	err = c.Ready()
	if err != nil {
		log.Fatalf("could not ready: %v", err)
	}

	exitChannel := osutils.NewShutdownSignal()
	for {
		osutils.WaitExit(exitChannel, func() {
			log.Println("Exit")
			if err = c.Logout(); err != nil {
				log.Fatalf("Logout failed: %v", err)
			}
		})
	}

}
