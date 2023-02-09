package mahjong

import (
	"github.com/dnovikoff/tempai-core/hand/calc"
	"github.com/dnovikoff/tempai-core/tile"
	"github.com/hphphp123321/mahjong-goserver/common"
)

func IntsToTiles(tiles Tiles) tile.Tiles {
	tilesT := tile.Tiles{}
	for _, num := range tiles {
		tilesT = append(tilesT, tile.Tile(num/4+1))
	}
	return tilesT
}

func CallToMeld(call Call) calc.Meld {
	var meld calc.Meld
	switch call.CallType {
	case Chi:
		var tileClass int = common.MinNum(call.CallTiles[:3])/4 + 1
		meld = calc.Open(calc.Chi(tile.Tile(tileClass)))
	case Pon:
		var tileClass int = call.CallTiles[0]/4 + 1
		meld = calc.Open(calc.Pon(tile.Tile(tileClass)))
	case DaiMinKan:
		var tileClass int = call.CallTiles[0]/4 + 1
		meld = calc.Open(calc.Kan(tile.Tile(tileClass)))
	case ShouMinKan:
		var tileClass int = call.CallTiles[0]/4 + 1
		meld = calc.Open(calc.Kan(tile.Tile(tileClass)))
	case AnKan:
		var tileClass int = call.CallTiles[0]/4 + 1
		meld = calc.Kan(tile.Tile(tileClass))
	}
	return meld
}

func CallsToMelds(melds Calls) calc.Melds {
	meldsT := calc.Melds{}
	for _, v := range melds {
		meldsT = append(meldsT, CallToMeld(v))
	}
	return meldsT
}
