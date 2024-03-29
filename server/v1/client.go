package v1

import (
	"github.com/google/uuid"
	"github.com/hphphp123321/mahjong-goserver/player"
	pb "github.com/hphphp123321/mahjong-goserver/services/mahjong/v1"
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
}

func newClient(playerName string, token uuid.UUID) *client {
	return &client{
		p:        player.NewPlayer(playerName, token),
		lastTime: time.Now(),
		done:     make(chan error),
	}
}

// sendReadyMessage send message to client in ready stage
func (c *client) sendReadyMessage(msg string) error {
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
