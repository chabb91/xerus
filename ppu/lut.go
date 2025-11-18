package ppu

var bitplaneLUT [256][8]byte
var tileMapDimensionsLUT = [4]struct{ W, H, mapsPerRow, mapsPerColumn, wordSize, divMaskW, divMaskH, modMaskW, modMaskH, mapsPerRowMinusOne uint16 }{
	{32, 32, 1, 1, 0x400, 5, 5, 31, 31, 0},
	{64, 32, 2, 1, 0x800, 6, 5, 63, 31, 1},
	{32, 64, 1, 2, 0x800, 5, 6, 31, 63, 0},
	{64, 64, 2, 2, 0x1000, 6, 6, 63, 63, 1},
}
var charTileSizeLUT = [2]struct{ W, H, divMask, modMask uint16 }{
	{8, 8, 3, 7},
	{16, 16, 4, 15},
}

var charMapIdToOffsetLUT = [4]uint16{0, 1, 0x10, 0x11}

type OBTileSize struct {
	W, H                        uint16
	divMaskW, divMaskH          uint16
	modMaskW, modMaskH          uint16
	tilesPerRow, tilesPerColumn uint16
}

var obTileSizeLUT = [8][2]OBTileSize{
	{
		{W: 8, H: 8, divMaskW: 3, divMaskH: 3, modMaskW: 7, modMaskH: 7, tilesPerRow: 1, tilesPerColumn: 1},
		{W: 16, H: 16, divMaskW: 4, divMaskH: 4, modMaskW: 15, modMaskH: 15, tilesPerRow: 2, tilesPerColumn: 2},
	},
	{
		{W: 8, H: 8, divMaskW: 3, divMaskH: 3, modMaskW: 7, modMaskH: 7, tilesPerRow: 1, tilesPerColumn: 1},
		{W: 32, H: 32, divMaskW: 5, divMaskH: 5, modMaskW: 31, modMaskH: 31, tilesPerRow: 4, tilesPerColumn: 4},
	},
	{
		{W: 8, H: 8, divMaskW: 3, divMaskH: 3, modMaskW: 7, modMaskH: 7, tilesPerRow: 1, tilesPerColumn: 1},
		{W: 64, H: 64, divMaskW: 6, divMaskH: 6, modMaskW: 63, modMaskH: 63, tilesPerRow: 8, tilesPerColumn: 8},
	},
	{
		{W: 16, H: 16, divMaskW: 4, divMaskH: 4, modMaskW: 15, modMaskH: 15, tilesPerRow: 2, tilesPerColumn: 2},
		{W: 32, H: 32, divMaskW: 5, divMaskH: 5, modMaskW: 31, modMaskH: 31, tilesPerRow: 4, tilesPerColumn: 4},
	},
	{
		{W: 16, H: 16, divMaskW: 4, divMaskH: 4, modMaskW: 15, modMaskH: 15, tilesPerRow: 2, tilesPerColumn: 2},
		{W: 64, H: 64, divMaskW: 6, divMaskH: 6, modMaskW: 63, modMaskH: 63, tilesPerRow: 8, tilesPerColumn: 8},
	},
	{
		{W: 32, H: 32, divMaskW: 5, divMaskH: 5, modMaskW: 31, modMaskH: 31, tilesPerRow: 4, tilesPerColumn: 4},
		{W: 64, H: 64, divMaskW: 6, divMaskH: 6, modMaskW: 63, modMaskH: 63, tilesPerRow: 8, tilesPerColumn: 8},
	},
	{
		{W: 16, H: 32, divMaskW: 3, divMaskH: 4, modMaskW: 15, modMaskH: 31, tilesPerRow: 2, tilesPerColumn: 4},
		{W: 32, H: 64, divMaskW: 5, divMaskH: 6, modMaskW: 31, modMaskH: 63, tilesPerRow: 4, tilesPerColumn: 8},
	},
	{
		{W: 16, H: 32, divMaskW: 3, divMaskH: 4, modMaskW: 15, modMaskH: 31, tilesPerRow: 2, tilesPerColumn: 4},
		{W: 32, H: 32, divMaskW: 5, divMaskH: 5, modMaskW: 31, modMaskH: 31, tilesPerRow: 4, tilesPerColumn: 4},
	},
}

var tileFlipXLUT [4][8]byte
var tileFlipYLUT [4][8]byte
var compositeFlipLUT [4][4]byte

func init() {
	initBitplaneLUT()
	initTileFlipLUT()
	initCompositeFlipLUT()

	NTSC_TIMING.VisibilityLUTs[false] = GenerateVisibilityLUT(&NTSC_TIMING, false)
	NTSC_TIMING.VisibilityLUTs[true] = GenerateVisibilityLUT(&NTSC_TIMING, true)

	PAL_TIMING.VisibilityLUTs[false] = GenerateVisibilityLUT(&PAL_TIMING, false)
	PAL_TIMING.VisibilityLUTs[true] = GenerateVisibilityLUT(&PAL_TIMING, true)
}

func initBitplaneLUT() {
	// Pre-compute all possible byte -> 8 pixels mappings
	for b := range 256 {
		for px := range 8 {
			bitplaneLUT[b][px] = byte((b >> (7 - px)) & 1)
		}
	}
}

var compositeFlip16x8LUT = [2][4]byte{
	{0, 1, 0, 1},
	{1, 0, 1, 0},
}

func initCompositeFlipLUT() {
	for charMapID := range 4 {
		for flipIndex := range 4 {
			hFlip := (flipIndex & 0b01) != 0
			vFlip := (flipIndex & 0b10) != 0

			row := charMapID >> 1
			col := charMapID & 1

			if hFlip {
				col = 1 - col
			}
			if vFlip {
				row = 1 - row
			}

			compositeFlipLUT[charMapID][flipIndex] = byte(row<<1 | col)
		}
	}
}

func initTileFlipLUT() {
	for flipIndex := range 4 {
		hFlip := (flipIndex & 0b01) != 0
		vFlip := (flipIndex & 0b10) != 0

		for coord := range byte(8) {
			finalX := coord
			if hFlip {
				finalX = 7 - coord
			}
			tileFlipXLUT[flipIndex][coord] = finalX

			finalY := coord
			if vFlip {
				finalY = 7 - coord
			}
			tileFlipYLUT[flipIndex][coord] = finalY
		}
	}
}
