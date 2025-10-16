package ppu

type colorDepth uint16

const (
	bpp2 colorDepth = 2
	bpp4 colorDepth = 4
	bpp8 colorDepth = 8
)

type bitPlaneRenderer func([]uint16, uint16, *[8][8]byte) // The PPU provides access to the epoch relevant to a specific BG layer

type BGEpochSource interface {
	GetBGSourceEpoch() *uint64
}

type Background interface {
	GetTileMapAddress() uint16
	GetTileMapSize() byte
	GetCharTileAddress() uint16
	GetCharTileSize() byte
	GetColorDepth() byte
	IsOffsetPerTile() bool
}

type Background1 struct {
	ds tileDataSource

	tileMap        [0x1000]BgTile //4x400
	tileMapAddress uint16
	tileMapSize    uint16

	charTiles           map[uint16]*CharTile
	charTileAddressBase uint16
	charTileSize        byte
	colorDepth          colorDepth

	vScroll uint16
	hScroll uint16

	currentEpoch *uint64
}

func NewBackground1(ds tileDataSource, epochPtr *uint64) *Background1 {
	bg := &Background1{
		ds:           ds,
		charTiles:    make(map[uint16]*CharTile),
		currentEpoch: epochPtr,
	}

	for i := range bg.tileMap {
		bg.tileMap[i].bg = bg
	}

	return bg
}

func (bg1 *Background1) Invalidate(addr uint16) {
	if bg1.tileMapAddress <= addr && bg1.tileMapAddress+tileMapDimensionsLUT[bg1.tileMapSize].wordSize > addr {
		index := addr - bg1.tileMapAddress
		if index < uint16(len(bg1.tileMap)) {
			bg1.tileMap[index].isValid = false
			//fmt.Println("invlidation")
		}
		return
	}

	if addr >= bg1.charTileAddressBase {
		wordsPerTile := uint16(bg1.colorDepth) * 4
		tileIndex := (addr - bg1.charTileAddressBase) / wordsPerTile
		if t, ok := bg1.charTiles[bg1.charTileAddressBase+(tileIndex*wordsPerTile)]; ok {
			t.isValid = false
			//fmt.Println("invlidation")
		}
	}
}

// in theory this entire thing can be cached between scrolling changes
func getTileIndexAndPixelCoordinates(tileMapSize uint16, charTileSize byte, H, V uint16) (byte, byte, byte, uint16) {
	tileDimensions := tileMapDimensionsLUT[tileMapSize]
	charDimensions := charTileSizeLUT[charTileSize]
	rowCnt := (V >> charDimensions.divMask) & tileDimensions.modMaskH
	columnCnt := (H >> charDimensions.divMask) & tileDimensions.modMaskW
	tileMapID := (rowCnt>>5)*tileDimensions.mapsPerRow + columnCnt>>5
	row := byte(V & charDimensions.modMask)
	px := byte(H & charDimensions.modMask)
	charMapID := (row>>3)<<1 + (px >> 3)
	tileIndex := tileMapID*0x400 + rowCnt<<5 + columnCnt

	return px, row, charMapID, tileIndex
}

// TODO this can be optimized like crazy
// save the char reference in the tile
// save the char address in the chartile
// basically free pixels
// the previously read tile can also be cached so its only 1 tile lookup instead of 64 per tile
func (bg1 *Background1) GetDotAt(H, V byte) uint16 {
	hScroll := uint16(H) + bg1.hScroll
	vScroll := uint16(V) + bg1.vScroll

	px, row, charMapID, tileIndex := getTileIndexAndPixelCoordinates(bg1.tileMapSize, bg1.charTileSize, hScroll, vScroll)

	tile := bg1.tileMap[tileIndex]
	tile.setup(tileIndex)

	px = tileFlipXLUT[tile.flipIndex][px]
	row = tileFlipYLUT[tile.flipIndex][row]

	if bg1.charTileSize == 1 {
		charMapID += compositeFlipLUT[charMapID][tile.flipIndex]
	}

	//TODO charaddress can also be cached in the bgtile. this is a pointless calculation
	charAddress := (tile.charIndex+charMapIdToOffsetLUT[charMapID])*uint16(bg1.colorDepth<<2) + bg1.charTileAddressBase
	char := bg1.charTiles[charAddress]
	if char == nil {
		char = &CharTile{isValid: false, ds: bg1.ds, tileAddress: charAddress, bg: bg1}
		bg1.charTiles[charAddress] = char
	}

	return bg1.ds.getCGRAM()[char.getPixelAt(bg1.colorDepth, px, row)+tile.paletteNum<<bg1.colorDepth]
}

type BgTile struct {
	isValid                      bool
	verticalFlip, horizontalFlip bool
	flipIndex                    byte

	priority   byte
	paletteNum byte
	charIndex  uint16

	lastRenderEpoch uint64
	bg              *Background1
}

func (bt *BgTile) setup(tileIndex uint16) {
	currentEpoch := *bt.bg.currentEpoch
	if bt.isValid && bt.lastRenderEpoch == currentEpoch {
		return
	}

	params := bt.bg.ds.getVRAM()[bt.bg.tileMapAddress+tileIndex]
	bt.flipIndex = byte((params >> 14) & 3)
	bt.priority = byte(params>>13) & 1
	bt.paletteNum = byte(params>>10) & 7
	bt.charIndex = params & 0x3FF

	bt.isValid = true
	bt.lastRenderEpoch = currentEpoch
}

// TODO chartile needs to be able to handle 16x16 tiles later on too
type CharTile struct {
	isValid bool

	renderer     bitPlaneRenderer
	resolvedData [8][8]byte

	tileAddress uint16
	ds          tileDataSource

	lastRenderEpoch uint64
	bg              *Background1
}

func (ct *CharTile) setup(bitPlanes colorDepth) {
	switch bitPlanes {
	case 2:
		ct.renderer = RenderTile2bppLUT
	case 4:
		ct.renderer = RenderTile4bppLUT
	case 8:
		ct.renderer = RenderTile8bppLUT
	}
}

func (ct *CharTile) getPixelAt(bitplanes colorDepth, px, row byte) byte {
	currentEpoch := *ct.bg.currentEpoch
	if ct.lastRenderEpoch != currentEpoch {
		goto RENDER_AND_CACHE
	}

	if !ct.isValid {
		goto RENDER_AND_CACHE
	}

	return ct.resolvedData[row][px]

RENDER_AND_CACHE:
	ct.setup(bitplanes)
	ct.renderer(ct.ds.getVRAM(), ct.tileAddress, &ct.resolvedData)
	ct.isValid = true
	ct.lastRenderEpoch = currentEpoch

	return ct.resolvedData[row][px]
}

// TODO swap this with the lookuptable approach
func resolveWordBitPlanePixel(word uint16, px int) byte {
	return byte(((word >> (7 - px)) & 1) | (((word >> (15 - px)) & 1) << 1))
}

func RenderTile2bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row)]
		for px := range 8 {
			out[row][px] = resolveWordBitPlanePixel(w1, px)
		}
	}
}

func RenderTile4bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row*2)]
		w2 := VRAM[wordBase+uint16(row*2)+1]
		for px := range 8 {
			out[row][px] = resolveWordBitPlanePixel(w1, px) | (resolveWordBitPlanePixel(w2, px) << 2)
		}
	}
}

func RenderTile8bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row*4)]
		w2 := VRAM[wordBase+uint16(row*4)+1]
		w3 := VRAM[wordBase+uint16(row*4)+2]
		w4 := VRAM[wordBase+uint16(row*4)+3]
		for px := range 8 {
			out[row][px] = resolveWordBitPlanePixel(w1, px) | (resolveWordBitPlanePixel(w2, px) << 2) |
				(resolveWordBitPlanePixel(w3, px) << 4) | (resolveWordBitPlanePixel(w4, px) << 6)
		}
	}
}

func RenderTile2bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		word := VRAM[wordBase+uint16(row)]
		low := byte(word)
		high := byte(word >> 8)

		lowBits := bitplaneLUT[low]
		highBits := bitplaneLUT[high]

		for px := range 8 {
			out[row][px] = lowBits[px] | (highBits[px] << 1)
		}
	}
}

func RenderTile4bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row*2)]
		w2 := VRAM[wordBase+uint16(row*2)+1]

		bp0 := bitplaneLUT[byte(w1)]
		bp1 := bitplaneLUT[byte(w1>>8)]
		bp2 := bitplaneLUT[byte(w2)]
		bp3 := bitplaneLUT[byte(w2>>8)]

		for px := range 8 {
			out[row][px] = bp0[px] | (bp1[px] << 1) | (bp2[px] << 2) | (bp3[px] << 3)
		}
	}
}

func RenderTile8bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		offset := wordBase + uint16(row*4)

		bp0 := bitplaneLUT[byte(VRAM[offset])]
		bp1 := bitplaneLUT[byte(VRAM[offset]>>8)]
		bp2 := bitplaneLUT[byte(VRAM[offset+1])]
		bp3 := bitplaneLUT[byte(VRAM[offset+1]>>8)]
		bp4 := bitplaneLUT[byte(VRAM[offset+2])]
		bp5 := bitplaneLUT[byte(VRAM[offset+2]>>8)]
		bp6 := bitplaneLUT[byte(VRAM[offset+3])]
		bp7 := bitplaneLUT[byte(VRAM[offset+3]>>8)]

		for px := range 8 {
			out[row][px] = bp0[px] | (bp1[px] << 1) | (bp2[px] << 2) | (bp3[px] << 3) |
				(bp4[px] << 4) | (bp5[px] << 5) | (bp6[px] << 6) | (bp7[px] << 7)
		}
	}
}
