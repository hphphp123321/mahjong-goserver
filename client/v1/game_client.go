package v1

import (
	"context"
	"github.com/google/uuid"
	"github.com/hphphp123321/mahjong-goserver/player"
	"github.com/hphphp123321/mahjong-goserver/room"
	pb "github.com/hphphp123321/mahjong-goserver/services/mahjong/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"io"
	"sync"
	"time"
)

type MahjongClient struct {
	Client      pb.MahjongClient
	ReadyStream pb.Mahjong_ReadyClient
	StartStream pb.Mahjong_StartClient

	Delay time.Duration
	Ctx   context.Context

	P *player.Player

	RoomList []*room.Room
	Room     *room.Room
}

func NewMahjongClient(ctx context.Context, playerName string, grpcClient pb.MahjongClient) *MahjongClient {
	return &MahjongClient{
		Client: grpcClient,
		P:      player.NewPlayer(playerName, uuid.Nil),
		Ctx:    ctx,
	}
}

func (c *MahjongClient) Ping() error {
	start := time.Now()
	_, err := c.Client.Ping(c.Ctx, &pb.Empty{})
	end := time.Now()
	usedTime := end.Sub(start)
	c.Delay = usedTime
	if err != nil {
		return err
	}
	//log.Printf("Ping: %s, %s", usedTime, pingReply.Message)
	return nil
}

func (c *MahjongClient) Login() error {
	log.Printf("Start Login: playerName: %s", c.P.PlayerName)
	loginReply, err := c.Client.Login(c.Ctx, &pb.LoginRequest{
		PlayerName: c.P.PlayerName,
	})
	if err != nil {
		return err
	}
	c.P.Token = uuid.MustParse(loginReply.Token)
	log.Printf("Login: %s, UUID: %s", loginReply.Message, loginReply.Token)
	header := metadata.New(map[string]string{"token": loginReply.Token})
	c.Ctx = metadata.NewOutgoingContext(c.Ctx, header)

	c.StartStream, err = c.Client.Start(c.Ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *MahjongClient) Logout() error {
	log.Printf("Start Logout: playerName: %s", c.P.PlayerName)
	logoutReply, err := c.Client.Logout(c.Ctx, &pb.Empty{})
	if err != nil {
		return err
	}
	log.Printf("Logout: %s", logoutReply.Message)
	return nil
}

func (c *MahjongClient) RefreshRoom(roomName string) error {
	log.Printf("Start RefreshRoom: playerName: %s", c.P.PlayerName)
	refreshRoomReply, err := c.Client.RefreshRoom(c.Ctx, &pb.RefreshRoomRequest{
		RoomName: &roomName,
	})
	c.RoomList = make([]*room.Room, 0)
	for _, r := range refreshRoomReply.Rooms {
		roomID, err := uuid.Parse(r.RoomID)
		if err != nil {
			return err
		}
		rn := room.NewRoom(roomID, r.RoomName, player.NewPlayer(r.OwnerName, uuid.Nil))
		rn.PlayerCount = int(r.PlayerCount)
		c.RoomList = append(c.RoomList, rn)
	}
	if err != nil {
		return err
	}
	log.Printf("RefreshRoom: %s", refreshRoomReply.Message)
	return nil
}

func (c *MahjongClient) CreateRoom(roomName string) error {
	log.Printf("Start CreateRoom: playerName: %s", c.P.PlayerName)
	createRoomReply, err := c.Client.CreateRoom(c.Ctx, &pb.CreateRoomRequest{
		RoomName: roomName,
	})
	if err != nil {
		return err
	}
	roomID, err := uuid.Parse(createRoomReply.Room.RoomID)
	if err != nil {
		return err
	}
	c.Room = room.NewRoom(roomID, createRoomReply.Room.RoomName, player.NewPlayer(createRoomReply.Room.OwnerName, uuid.Nil))
	c.Room.PlayerCount = int(createRoomReply.Room.PlayerCount)
	log.Printf("CreateRoom: %s", createRoomReply.Message)
	c.ReadyStream, err = c.Client.Ready(c.Ctx)
	if err != nil {
		return err
	}
	log.Printf("Start ReadyStream")
	return nil
}

func (c *MahjongClient) JoinRoom(roomId string) error {
	log.Printf("Start JoinRoom: roomID: %s", roomId)
	joinRoomReply, err := c.Client.JoinRoom(c.Ctx, &pb.JoinRoomRequest{
		RoomID: roomId,
	})
	if err != nil {
		return err
	}
	roomID, err := uuid.Parse(joinRoomReply.Room.RoomID)
	if err != nil {
		return err
	}
	c.Room = room.NewRoom(roomID, joinRoomReply.Room.RoomName, player.NewPlayer(joinRoomReply.Room.OwnerName, uuid.Nil))
	c.Room.PlayerCount = int(joinRoomReply.Room.PlayerCount)
	c.P.Seat = int(joinRoomReply.Seat)
	log.Printf("JoinRoom: %s", joinRoomReply.Message)
	c.ReadyStream, err = c.Client.Ready(c.Ctx)
	if err != nil {
		return err
	}
	log.Printf("Start ReadyStream")
	return nil
}

func (c *MahjongClient) Ready() error {
	var err error
	var wg sync.WaitGroup

	// Receive
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			readyReply, err := c.ReadyStream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			log.Printf("ReadyStream.Recv: %s", readyReply.Message)
			switch readyReply.GetReply().(type) {
			case *pb.ReadyReply_GetReady:
				c.handleGetReadyReply(readyReply)
			}
		}
	}()

	// Send
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.ReadyStream.CloseSend()
		for {
			if !c.P.Ready {
				log.Printf("send ready")
				err = c.ReadyStream.Send(&pb.ReadyRequest{
					Request: &pb.ReadyRequest_GetReady{
						GetReady: &pb.Empty{}}})
				if err != nil {
					log.Printf("ReadyStream.Send: %s", err)
					return
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	wg.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (c *MahjongClient) handleGetReadyReply(readyReply *pb.ReadyReply) {
	if int(readyReply.GetGetReady().Seat) == c.P.Seat {
		c.P.Ready = true
	}
}
