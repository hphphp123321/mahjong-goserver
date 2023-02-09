package room

import (
	"errors"
	"github.com/google/uuid"
	"github.com/hphphp123321/mahjong-goserver/common"
	"github.com/hphphp123321/mahjong-goserver/player"
	"sync"
)

type Room struct {
	RoomID      uuid.UUID      `json:"room_id"`
	RoomName    string         `json:"room_name"`
	PlayerCount int            `json:"player_count"`
	Owner       *player.Player `json:"owner"`

	IdleSeats []int            `json:"idle_seats"`
	Players   []*player.Player `json:"players"`

	mu sync.RWMutex
}

func (r *Room) AddRobot(p *player.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.PlayerCount == 4 {
		return errors.New("room is full")
	}
	if !common.Contain(p.Seat, r.IdleSeats) {
		return errors.New("seat already used")
	}
	r.Players = append(r.Players, p)
	idleSeats, err := common.Remove(p.Seat, r.IdleSeats)
	if err != nil {
		return err
	}
	r.IdleSeats = idleSeats.([]int)
	r.PlayerCount++
	return nil
}

func (r *Room) AddPlayer(p *player.Player) error {
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, v := range r.Players {
		if v.Token == p.Token {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			r.PlayerCount--
			r.IdleSeats = append(r.IdleSeats, p.Seat)
			//if r.PlayerCount > 0 {
			//	r.OwnerName = r.Players[0].PlayerName
			//}
			if p == r.Owner {
				if r.PlayerCount > 0 {
					r.Owner = r.Players[0]
				} else {
					r.Owner = nil
				}
			}
			return nil
		}
	}
	return errors.New("player not found")
}

func (r *Room) GetPlayerBySeat(seat int) (*player.Player, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.Players {
		if v.Seat == seat {
			return v, nil
		}
	}
	return nil, errors.New("player in seat not found")
}

func (r *Room) IsFull() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.PlayerCount == 4
}

func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.PlayerCount == 0
}

func (r *Room) CheckAllReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.Players {
		if !v.Ready {
			return false
		}
	}
	return true
}

func (r *Room) GetSeat(p *player.Player) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.Players {
		if v.Token == p.Token {
			return v.Seat, nil
		}
	}
	return -1, errors.New("player not found")
}

func NewRoom(roomID uuid.UUID, roomName string, owner *player.Player) *Room {
	return &Room{
		RoomID:      roomID,
		RoomName:    roomName,
		PlayerCount: 0,
		Owner:       owner,
		IdleSeats:   []int{0, 1, 2, 3},
	}
}
