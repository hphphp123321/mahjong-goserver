package simple

import (
	"errors"
	"github.com/hphphp123321/mahjong-goserver/player"
	"github.com/hphphp123321/mahjong-goserver/robots"
	pb "github.com/hphphp123321/mahjong-goserver/services/mahjong/v1"
)

type Robot struct {
	player.GameAgent
}

func (r *Robot) ChooseAction() (*pb.Action, error) {
	return nil, errors.New("need Implement")
}

func init() {
	robots.RegisterRobot("Simple", func() player.GameAgent {
		return new(Robot)
	})
}
