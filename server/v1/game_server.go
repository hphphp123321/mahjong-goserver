package v1

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"mahjong-goserver/common"
	"mahjong-goserver/player"
	"mahjong-goserver/robots"
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

	done chan error

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
	c, err := s.getClient(ctx)
	if c.p.Token != uuid.MustParse(in.Token) {
		return nil, errors.New("token not match")
	}
	if err != nil {
		return nil, err
	}
	err = s.LeaveRoom(c)
	if err != nil {
		return nil, err
	}
	log.Printf("Logout: PlayerName: %s, UUID: %s", c.p.PlayerName, c.p.Token.String())
	s.removeClient(c)
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
				OwnerName:   r.Owner.PlayerName,
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
	newRoom := room.NewRoom(roomId, in.RoomName, c.p)
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
	seat, err := joinRoom.GetSeat(c.p)
	log.Printf("JoinRoom: PlayerName: %s, RoomName: %s, Seat: %d", c.p.PlayerName, c.room.RoomName, seat)
	if err != nil {
		return nil, err
	}

	rep := &pb.ReadyReply{
		Message: fmt.Sprintf("player: %s, join room", c.p.PlayerName),
		Reply: &pb.ReadyReply_PlayerJoin{PlayerJoin: &pb.PlayerJoinReply{
			Seat:       int32(seat),
			PlayerName: c.p.PlayerName,
		}},
	}
	err = s.readyBoardCast(c, rep, false)
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
			OwnerName:   joinRoom.Owner.PlayerName,
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
	if c.readyStream != nil {
		return errors.New("already has ready stream")
	}
	c.readyStream = stream
	log.Printf("Start new ReadyStream for player: %s", c.p.PlayerName)
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("receive error %v", err)
				c.done <- errors.New("failed to receive request")
				return
			}
			switch in.GetRequest().(type) {
			case *pb.ReadyRequest_GetReady:
				err = s.handleGetReadyRequest(c, in)
				if err != nil {
					return
				}
			case *pb.ReadyRequest_CancelReady:
				err = s.handleCancelReadyRequest(c, in)
				if err != nil {
					return
				}
			case *pb.ReadyRequest_LeaveRoom:
				err = s.handleLeaveRoomRequest(c, in)
				if err != nil {
					return
				}
			case *pb.ReadyRequest_RemovePlayer:
				err = s.handleRemovePlayerRequest(c, in)
				if err != nil {
					return
				}
			case *pb.ReadyRequest_AddRobot:
				err = s.handleAddRobotRequest(c, in)
				if err != nil {
					return
				}
			case *pb.ReadyRequest_Chat:
				rep := &pb.ReadyReply{
					Message: fmt.Sprintf("player: %s, send chat message", c.p.PlayerName),
					Reply: &pb.ReadyReply_Chat{Chat: &pb.ChatReply{
						Message:    in.GetChat().Message,
						PlayerName: c.p.PlayerName,
					}},
				}
				err = s.readyBoardCast(c, rep, true)
				if err != nil {
					c.done <- err
					return
				}
			}
		}
	}()
	var doneError error
	select {
	case <-ctx.Done():
		doneError = ctx.Err()
	case doneError = <-c.done:
	}
	if err != nil {
		return err
	}
	return doneError
}

func (s *MahjongServer) readyBoardCast(c *client, resp *pb.ReadyReply, includeSelf bool) error {
	if c.room == nil {
		return errors.New("not in room")
	}
	for _, p := range c.room.Players {
		if p.Token == c.p.Token {
			if !includeSelf {
				continue
			}
		}
		if s.clients[p.Token].readyStream == nil {
			return errors.New("don't have ready stream")
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

func (s *MahjongServer) removeClient(c *client) {
	s.mu.Lock()
	delete(s.clients, c.p.Token)
	s.mu.Unlock()
}

// LeaveRoom detect when client disconnects or leaves room
func (s *MahjongServer) LeaveRoom(c *client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.room == nil {
		return nil
	} else {
		roomID := c.room.RoomID
		if err := c.room.RemovePlayer(c.p); err != nil {
			return err
		}
		rep := &pb.ReadyReply{Message: fmt.Sprintf("player: %s, leave room", c.p.PlayerName),
			Reply: &pb.ReadyReply_PlayerLeave{PlayerLeave: &pb.PlayerLeaveReply{
				Seat:       int32(c.p.Seat),
				OwnerSeat:  int32(c.room.Owner.Seat),
				PlayerName: c.p.PlayerName,
			}}}
		if err := s.readyBoardCast(c, rep, true); err != nil {
			return err
		}
		log.Printf("LeaveRoom: PlayerName: %s, RoomName: %s", c.p.PlayerName, c.room.RoomName)
		if c.room.IsEmpty() {
			delete(s.rooms, roomID)
			log.Printf("Room %s is empty, delete", roomID.String())
		}
		c.room = nil
		c.readyStream = nil
	}
	return nil
}

func (s *MahjongServer) sendReadyNotify(c *client, msg string) error {
	var err error
	rep := &pb.ReadyReply{
		Message: msg,
	}
	err = c.readyStream.Send(rep)
	if err != nil {
		c.done <- err
		return err
	}
	return nil
}

func (s *MahjongServer) handleGetReadyRequest(c *client, in *pb.ReadyRequest) error {
	var err error
	log.Printf("GetReady Req: PlayerName: %s, RoomName: %s, request: %s", c.p.PlayerName, c.room.RoomName, in.GetGetReady().String())
	c.p.SetReady(true)

	rep := &pb.ReadyReply{
		Message: fmt.Sprintf("player: %s, get ready success", c.p.PlayerName),
		Reply: &pb.ReadyReply_GetReady{GetReady: &pb.GetReadyReply{
			Seat:       int32(c.p.Seat),
			PlayerName: c.p.PlayerName,
		}},
	}
	err = s.readyBoardCast(c, rep, true)
	if err != nil {
		c.done <- err
		return err
	}
	return nil
}

func (s *MahjongServer) handleCancelReadyRequest(c *client, in *pb.ReadyRequest) error {
	var err error
	log.Printf("CancelReady Req: PlayerName: %s, RoomName: %s, request: %s", c.p.PlayerName, c.room.RoomName, in.GetCancelReady().String())
	c.p.SetReady(false)

	rep := &pb.ReadyReply{
		Message: fmt.Sprintf("player: %s, cancel ready success", c.p.PlayerName),
		Reply: &pb.ReadyReply_CancelReady{CancelReady: &pb.CancelReadyReply{
			Seat:       int32(c.p.Seat),
			PlayerName: c.p.PlayerName,
		}},
	}
	err = s.readyBoardCast(c, rep, true)
	if err != nil {
		c.done <- err
		return err
	}
	return nil
}

func (s *MahjongServer) handleLeaveRoomRequest(c *client, in *pb.ReadyRequest) error {
	var err error
	log.Printf("LeaveRoom Req: PlayerName: %s, RoomName: %s, request: %s", c.p.PlayerName, c.room.RoomName, in.GetLeaveRoom().String())
	err = s.LeaveRoom(c)
	if err != nil {
		c.done <- err
		return err
	}
	return nil
}

func (s *MahjongServer) handleRemovePlayerRequest(c *client, in *pb.ReadyRequest) error {
	var err error
	if c.room.Owner != c.p {
		err = s.sendReadyNotify(c, fmt.Sprintf("player: %s, is not owner, can't remove player", c.p.PlayerName))
		if err != nil {
			return err
		}
	}
	log.Printf("RemovePlayer Req: PlayerName: %s, RoomName: %s, request: %s", c.p.PlayerName, c.room.RoomName, in.GetRemovePlayer().String())
	seat := int(in.GetRemovePlayer().PlayerSeat)
	p, err := c.room.GetPlayerBySeat(seat)
	if err != nil {
		return err
	}
	if p == c.p {
		err = s.sendReadyNotify(c, fmt.Sprintf("player: %s, can't remove self", c.p.PlayerName))
		if err != nil {
			return err
		}
	}
	err = c.room.RemovePlayer(p)
	if err != nil {
		c.done <- err
		return err
	}
	rep := &pb.ReadyReply{
		Message: fmt.Sprintf("player: %s, remove player: %s success", c.p.PlayerName, p.PlayerName),
		Reply: &pb.ReadyReply_PlayerLeave{PlayerLeave: &pb.PlayerLeaveReply{
			Seat:       int32(p.Seat),
			PlayerName: p.PlayerName,
			OwnerSeat:  int32(c.room.Owner.Seat),
		}},
	}
	err = s.readyBoardCast(c, rep, true)
	if err != nil {
		c.done <- err
		return err
	}
	return nil
}

func (s *MahjongServer) handleAddRobotRequest(c *client, in *pb.ReadyRequest) error {
	var err error
	if c.room.Owner != c.p {
		err = s.sendReadyNotify(c, fmt.Sprintf("player: %s, is not owner, can't add robot", c.p.PlayerName))
		if err != nil {
			return err
		}
	}
	log.Printf("AddRobot Req: PlayerName: %s, RoomName: %s, request: %s", c.p.PlayerName, c.room.RoomName, in.GetAddRobot().String())
	seat := int(in.GetAddRobot().RobotSeat)
	if !common.Contain(seat, c.room.IdleSeats) {
		err = s.sendReadyNotify(c, fmt.Sprintf("seat %v not valid", seat))
		if err != nil {
			return err
		}
	}
	level := in.GetAddRobot().RobotLevel
	robot, err := robots.GetRobot(level)
	if err != nil {
		err = s.sendReadyNotify(c, fmt.Sprintf("robot level %s not valid", level))
		if err != nil {
			return err
		}
	}
	robotPlayer := player.NewRobot(level, seat, robot)
	err = c.room.AddRobot(robotPlayer)
	if err != nil {
		err = s.sendReadyNotify(c, err.Error())
		if err != nil {
			return err
		}
	}
	rep := &pb.ReadyReply{
		Message: fmt.Sprintf("player: %s, add robot: %s success", c.p.PlayerName, robotPlayer.PlayerName),
		Reply: &pb.ReadyReply_AddRobot{AddRobot: &pb.AddRobotReply{
			RobotSeat:  int32(robotPlayer.Seat),
			RobotLevel: level,
		}},
	}
	err = s.readyBoardCast(c, rep, true)
	if err != nil {
		c.done <- err
		return err
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
