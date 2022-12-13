package room

type Room struct {
	RoomID      string
	RoomName    string
	PlayerCount int
	OwnerName   string
}

func NewRoom(roomID string, roomName string, playerCount int, masterName string) *Room {
	return &Room{
		RoomID:      roomID,
		RoomName:    roomName,
		PlayerCount: playerCount,
		OwnerName:   masterName,
	}
}
