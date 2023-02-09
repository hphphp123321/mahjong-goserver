package player

import "github.com/google/uuid"

type Player struct {
	PlayerName string    `json:"player_name"`
	Token      uuid.UUID `json:"token"`

	RoomID uuid.UUID `json:"room_id"`
	Seat   int       `json:"seat"`
	Ready  bool      `json:"ready"`

	Agent GameAgent `json:"-"`
}

func NewPlayer(playerName string, playerID uuid.UUID) *Player {
	return &Player{
		PlayerName: playerName,
		Token:      playerID,
		RoomID:     uuid.Nil,
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
