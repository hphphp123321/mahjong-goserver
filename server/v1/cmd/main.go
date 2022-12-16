package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	v1 "mahjong-goserver/server/v1"
	pb "mahjong-goserver/services/mahjong/v1"
	"net"
	"time"
)

var (
	maxClients int
	address    string
	port       int

	minTime               int
	maxConnectionIdle     int
	maxConnectionAgeGrace int
	timeTick              int
	timeout               int
)

func parseFlags() {
	flag.IntVar(&maxClients, "maxClients", 10, "max clients")
	flag.StringVar(&address, "address", "127.0.0.1", "server address")
	flag.IntVar(&port, "port", 7777, "port")
	flag.IntVar(&minTime, "minTime", 1, "If a client pings more than once every MinTime seconds, terminate the connection")
	flag.IntVar(&maxConnectionIdle, "maxConnectionIdle", 15, "If a client is idle for Idle seconds, send a GOAWAY")
	flag.IntVar(&maxConnectionAgeGrace, "maxConnectionAgeGrace", 5, "Allow Grace seconds for pending RPCs to complete before forcibly closing connections")
	flag.IntVar(&timeTick, "timeTick", 10, "Ping the client if it is idle for timeTick seconds to ensure the connection is still active")
	flag.IntVar(&timeout, "timeout", 5, "Wait 1 second for the ping ack before assuming the connection is dead")
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

	var kasp = keepalive.ServerParameters{
		MaxConnectionIdle:     time.Duration(maxConnectionIdle) * time.Second,
		MaxConnectionAgeGrace: time.Duration(maxConnectionAgeGrace) * time.Second,
		Time:                  time.Duration(timeTick) * time.Second,
		Timeout:               time.Duration(timeout) * time.Second,
	}

	var kaep = keepalive.EnforcementPolicy{
		MinTime:             time.Duration(minTime) * time.Second,
		PermitWithoutStream: true,
	}
	s := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	server := v1.NewMahjongServer(10)
	pb.RegisterMahjongServer(s, server)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
