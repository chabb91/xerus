package ppu

var bitplaneLUT [256][8]byte
var tileMapDimensionsLUT = [4]struct{ W, H, mapsPerRow, mapsPerColumn, wordSize, divMaskW, divMaskH, modMaskW, modMaskH, mapsPerRowMinusOne uint16 }{
	{32, 32, 1, 1, 0x400, 5, 5, 31, 31, 0},
	{64, 32, 2, 1, 0x800, 6, 5, 63, 31, 1},
	{32, 64, 1, 2, 0x800, 5, 6, 31, 63, 0},
	{64, 64, 2, 2, 0x1000, 6, 6, 63, 63, 1},
}
var charTileSizeLUT = [4]struct{ W, H, divMaskW, divMaskH, modMaskW, modMaskH uint16 }{
	//normal
	{8, 8, 3, 3, 7, 7},
	{16, 16, 4, 4, 15, 15},
	//hires
	{16, 8, 4, 3, 15, 7},
	{16, 16, 4, 4, 15, 15},
}

var obTileSizeLUT = [8][2]ObTileSize{
	{
		newObTileSize(8, 8, 3, 3, 7, 7, 1, 1),
		newObTileSize(16, 16, 4, 4, 15, 15, 2, 2),
	},
	{
		newObTileSize(8, 8, 3, 3, 7, 7, 1, 1),
		newObTileSize(32, 32, 5, 5, 31, 31, 4, 4),
	},
	{
		newObTileSize(8, 8, 3, 3, 7, 7, 1, 1),
		newObTileSize(64, 64, 6, 6, 63, 63, 8, 8),
	},
	{
		newObTileSize(16, 16, 4, 4, 15, 15, 2, 2),
		newObTileSize(32, 32, 5, 5, 31, 31, 4, 4),
	},
	{
		newObTileSize(16, 16, 4, 4, 15, 15, 2, 2),
		newObTileSize(64, 64, 6, 6, 63, 63, 8, 8),
	},
	{
		newObTileSize(32, 32, 5, 5, 31, 31, 4, 4),
		newObTileSize(64, 64, 6, 6, 63, 63, 8, 8),
	},
	{
		newObTileSize(16, 32, 3, 4, 15, 31, 2, 4),
		newObTileSize(32, 64, 5, 6, 31, 63, 4, 8),
	},
	{
		newObTileSize(16, 32, 3, 4, 15, 31, 2, 4),
		newObTileSize(32, 32, 5, 5, 31, 31, 4, 4),
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
