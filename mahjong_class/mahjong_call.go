package mahjong

type CallType int

const (
	Get CallType = -1 + iota
	Skip
	Discard
	Chi
	Pon
	DaiMinKan
	ShouMinKan
	AnKan
	Riichi
	Ron
	Tsumo
	KyuShuKyuHai
	ChanKan
)

type Calls []Call

type Call struct {
	CallType         CallType `json:"type"`
	CallTiles        Tiles    `json:"tiles"`
	CallTilesFromWho []int    `json:"who"`
}

func NewCall(meldType CallType, CallTiles Tiles, CallTilesFromWho []int) *Call {
	return &Call{
		CallType:         meldType,
		CallTiles:        CallTiles,
		CallTilesFromWho: CallTilesFromWho,
	}
}

func CallEqual(call1 Call, call2 Call) bool {
	if call1.CallType != call2.CallType {
		return false
	}
	if TilesEqual(call1.CallTiles, call2.CallTiles) {
		return true
	}
	return false
}

func (calls *Calls) Index(call Call) int {
	for idx, c := range *calls {
		if CallEqual(c, call) {
			return idx
		}
	}
	panic("call not in calls!")
}
