package room

import (
	"errors"
	"github.com/google/uuid"
	"mahjong-goserver/player"
)

type Room struct {
	RoomID      uuid.UUID
	RoomName    string
	PlayerCount int
	OwnerName   string

	IdleSeats []int
	Players   []*player.Player
}

func (r *Room) AddPlayer(p *player.Player) error {
	if r.PlayerCount == 4 {
		return errors.New("room is full")
	}
	r.Players = append(r.Players, p)
	p.Seat = r.IdleSeats[0]
	r.IdleSeats = r.IdleSeats[1:]
	r.PlayerCount++
	return nil
}

func (r *Room) RemovePlayer(p *player.Player) error {
	for i, v := range r.Players {
		if v.Token == p.Token {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			r.PlayerCount--
			r.IdleSeats = append(r.IdleSeats, p.Seat)
			//if r.PlayerCount > 0 {
			//	r.OwnerName = r.Players[0].PlayerName
			//}
			return nil
		}
	}
	return errors.New("player not found")
}

func (r *Room) IsFull() bool {
	return r.PlayerCount == 4
}

func (r *Room) IsEmpty() bool {
	return r.PlayerCount == 0
}

func (r *Room) CheckAllReady() bool {
	for _, v := range r.Players {
		if !v.Ready {
			return false
		}
	}
	return true
}

func (r *Room) GetSeat(p *player.Player) (int, error) {
	for _, v := range r.Players {
		if v.Token == p.Token {
			return v.Seat, nil
		}
	}
	return -1, errors.New("player not found")
}

func NewRoom(roomID uuid.UUID, roomName string, masterName string) *Room {
	return &Room{
		RoomID:      roomID,
		RoomName:    roomName,
		PlayerCount: 0,
		OwnerName:   masterName,
		IdleSeats:   []int{0, 1, 2, 3},
	}
}
