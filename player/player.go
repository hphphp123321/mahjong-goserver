package player

import "github.com/google/uuid"

type Player struct {
	PlayerName string
	Token      uuid.UUID

	Seat  int
	Ready bool
}

func NewPlayer(playerName string, playerID uuid.UUID) *Player {
	return &Player{
		PlayerName: playerName,
		Token:      playerID,
	}
}

func (p *Player) SetReady(ready bool) {
	p.Ready = ready
}
