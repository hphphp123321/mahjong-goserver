package player

import pb "mahjong-goserver/services/mahjong/v1"

type GameAgent interface {
	ChooseAction() (*pb.Action, error)
}
