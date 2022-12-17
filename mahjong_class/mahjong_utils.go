package mahjong

import (
	"github.com/dnovikoff/tempai-core/compact"
	"github.com/dnovikoff/tempai-core/hand/calc"
	"github.com/dnovikoff/tempai-core/hand/shanten"
	"github.com/dnovikoff/tempai-core/hand/tempai"
)

var YaoKyuTiles = [...]int{0, 8, 9, 17, 18, 26, 27, 28, 29, 30, 31, 32, 33}

func CalculateShantenNum(handTiles Tiles, melds Calls) int {
	handTilesT := IntsToTiles(handTiles)
	generator := compact.NewTileGenerator()
	instances := compact.NewInstances()
	hand := generator.Tiles(handTilesT)
	instances.Add(hand)

	var meldsOpt calc.Option
	meldsT := CallsToMelds(melds)
	meldsOpt = calc.Declared(meldsT)

	res := shanten.Calculate(instances, meldsOpt)
	return res.Total.Value
}

func CalculateTenhaiSlice(handTiles Tiles, melds Calls) []int {
	var tenhaiSlice []int
	handTilesT := IntsToTiles(handTiles)
	generator := compact.NewTileGenerator()
	instances := compact.NewInstances()
	hand := generator.Tiles(handTilesT)
	instances.Add(hand)

	var meldsOpt calc.Option = nil
	if melds != nil {
		meldsT := CallsToMelds(melds)
		meldsOpt = calc.Declared(meldsT)
	}
	res := tempai.Calculate(instances, meldsOpt)
	tiles := tempai.GetWaits(res).Tiles()
	for _, tile := range tiles {
		tenhaiSlice = append(tenhaiSlice, int(tile)-1)
	}
	return tenhaiSlice
}
