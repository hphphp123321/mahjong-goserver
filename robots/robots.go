package robots

import (
	"errors"
	"github.com/hphphp123321/mahjong-goserver/player"
)

var (
	RobotsRegistry = map[string]func() player.GameAgent{}
)

func RegisterRobot(name string, robot func() player.GameAgent) {
	RobotsRegistry[name] = robot
}

func GetRobot(name string) (player.GameAgent, error) {
	if f, ok := RobotsRegistry[name]; ok {
		return f(), nil
	} else {
		return nil, errors.New("robot not found")
	}
}
