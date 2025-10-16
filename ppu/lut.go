package ppu

var bitplaneLUT [256][8]byte
var tileMapDimensionsLUT = [4]struct{ W, H, mapsPerRow, mapsPerColumn, wordSize, divMaskW, divMaskH, modMaskW, modMaskH uint16 }{
	{32, 32, 1, 1, 0x400, 5, 5, 31, 31},
	{64, 32, 2, 1, 0x800, 6, 5, 63, 31},
	{32, 64, 1, 2, 0x800, 5, 6, 31, 63},
	{64, 64, 2, 2, 0x1000, 6, 6, 63, 63},
}
var charTileSizeLUT = [2]struct{ W, H, divMask, modMask uint16 }{
	{8, 8, 3, 7},
	{16, 16, 4, 15},
}

var charMapIdToOffsetLUT = [4]uint16{0, 1, 0x10, 0x11}
var charMapIdTransformFlipLUT = [4]struct{ H, V int8 }{
	{1, 2}, {-1, 1}, {1, -2}, {-1, -2},
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
