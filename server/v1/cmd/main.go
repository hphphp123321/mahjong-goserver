package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"mahjong-goserver/server/v1"
	pb "mahjong-goserver/services/mahjong/v1"
	"net"
)

var (
	maxClients int
	address    string
	port       int
)

func parseFlags() {
	flag.IntVar(&maxClients, "maxClients", 10, "max clients")
	flag.StringVar(&address, "address", "127.0.0.1", "server address")
	flag.IntVar(&port, "port", 7777, "port")
	flag.Parse()
}

func main() {
	parseFlags()
	fmt.Println("Hello World!")
	tcpAddr := fmt.Sprintf("%s:%d", address, port)
	lis, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	server := v1.NewMahjongServer(10)
	pb.RegisterMahjongServer(s, server)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
