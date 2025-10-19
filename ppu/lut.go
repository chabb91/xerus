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

var tileFlipXLUT [4][8]byte
var tileFlipYLUT [4][8]byte
var compositeFlipLUT [4][4]byte
var baseTileFlipOffsets = [4]struct{ H, V int8 }{
	{1, 2}, {-1, 1}, {1, -2}, {-1, -2},
}

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

			finalOffset := int8(charMapID)

			if hFlip {
				finalOffset += baseTileFlipOffsets[charMapID].H
			}
			if vFlip {
				finalOffset += baseTileFlipOffsets[charMapID].V
			}

			compositeFlipLUT[charMapID][flipIndex] = byte(finalOffset)
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
