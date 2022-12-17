package player

import "github.com/google/uuid"

type Player struct {
	PlayerName string
	Token      uuid.UUID

	Seat  int
	Ready bool

	Agent GameAgent
}

func NewPlayer(playerName string, playerID uuid.UUID) *Player {
	return &Player{
		PlayerName: playerName,
		Token:      playerID,
	}
}

func NewRobot(robotName string, seat int, agent GameAgent) *Player {
	return &Player{
		PlayerName: robotName,
		Seat:       seat,
		Agent:      agent,
	}
}

func (p *Player) SetReady(ready bool) {
	p.Ready = ready
}
