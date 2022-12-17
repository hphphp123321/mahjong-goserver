package simple

import (
	"errors"
	"mahjong-goserver/player"
	"mahjong-goserver/robots"
	pb "mahjong-goserver/services/mahjong/v1"
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
