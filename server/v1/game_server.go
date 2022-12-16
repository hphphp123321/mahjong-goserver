package v1

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"mahjong-goserver/player"
	"mahjong-goserver/room"
	pb "mahjong-goserver/services/mahjong/v1"
	"strings"
	"sync"
	"time"
)

type client struct {
	readyStream pb.Mahjong_ReadyServer
	startStream pb.Mahjong_StartServer

	lastTime time.Time
	online   bool

	p *player.Player
	//playerName string
	//token      uuid.UUID

	room *room.Room
}

type MahjongServer struct {
	pb.UnimplementedMahjongServer
	clients    map[uuid.UUID]*client
	mu         sync.RWMutex
	maxClients int

	rooms map[uuid.UUID]*room.Room
}

func NewMahjongServer(maxClients int) *MahjongServer {
	return &MahjongServer{
		clients:    make(map[uuid.UUID]*client),
		rooms:      make(map[uuid.UUID]*room.Room),
		maxClients: maxClients,
	}
}

func (s *MahjongServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingReply, error) {
	return &pb.PingReply{Message: "pong"}, nil
}

func (s *MahjongServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, p := range s.clients {
		if p.p.PlayerName == in.PlayerName {
			p.online = true
			return &pb.LoginReply{
				Message: "login success",
				Token:   t.String(),
			}, nil
		}
	}
	if len(s.clients) >= s.maxClients {
		return nil, errors.New("too many clients")
	}
	token := uuid.New()
	s.clients[token] = &client{
		p:      player.NewPlayer(in.PlayerName, token),
		online: true,
	}
	log.Printf("Login: playerName: %s, UUID: %s", in.PlayerName, token.String())
	return &pb.LoginReply{
		Message: "login success",
		Token:   token.String(),
	}, nil
}

func (s *MahjongServer) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	token, err := s.getToken(ctx)
	token, err = uuid.Parse(in.Token)
	if err != nil {
		return nil, err
	}
	err = s.LeaveRoom(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Logout: PlayerName: %s, UUID: %s", s.clients[token].p.PlayerName, token.String())
	s.removeClient(token)
	return &pb.LogoutReply{
		Message: "logout success",
	}, nil
}

func (s *MahjongServer) RefreshRoom(ctx context.Context, in *pb.RefreshRoomRequest) (*pb.RefreshRoomReply, error) {
	_, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}
	//if c.room != nil {
	//	return nil, errors.New("already in room")
	//}
	roomSlice := make([]*pb.Room, 0)
	rName := ""
	if in.RoomName != nil {
		rName = *in.RoomName
	}
	for id, r := range s.rooms {
		if strings.Contains(r.RoomName, rName) {
			roomSlice = append(roomSlice, &pb.Room{
				RoomID:      id.String(),
				RoomName:    r.RoomName,
				PlayerCount: int32(r.PlayerCount),
				OwnerName:   r.OwnerName,
			})
		}
	}
	return &pb.RefreshRoomReply{
		Message: "refresh room success, room count: " + fmt.Sprint(len(roomSlice)),
		Rooms:   roomSlice,
	}, nil
}

func (s *MahjongServer) CreateRoom(ctx context.Context, in *pb.CreateRoomRequest) (*pb.CreateRoomReply, error) {
	c, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}
	if c.room != nil {
		return nil, errors.New("already in room")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	roomId := uuid.New()
	newRoom := room.NewRoom(roomId, in.RoomName, c.p.PlayerName)
	err = newRoom.AddPlayer(c.p)
	if err != nil {
		return nil, err
	}
	s.rooms[roomId] = newRoom
	c.room = newRoom
	log.Printf("CreateRoom: PlayerName: %s, RoomName: %s, UUID: %s", c.p.PlayerName, in.RoomName, roomId.String())
	return &pb.CreateRoomReply{
		Message: fmt.Sprintf("Create Room Success! Room UUID: %s", roomId.String()),
		Room: &pb.Room{
			RoomID:      roomId.String(),
			RoomName:    in.RoomName,
			PlayerCount: 1,
			OwnerName:   c.p.PlayerName,
		}}, nil
}

func (s *MahjongServer) JoinRoom(ctx context.Context, in *pb.JoinRoomRequest) (*pb.JoinRoomReply, error) {
	c, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}
	if c.room != nil {
		return nil, errors.New("already in room")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	roomId, err := uuid.Parse(in.RoomID)
	if err != nil {
		return nil, err
	}
	joinRoom, ok := s.rooms[roomId]
	if !ok {
		return nil, errors.New("room not found")
	}
	if joinRoom.IsFull() {
		return nil, errors.New("room is full")
	}
	if err = joinRoom.AddPlayer(c.p); err != nil {
		return nil, err
	}
	c.room = joinRoom
	log.Printf("JoinRoom: PlayerName: %s, RoomName: %s, UUID: %s", c.p.PlayerName, c.room.RoomName, roomId.String())
	seat, err := joinRoom.GetSeat(c.p)
	if err != nil {
		return nil, err
	}
	return &pb.JoinRoomReply{
		Message: fmt.Sprintf("Join Room Success! Room UUID: %s", roomId.String()),
		Seat:    int32(seat),
		Room: &pb.Room{
			RoomID:      roomId.String(),
			RoomName:    joinRoom.RoomName,
			PlayerCount: int32(joinRoom.PlayerCount),
			OwnerName:   joinRoom.OwnerName,
		}}, nil
}

func (s *MahjongServer) Ready(stream pb.Mahjong_ReadyServer) error {
	ctx := stream.Context()
	c, err := s.getClient(ctx)
	if err != nil {
		return err
	}
	if c.room == nil {
		return errors.New("not in room")
	}
	log.Printf("READY receive")
	if c.readyStream != nil {
		return errors.New("already has ready stream")
	}
	c.readyStream = stream
	log.Printf("Start new ReadyStream for player: %s", c.p.PlayerName)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch in.GetRequest().(type) {
		case *pb.ReadyRequest_GetReady:
			log.Printf("GetReady Req: PlayerName: %s, RoomName: %s", c.p.PlayerName, c.room.RoomName)
			c.p.SetReady(true)
			repp := &pb.GetReadyReply{
				Seat:       int32(c.p.Seat),
				PlayerName: c.p.PlayerName,
			}
			rep := &pb.ReadyReply{
				Message: "get ready success",
				Reply:   &pb.ReadyReply_GetReadyReply{GetReadyReply: repp},
			}
			err = c.readyStream.Send(rep)
			if err != nil {
				return err
			}
			err = s.readyBoardCast(ctx, rep)
			if err != nil {
				return err
			}

		}
	}
}

func (s *MahjongServer) readyBoardCast(ctx context.Context, resp *pb.ReadyReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.getClient(ctx)
	if err != nil {
		return err
	}
	if c.room == nil {
		return errors.New("not in room")
	}
	if c.readyStream == nil {
		return errors.New("don't have ready stream")
	}
	for _, p := range c.room.Players {
		if p.PlayerName == c.p.PlayerName {
			continue
		}
		if err := s.clients[p.Token].readyStream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *MahjongServer) getToken(ctx context.Context) (uuid.UUID, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.UUID{}, errors.New("no metadata in context")
	}
	token, err := uuid.Parse(headers["token"][0])
	_, ok = s.clients[token]
	if err != nil || !ok {
		return uuid.UUID{}, errors.New("invalid token")
	}
	return token, nil
}

func (s *MahjongServer) getClient(ctx context.Context) (*client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}
	c, ok := s.clients[token]
	c.lastTime = time.Now()
	if !ok {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

func (s *MahjongServer) removeClient(token uuid.UUID) {
	s.mu.Lock()
	delete(s.clients, token)
	s.mu.Unlock()
}

// LeaveRoom detect when client disconnects or leaves room
func (s *MahjongServer) LeaveRoom(playerToken uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[playerToken]
	if !ok {
		return errors.New("invalid token")
	}
	if c.room == nil {
		return nil
	} else {
		roomID := c.room.RoomID
		if err := c.room.RemovePlayer(c.p); err != nil {
			return err
		}
		log.Printf("LeaveRoom: PlayerName: %s, RoomName: %s", c.p.PlayerName, c.room.RoomName)
		if c.room.IsEmpty() {
			delete(s.rooms, roomID)
			log.Printf("Room %s is empty, delete", roomID.String())
		}
		c.room = nil
	}
	return nil
}

//func (s *MahjongServer) CheckClients() {
//	for {
//		time.Sleep(30 * time.Second)
//		s.mu.Lock()
//		for token, c := range s.clients {
//			if c.online {
//				if time.Since(c.lastTime) > 60*time.Second {
//					c.online = false
//
//				}
//			} else {
//				if time.Since(c.lastTime) > 60*time.Second {
//					if c.room != nil {
//						if err := c.room.RemovePlayer(c.p); err != nil {
//							log.Printf("CheckClients: %s", err)
//						}
//						if c.room.PlayerCount == 0 {
//							delete(s.rooms, c.room.RoomID)
//							log.Printf("Room %s is empty, delete", c.room.RoomID.String())
//						}
//						c.room = nil
//					}
//					delete(s.clients, token)
//				}
//			}
//		}
//		s.mu.Unlock()
//	}
//}
