package ppu

var bitplaneLUT [256][8]byte
var tileMapDimensionsLUT = [4]struct{ W, H, mapsPerRow, mapsPerColumn, wordSize uint16 }{
	{32, 32, 1, 1, 0x400},
	{64, 32, 2, 1, 0x800},
	{32, 64, 1, 2, 0x800},
	{64, 64, 2, 2, 0x1000},
}
var charTileSizeLUT = [2]struct{ W, H byte }{
	{8, 8},
	{16, 16},
}

func init() {
	initBitplaneLUT()
}

func initBitplaneLUT() {
	// Pre-compute all possible byte -> 8 pixels mappings
	for b := range 256 {
		for px := range 8 {
			bitplaneLUT[b][px] = byte((b >> (7 - px)) & 1)
		}
	}
}
