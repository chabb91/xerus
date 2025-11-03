package ppu

type colorDepth uint16
type ppuLayer uint16

const BG_BACKDROP_COLOR = 0

const (
	bg1      ppuLayer = 0
	bg2      ppuLayer = 1
	bg3      ppuLayer = 2
	bg4      ppuLayer = 3
	bgMode7  ppuLayer = 4
	obj      ppuLayer = 5
	backdrop ppuLayer = 6
)

const (
	bpp2 colorDepth = 2
	bpp4 colorDepth = 4
	bpp8 colorDepth = 8
)

type bitPlaneRenderer func([]uint16, uint16, *[8][8]byte)
type optResolver func(*Background, uint16, uint16) (uint16, uint16)
type VRAMAddressCalculator func(tileIndex byte) uint16
type colorIndex func(ppuLayer, colorDepth, byte) byte

// The PPU provides access to the epoch relevant to a specific BG layer
type LayerEpochSource interface {
	GetLayerSourceEpoch() *uint64
}

type BackgroundI interface {
	GetTileMapAddress() uint16
	GetTileMapSize() byte
	GetCharTileAddress() uint16
	GetCharTileSize() byte
	GetColorDepth() byte
	IsOffsetPerTile() bool
}

type tileAndPixelCacheEntry struct {
	px, row, charMapID byte
	tileIndex          uint16
	entryEpoch         uint64
}

type renderedDotCache struct {
	color    uint16
	priority byte
	H        uint16
}

type Background struct {
	ds tileDataSource

	tileMap        [0x1000]BgTile //4x400
	tileMapAddress uint16
	tileMapSize    uint16

	charTiles           [0x1000]CharTile //VRAM is 0x8000 words. 2bpp takes 8 words to store. there are 0x1000 char tiles max
	charTileAddressBase uint16
	charTileSize        byte
	colorDepth          colorDepth
	getPaletteIndex     colorIndex
	isDirectColor       bool

	vScroll     uint16
	hScroll     uint16
	scrollEpoch uint64

	tileMapLookupCacke [600][600]tileAndPixelCacheEntry

	currentEpoch *uint64

	layerId ppuLayer

	OPTMap  *Background
	optFunc optResolver

	enabledOnMainScreen, enabledOnSubScreen bool

	renderedDotCache renderedDotCache
}

func NewBackground(ds tileDataSource, epochPtr *uint64, layer ppuLayer) *Background {
	bg := &Background{
		ds:           ds,
		currentEpoch: epochPtr,
		scrollEpoch:  1,
		layerId:      layer,
	}

	for i := range bg.tileMap {
		bg.tileMap[i].bg = bg
	}

	for i := range len(bg.charTiles) {
		bg.charTiles[i].layerEpoch = bg
		bg.charTiles[i].ds = bg.ds
		bg.charTiles[i].isValid = false
	}

	bg.renderedDotCache.H = 0xFFFF

	return bg
}

func (bg *Background) InvalidateScrollCache() {
	bg.scrollEpoch++
}

func (bg *Background) isActive() bool {
	return bg.enabledOnMainScreen || bg.enabledOnSubScreen
}

func (bg *Background) GetLayerSourceEpoch() *uint64 {
	return bg.currentEpoch
}

func (bg *Background) Invalidate(addr uint16) {
	if bg.tileMapAddress <= addr && bg.tileMapAddress+tileMapDimensionsLUT[bg.tileMapSize].wordSize > addr {
		index := addr - bg.tileMapAddress
		if index < uint16(len(bg.tileMap)) {
			bg.tileMap[index].isValid = false
			//fmt.Println("invlidation")
		}
		return
	}

	if addr >= bg.charTileAddressBase {
		wordsPerTile := uint16(bg.colorDepth) << 2
		tileIndex := (addr - bg.charTileAddressBase) / wordsPerTile
		if tileIndex < uint16(len(bg.charTiles)) {
			bg.charTiles[tileIndex].isValid = false
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
	if charDimensions.modMask == 7 {
		return px, row, 0, (rowCnt&31)<<5 + columnCnt&31
	}
	charMapID := (row>>3)<<1 + (px >> 3)
	row &= 7
	px &= 7
	tileIndex := tileMapID*0x400 + (rowCnt&31)<<5 + columnCnt&31

	return px, row, charMapID, tileIndex
}

// TODO this can be optimized like crazy
// save the char reference in the tile
// save the char address in the chartile
// basically free pixels
// the previously read tile can also be cached so its only 1 tile lookup instead of 64 per tile
func (bg *Background) GetDotAt(H, V uint16) (uint16, byte, bool) {
	if bg.renderedDotCache.H == H {
		ret := bg.renderedDotCache
		return ret.color, ret.priority, true
	}
	cache := &bg.tileMapLookupCacke[H][V]
	if bg.scrollEpoch != cache.entryEpoch {
		//TODO add a nested for loop that set up all 8x8 dots of the tile with this data
		//so this is only calculated once every character tile which i think is fast
		hScroll, vScroll := H+bg.hScroll, V+bg.vScroll
		cache.px, cache.row, cache.charMapID, cache.tileIndex = getTileIndexAndPixelCoordinates(bg.tileMapSize, bg.charTileSize, hScroll, vScroll)
		cache.entryEpoch = bg.scrollEpoch
	}

	charMapID := cache.charMapID
	px := cache.px
	row := cache.row
	tileIndex := cache.tileIndex

	if bg.OPTMap != nil {
		if tileColumn := (H + uint16(7-cache.px)) >> 3; tileColumn > 0 {
			//I THINK this is the correct logic tho i cant verify yet so it is what it is.
			hScroll, vScroll := bg.optFunc(bg, H, V)
			px, row, charMapID, tileIndex = getTileIndexAndPixelCoordinates(bg.tileMapSize, bg.charTileSize, hScroll, vScroll)
		}
	}

	tile := &bg.tileMap[tileIndex]
	if currentEpoch := *bg.currentEpoch; !tile.isValid || tile.lastRenderEpoch != currentEpoch {
		tile.setup(tileIndex, currentEpoch)
	}

	px = tileFlipXLUT[tile.flipIndex][px]
	row = tileFlipYLUT[tile.flipIndex][row]

	charIndex := tile.charIndex
	if bg.charTileSize == 1 {
		charMapID = compositeFlipLUT[charMapID][tile.flipIndex]
		charIndex += charMapIdToOffsetLUT[charMapID]
	}

	charData := bg.charTiles[charIndex].getPixelAt(bg.colorDepth, tile.GetVramTileWordIndex, charMapID, px, row)

	bg.renderedDotCache.priority = tile.priority
	bg.renderedDotCache.H = H

	if bg.colorDepth == bpp8 && bg.isDirectColor {
		if charData == 0 {
			bg.renderedDotCache.color = BG_BACKDROP_COLOR
			return BG_BACKDROP_COLOR, tile.priority, true
		} else {
			red := (charData&7)<<2 | ((tile.paletteNum & 1) << 1)
			green := (charData&0x1C)>>1 | (tile.paletteNum & 2)
			blue := (charData&0xC0)>>3 | (tile.paletteNum & 4)
			ret := uint16(blue)<<10 | uint16(green)<<5 | uint16(red)
			bg.renderedDotCache.color = ret
			return ret, tile.priority, true
		}
	}
	ret := bg.ds.getCGRAM()[charData+bg.getPaletteIndex(bg.layerId, bg.colorDepth, tile.paletteNum)]
	bg.renderedDotCache.color = ret
	return ret, tile.priority, true
}

// my best guess for OPT. will test it in a year when i can run games LUL
// TODO once i know this works it should be heavily optimized
func resolveOPTMode26(bg *Background, H, V uint16) (uint16, uint16) {
	HOFS := bg.hScroll + H
	VOFS := bg.vScroll + V

	layer := bg.layerId
	if layer != bg1 && layer != bg2 {
		return HOFS, VOFS
	}

	hLookup := HOFS&7 | (((H - 8) & 0xFFF8) + (bg.OPTMap.hScroll & 0xFFF8))
	vLookup := bg.OPTMap.vScroll

	_, _, _, hTileIndex := getTileIndexAndPixelCoordinates(
		bg.OPTMap.tileMapSize, bg.OPTMap.charTileSize, hLookup, vLookup)
	_, _, _, vTileIndex := getTileIndexAndPixelCoordinates(
		bg.OPTMap.tileMapSize, bg.OPTMap.charTileSize, hLookup, vLookup+8)

	hScrollData := bg.ds.getVRAM()[bg.OPTMap.tileMapAddress+hTileIndex]
	vScrollData := bg.ds.getVRAM()[bg.OPTMap.tileMapAddress+vTileIndex]

	checkBit := uint16(0x2000) // BG1
	if layer == bg2 {
		checkBit = 0x4000 // BG2
	}

	if hScrollData&checkBit != 0 {
		HOFS = (HOFS & 7) | (H & 0xFFF8) + (hScrollData & 0x3F8) // 0000001111111000
	}

	if vScrollData&checkBit != 0 {
		VOFS = vScrollData&0x3FF + V
	}

	return HOFS, VOFS
}

func resolveOPTMode4(bg *Background, H, V uint16) (uint16, uint16) {
	HOFS := bg.hScroll + H
	VOFS := bg.vScroll + V

	layer := bg.layerId
	if layer != bg1 && layer != bg2 {
		return HOFS, VOFS
	}

	hLookup := HOFS&7 | (((H - 8) & 0xFFF8) + (bg.OPTMap.hScroll & 0xFFF8))
	vLookup := bg.OPTMap.vScroll

	_, _, _, tileIndex := getTileIndexAndPixelCoordinates(
		bg.OPTMap.tileMapSize, bg.OPTMap.charTileSize, hLookup, vLookup)

	scrollData := bg.ds.getVRAM()[bg.OPTMap.tileMapAddress+tileIndex]

	var hScrollData, vScrollData uint16

	if scrollData&0x8000 != 0 {
		hScrollData = 0
		vScrollData = scrollData
	} else {
		hScrollData = scrollData
		vScrollData = 0
	}

	checkBit := uint16(0x2000) // BG1
	if layer == bg2 {
		checkBit = 0x4000 // BG2
	}

	if hScrollData&checkBit != 0 {
		HOFS = (HOFS & 7) | (H & 0xFFF8) + (hScrollData & 0x3F8) // 0000001111111000
	}

	if vScrollData&checkBit != 0 {
		VOFS = vScrollData&0x3FF + V
	}

	return HOFS, VOFS
}

type BgTile struct {
	isValid                      bool
	verticalFlip, horizontalFlip bool
	flipIndex                    byte

	priority   byte
	paletteNum byte
	charIndex  uint16

	lastRenderEpoch uint64
	bg              *Background
}

func (bt *BgTile) setup(tileIndex uint16, currentEpoch uint64) {
	params := bt.bg.ds.getVRAM()[bt.bg.tileMapAddress+tileIndex]
	bt.flipIndex = byte((params >> 14) & 3)
	bt.priority = byte(params>>13) & 1
	bt.paletteNum = byte(params>>10) & 7
	bt.charIndex = params & 0x3FF

	bt.isValid = true
	bt.lastRenderEpoch = currentEpoch
}

func (tile *BgTile) GetVramTileWordIndex(tileIndex byte) uint16 {
	return ((tile.charIndex+charMapIdToOffsetLUT[tileIndex])*uint16(tile.bg.colorDepth<<2) + tile.bg.charTileAddressBase) & 0x7FFF
}

// TODO chartile needs to be able to handle 16x16 tiles later on too
type CharTile struct {
	isValid bool

	renderer     bitPlaneRenderer
	resolvedData [8][8]byte

	tileAddress uint16
	ds          tileDataSource

	lastRenderEpoch uint64
	layerEpoch      LayerEpochSource
}

func (ct *CharTile) setup(bitPlanes colorDepth) {
	switch bitPlanes {
	case 2:
		ct.renderer = RenderTile2bppLUT
	case 4:
		ct.renderer = RenderTile4bppLUT
	case 8:
		ct.renderer = RenderTile8bppLUT
	default:
		//WHY WOULD A ROM NOT CALL BGMODE
		ct.renderer = RenderTile2bppLUT
	}
}

func (ct *CharTile) getPixelAt(bitplanes colorDepth, addr VRAMAddressCalculator, tileId, px, row byte) byte {
	currentEpoch := *ct.layerEpoch.GetLayerSourceEpoch()
	if ct.lastRenderEpoch != currentEpoch {
		ct.tileAddress = addr(tileId)
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

// FIXME this one doesnt work
func RenderTile4bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row*2)]
		w2 := VRAM[wordBase+uint16(row*2)+1]
		for px := range 8 {
			out[row][px] = resolveWordBitPlanePixel(w1, px) | (resolveWordBitPlanePixel(w2, px) << 2)
		}
	}
}

// EVERYTHING IS BASED ON 2bpp. in memory 8bpp is just 2bpp 2bpp 2bpp 2bpp
func RenderTile8bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := 0; row < 8; row++ {
		w01 := VRAM[wordBase+uint16(row)]    // bitplanes 0-1
		w23 := VRAM[wordBase+uint16(row)+8]  // bitplanes 2-3
		w45 := VRAM[wordBase+uint16(row)+16] // bitplanes 4-5
		w67 := VRAM[wordBase+uint16(row)+24] // bitplanes 6-7

		p0 := byte(w01)
		p1 := byte(w01 >> 8)
		p2 := byte(w23)
		p3 := byte(w23 >> 8)
		p4 := byte(w45)
		p5 := byte(w45 >> 8)
		p6 := byte(w67)
		p7 := byte(w67 >> 8)
		for px := 0; px < 8; px++ {
			mask := byte(1 << (7 - px))
			idx := (p0&mask)>>(7-px)<<0 |
				(p1&mask)>>(7-px)<<1 |
				(p2&mask)>>(7-px)<<2 |
				(p3&mask)>>(7-px)<<3 |
				(p4&mask)>>(7-px)<<4 |
				(p5&mask)>>(7-px)<<5 |
				(p6&mask)>>(7-px)<<6 |
				(p7&mask)>>(7-px)<<7
			out[row][px] = idx
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
	for row := range uint16(8) {
		w01 := VRAM[wordBase+row]   // bitplanes 0-1
		w23 := VRAM[wordBase+row+8] // bitplanes 2-3

		p0 := byte(w01)
		p1 := byte(w01 >> 8)
		p2 := byte(w23)
		p3 := byte(w23 >> 8)

		for px := range 8 {
			out[row][px] = bitplaneLUT[p0][px] |
				bitplaneLUT[p1][px]<<1 |
				bitplaneLUT[p2][px]<<2 |
				bitplaneLUT[p3][px]<<3
		}
	}
}

func RenderTile8bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range uint16(8) {
		w01 := VRAM[wordBase+row]    // bitplanes 0-1
		w23 := VRAM[wordBase+row+8]  // bitplanes 2-3
		w45 := VRAM[wordBase+row+16] // bitplanes 4-5
		w67 := VRAM[wordBase+row+24] // bitplanes 6-7

		p0 := byte(w01)
		p1 := byte(w01 >> 8)
		p2 := byte(w23)
		p3 := byte(w23 >> 8)
		p4 := byte(w45)
		p5 := byte(w45 >> 8)
		p6 := byte(w67)
		p7 := byte(w67 >> 8)

		for px := range 8 {
			out[row][px] = bitplaneLUT[p0][px] |
				bitplaneLUT[p1][px]<<1 |
				bitplaneLUT[p2][px]<<2 |
				bitplaneLUT[p3][px]<<3 |
				bitplaneLUT[p4][px]<<4 |
				bitplaneLUT[p5][px]<<5 |
				bitplaneLUT[p6][px]<<6 |
				bitplaneLUT[p7][px]<<7
		}
	}
}
