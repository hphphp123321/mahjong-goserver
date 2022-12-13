package v1

import (
	"context"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"log"
	"mahjong-goserver/room"
	pb "mahjong-goserver/services/mahjong/v1"
	"time"
)

type MahjongClient struct {
	Client      pb.MahjongClient
	ReadyStream pb.Mahjong_ReadyClient
	StartStream pb.Mahjong_StartClient

	Delay      time.Duration
	Ctx        context.Context
	PlayerName string
	Token      uuid.UUID

	Room *room.Room
}

func NewMahjongClient(ctx context.Context, playerName string, grpcClient pb.MahjongClient) *MahjongClient {
	return &MahjongClient{
		Client:     grpcClient,
		PlayerName: playerName,
		Ctx:        ctx,
	}
}

func (c *MahjongClient) Ping() error {
	start := time.Now()
	pingReply, err := c.Client.Ping(c.Ctx, &pb.PingRequest{Message: "ping"})
	end := time.Now()
	usedTime := end.Sub(start)
	c.Delay = usedTime
	if err != nil {
		return err
	}
	log.Printf("Ping: %s, %s", usedTime, pingReply.Message)
	return nil
}

func (c *MahjongClient) Login() error {
	log.Printf("Start Login: playerName: %s", c.PlayerName)
	loginReply, err := c.Client.Login(c.Ctx, &pb.LoginRequest{
		PlayerName: c.PlayerName,
	})
	if err != nil {
		return err
	}
	c.Token = uuid.MustParse(loginReply.Token)
	log.Printf("Login: %s, UUID: %s", loginReply.Message, loginReply.Token)
	header := metadata.New(map[string]string{"token": loginReply.Token})
	c.Ctx = metadata.NewOutgoingContext(c.Ctx, header)
	c.ReadyStream, err = c.Client.Ready(c.Ctx)
	if err != nil {
		return err
	}
	c.StartStream, err = c.Client.Start(c.Ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *MahjongClient) Logout() error {
	log.Printf("Start Logout: playerName: %s", c.PlayerName)
	logoutReply, err := c.Client.Logout(c.Ctx, &pb.LogoutRequest{
		Token: c.Token.String(),
	})
	if err != nil {
		return err
	}
	log.Printf("Logout: %s", logoutReply.Message)
	return nil
}

func (c *MahjongClient) RefreshRoom(roomName string) error {
	log.Printf("Start RefreshRoom: playerName: %s", c.PlayerName)
	refreshRoomReply, err := c.Client.RefreshRoom(c.Ctx, &pb.RefreshRoomRequest{
		RoomName: &roomName,
	})
	if err != nil {
		return err
	}
	roomSlice := refreshRoomReply.Rooms
	println(len(roomSlice))
	log.Printf("RefreshRoom: %s", refreshRoomReply.Message)
	return nil
}

func (c *MahjongClient) CreateRoom(roomName string) error {
	log.Printf("Start CreateRoom: playerName: %s", c.PlayerName)
	createRoomReply, err := c.Client.CreateRoom(c.Ctx, &pb.CreateRoomRequest{
		RoomName: roomName,
	})
	if err != nil {
		return err
	}
	c.Room = room.NewRoom(createRoomReply.Room.RoomID, createRoomReply.Room.RoomName, int(createRoomReply.Room.PlayerCount), createRoomReply.Room.OwnerName)
	log.Printf("CreateRoom: %s", createRoomReply.Message)
	return nil
}
