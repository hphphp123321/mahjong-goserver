package v1

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"log"
	"mahjong-goserver/room"
	pb "mahjong-goserver/services/mahjong/v1"
	"strings"
	"sync"
)

type client struct {
	readyStream pb.Mahjong_ReadyServer
	startStream pb.Mahjong_StartServer

	playerName string
	token      uuid.UUID

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
	if len(s.clients) >= s.maxClients {
		return nil, errors.New("too many clients")
	}
	token := uuid.New()
	s.clients[token] = &client{
		playerName: in.PlayerName,
		token:      token,
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
	err = s.DetectRoom(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Logout: PlayerName: %s, UUID: %s", s.clients[token].playerName, token.String())
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
	newRoom := room.NewRoom(roomId.String(), in.RoomName, 1, c.playerName)
	s.rooms[roomId] = newRoom
	c.room = newRoom
	log.Printf("CreateRoom: PlayerName: %s, RoomName: %s, UUID: %s", c.playerName, in.RoomName, roomId.String())
	return &pb.CreateRoomReply{
		Message: fmt.Sprintf("Create Room Success! Room UUID: %s", roomId.String()),
		Room: &pb.Room{
			RoomID:      roomId.String(),
			RoomName:    in.RoomName,
			PlayerCount: 1,
			OwnerName:   c.playerName,
		}}, nil
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
	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clients[token]
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

// DetectRoom detect when client disconnects or leaves room
func (s *MahjongServer) DetectRoom(playerToken uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[playerToken]
	if !ok {
		return errors.New("invalid token")
	}
	if c.room == nil {
		return nil
	} else {
		roomID, err := uuid.Parse(c.room.RoomID)
		if err != nil {
			return errors.New("invalid room id")
		}
		delete(s.rooms, roomID)
	}
	return nil
}
