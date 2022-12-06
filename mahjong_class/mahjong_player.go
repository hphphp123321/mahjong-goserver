package mahjong

import "sort"

type MahjongPlayer struct {
	Points          int
	Wind            int
	JunNum          int
	KanNum          int
	HandTiles       Tiles
	DiscardTiles    Tiles
	TilesTsumoGiri  []int
	BoardTiles      Tiles
	Melds           Calls
	TenhaiTiles     Tiles
	ShantenNum      int
	TenhaiSlice     []int
	JunFuriten      bool
	DiscardFuriten  bool
	RiichiFuriten   bool
	IppatsuStatus   bool
	RyuukyokuStatus bool
	FuritenStatus   bool
	IsTsumo         bool
	IsRiichi        bool
	IsIppatsu       bool
	IsRinshan       bool
	IsChankan       bool
	IsHaitei        bool
	IsHoutei        bool
	IsDaburuRiichi  bool
	IsTenhou        bool
	IsChiihou       bool
	Reward          int
	FinalReward     int
	RiichiStep      int
}

func NewMahjongPlayer() *MahjongPlayer {
	p := MahjongPlayer{}
	p.ResetForGame()
	return &p
}

//func (player *MahjongPlayer) GetMelds() calc.Melds {
//	return CallsToMelds(player.melds)
//}

func (player *MahjongPlayer) InitTilesWind(tiles Tiles, wind int) {
	player.HandTiles = tiles
	sort.Ints(player.HandTiles)
	player.Wind = wind
}

func (player *MahjongPlayer) GetShantenNum() int {
	return CalculateShantenNum(player.HandTiles, player.Melds)
}

func (player *MahjongPlayer) GetTenhaiSlice() []int {
	return CalculateTenhaiSlice(player.TenhaiTiles, player.Melds)
}

func (player *MahjongPlayer) GetRiichiTiles() Tiles {
	if player.ShantenNum > 1 && player.JunNum > 1 {
		panic("player's shanten num should not be greater than 1 before Riichi!")
	}
	rTiles := make(Tiles, 0, len(player.HandTiles))
	handTilesCopy := make(Tiles, len(player.HandTiles)-1, len(player.HandTiles))
	for _, tile := range player.HandTiles {
		handTilesCopy = append(handTilesCopy, -1)
		copy(handTilesCopy, player.HandTiles)
		handTilesCopy = handTilesCopy.Remove(tile)
		shantenNum := CalculateShantenNum(handTilesCopy, player.Melds)
		if shantenNum == 0 {
			rTiles = append(rTiles, tile)
		}
	}
	return rTiles
}

func (player *MahjongPlayer) GetHandTilesClass() []int {
	tilesClass := make([]int, 0, len(player.HandTiles))
	for _, tile := range player.HandTiles {
		tilesClass = append(tilesClass, tile/4)
	}
	return tilesClass
}

func (player *MahjongPlayer) IsNagashiMangan() bool {
	if len(player.BoardTiles) != len(player.DiscardTiles) {
		return false
	}
	for _, tileID := range player.DiscardTiles {
		if !Contain(tileID, YaoKyuTiles) {
			return false
		}
	}
	return true
}

func (player *MahjongPlayer) IsFuriten() bool {
	return player.JunFuriten || player.RiichiFuriten || player.DiscardFuriten
}

func (player *MahjongPlayer) ResetForRound() {
	player.Wind = -1
	player.JunNum = 0
	player.KanNum = 0
	player.HandTiles = make(Tiles, 0, 14)
	player.DiscardTiles = make(Tiles, 0, 25)
	player.TilesTsumoGiri = make([]int, 0, 25)
	player.BoardTiles = make(Tiles, 0, 25)
	player.Melds = make([]Call, 0, 4)
	player.TenhaiTiles = make(Tiles, 0, 13)
	player.ShantenNum = 7
	player.TenhaiSlice = []int{}
	player.JunFuriten = false
	player.DiscardFuriten = false
	player.RiichiFuriten = false
	player.IppatsuStatus = true
	player.RyuukyokuStatus = true
	player.FuritenStatus = false
	player.IsTsumo = false
	player.IsRiichi = false
	player.IsIppatsu = false
	player.IsRinshan = false
	player.IsChankan = false
	player.IsHaitei = false
	player.IsHoutei = false
	player.IsDaburuRiichi = false
	player.IsTenhou = false
	player.IsChiihou = false
	player.Reward = 0
	player.RiichiStep = 0
}

func (player *MahjongPlayer) ResetForGame() {
	player.Points = 25000
	player.FinalReward = -1
	player.ResetForRound()
}
