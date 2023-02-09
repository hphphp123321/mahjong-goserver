package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hphphp123321/mahjong-goserver/client/v1"
	"github.com/hphphp123321/mahjong-goserver/osutils"
	pb "github.com/hphphp123321/mahjong-goserver/services/mahjong/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"
)

var (
	playerName string
	address    string
	port       int
	timeout    int
	timeTicker int
	logFormat  string
	logLevel   string
	logOutput  string
	logFile    string
)

func parseFlags() {
	flag.StringVar(&playerName, "playerName", "player2", "player name")
	flag.StringVar(&address, "address", "127.0.0.1", "server address")
	flag.IntVar(&port, "port", 16548, "port")
	flag.IntVar(&timeout, "timeout", 5, "seconds for timeout")
	flag.IntVar(&timeTicker, "timeTicker", 10, "seconds for time ticker")
	flag.StringVar(&logFormat, "logFormat", "text", "log format(json or text)")
	flag.StringVar(&logLevel, "logLevel", "debug", "log level(debug, info, warn, error, fatal, panic)")
	flag.StringVar(&logOutput, "logOutput", "stdout", "log output(stdout or stderr)")
	flag.StringVar(&logFile, "logFile", "", "log file path")
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

func setupLogger() {
	switch logFormat {
	case "text":
		log.SetFormatter(&log.TextFormatter{
			ForceColors:               true,
			TimestampFormat:           "2006-01-02 15:04:05",
			FullTimestamp:             true,
			EnvironmentOverrideColors: true,
			CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
				//处理文件名
				fileName := path.Base(frame.File)
				return ": " + strconv.Itoa(frame.Line), fileName
			},
		})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.Error("set log format error")
	}

	switch logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.Error("set log level error")
	}

	switch logOutput {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "stderr":
		log.SetOutput(os.Stderr)
	default:
		log.Error("set log output error")
	}

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(f)
		} else {
			log.Error("set log file error")
		}
	}

}

func main() {
	parseFlags()
	setupLogger()
	log.Debug("Hello World!")

	var kacp = keepalive.ClientParameters{
		Time:                time.Duration(timeTicker),
		Timeout:             time.Duration(timeout),
		PermitWithoutStream: true,
	}

	tcpAddr := fmt.Sprintf("%s:%d", address, port)
	log.Debug("Start dial tcpAddr: ", tcpAddr)
	conn, err := grpc.Dial(tcpAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithKeepaliveParams(kacp))

	//conn, err := grpc.Dial(tcpAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(unaryInterceptor), grpc.WithKeepaliveParams(kacp))
	defer conn.Close()
	if err != nil {
		log.Fatalf("can not dial: %v", err)
	}

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

	go func() {
		exitChannel := osutils.NewShutdownSignal()
		for {
			osutils.WaitExit(exitChannel, func() {
				log.Println("Exit")
				if err = c.Logout(); err != nil {
					log.Fatalf("Logout failed: %v", err)
				}
			})
		}
	}()

	defer func() {
		err := c.Logout()
		if err != nil {
			log.Warning("Logout failed: %v", err)
		}
	}()

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
		log.Warning("could not refresh room: %v", err)
	}

	// JoinRoom
	if len(c.RoomList) > 0 {
		err = c.JoinRoom(c.RoomList[0].RoomID.String())
		if err != nil {
			log.Warning("could not join room: %v", err)
		}
	}

	// CreateRoom
	if c.Room == nil {
		err = c.CreateRoom("room1")
		if err != nil {
			log.Warning("could not create room: %v", err)
		}
	}

	err = c.Ready()
	if err != nil {
		log.Warning("could not ready: %v", err)
	}

}
