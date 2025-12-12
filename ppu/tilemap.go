package ppu

import (
	"math/bits"
)

type colorDepth uint16
type ppuLayer uint16

const BG_BACKDROP_COLOR = -1

const (
	bg1      ppuLayer = 0
	bg2      ppuLayer = 1
	bg3      ppuLayer = 2
	bg4      ppuLayer = 3
	obj      ppuLayer = 4
	backdrop ppuLayer = 5
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
type rendererFunction func(uint16, uint16) (int, byte, bool)

type LayerEpochSource interface {
	GetLayerSourceEpoch() uint64
}

type renderedDotCache struct {
	color    int
	priority byte
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
	paletteIndexMask    byte //used for quick transparency check
	getPaletteIndex     colorIndex
	isDirectColor       bool

	vScroll uint16
	hScroll uint16

	currentEpoch uint64

	layerId ppuLayer

	OPTMap  *Background
	optFunc optResolver

	enabledOnMainScreen, enabledOnSubScreen bool

	renderCacheEnd uint16
	renderCache    [SCREEN_WIDTH << 1]renderedDotCache

	mosaic bool
}

func NewBackground(ds tileDataSource, layer ppuLayer) *Background {
	bg := &Background{
		ds:      ds,
		layerId: layer,
	}

	for i := range bg.tileMap {
		bg.tileMap[i].bg = bg
	}

	for i := range len(bg.charTiles) {
		bg.charTiles[i].layerEpoch = bg
		bg.charTiles[i].ds = bg.ds
		bg.charTiles[i].isValid = false
	}

	return bg
}

func (bg *Background) isActive() bool {
	return bg.enabledOnMainScreen || bg.enabledOnSubScreen
}

func (bg *Background) setBgColorDepth(colorDepth colorDepth) {
	bg.colorDepth = colorDepth
	bg.paletteIndexMask = (1 << bg.colorDepth) - 1 //uint16 to uint8 ??
}

func (bg *Background) GetLayerSourceEpoch() uint64 {
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
		k := bits.TrailingZeros(uint(bg.colorDepth)) // V=2 (0010) k=1, V=4 (0100) k=2, V=8 (1000), k=3
		tileIndex := (addr - bg.charTileAddressBase) >> (k + 2)
		if tileIndex < uint16(len(bg.charTiles)) {
			bg.charTiles[tileIndex].isValid = false
			//fmt.Println("invlidation")
		}
	}
}

func getTileIndexAndPixelCoordinates(tileMapSize uint16, charTileSize byte, H, V uint16) (byte, byte, byte, uint16) {
	var px byte
	var tileIndex uint16
	tileDimensions := tileMapDimensionsLUT[tileMapSize]
	charDimensions := charTileSizeLUT[charTileSize]
	rowCnt := (V >> charDimensions.divMask) & tileDimensions.modMaskH
	row := byte(V & charDimensions.modMask)
	if hires == 1 {
		columnCnt := (H >> 4) & tileDimensions.modMaskW
		tileMapID := (rowCnt>>5)<<tileDimensions.mapsPerRowMinusOne + columnCnt>>5
		tileIndex = tileMapID<<10 + (rowCnt&31)<<5 + columnCnt&31
		px = byte(H & 15)
	} else {
		columnCnt := (H >> charDimensions.divMask) & tileDimensions.modMaskW
		tileMapID := (rowCnt>>5)<<tileDimensions.mapsPerRowMinusOne + columnCnt>>5
		tileIndex = tileMapID<<10 + (rowCnt&31)<<5 + columnCnt&31
		px = byte(H & charDimensions.modMask)
		if charTileSize == 0 {
			return px, row, 0, tileIndex
		}
	}
	charMapID := (row>>3)<<1 + (px >> 3)
	row &= 7
	px &= 7

	return px, row, charMapID, tileIndex
}

// TODO this can be optimized like crazy
// save the char address in the chartile
// basically free pixels
// the previously read tile can also be cached so its only 1 tile lookup instead of 64 per tile
// TODO isSubscreen isnt used by anything
func (bg *Background) GetDotAt(H, V uint16) (int, byte, bool) {
	if H < bg.renderCacheEnd {
		ret := bg.renderCache[H]
		return ret.color, ret.priority, true
	}

	hScroll, vScroll := H+bg.hScroll, V+bg.vScroll+(1<<interlace)
	if bg.OPTMap != nil && (H+(7-(hScroll&7)))>>3 > 0 {
		hScroll, vScroll = bg.optFunc(bg, H, V+(1<<interlace))
	}
	px, row, charMapID, tileIndex := getTileIndexAndPixelCoordinates(bg.tileMapSize, bg.charTileSize, hScroll, vScroll)

	tile := &bg.tileMap[tileIndex]
	if currentEpoch := bg.currentEpoch; tile.lastRenderEpoch != currentEpoch || !tile.isValid {
		tile.setup(tileIndex, currentEpoch)
	}

	row = tileFlipYLUT[tile.flipIndex][row]
	if bg.mosaic {
		offset := 8 - px
		size := (offset / mosaicSize) * mosaicSize
		if (offset % mosaicSize) > 0 {
			size += mosaicSize
		}
		bg.renderCacheEnd = min(H+uint16(size), SCREEN_WIDTH<<hires)
	} else {
		bg.renderCacheEnd = min(H+uint16(8-px), SCREEN_WIDTH<<hires)
	}

	if tile.flipIndex > 0 {
		if bg.charTileSize == 1 {
			charMapID = compositeFlipLUT[charMapID][tile.flipIndex]
		} else if hires == 1 {
			charMapID = compositeFlip16x8LUT[charMapID][tile.flipIndex]
		}
	}

	var ret, color int
	charTile := tile.charTiles[charMapID]
	flipXTtable := &tileFlipXLUT[tile.flipIndex]
	rowData := charTile.getRowAt(bg.colorDepth, tile.GetVramTileWordIndex, charMapID, row)
	cgram := bg.ds.getCGRAM()

	for i := H; i < bg.renderCacheEnd; i++ {
		if !bg.mosaic || (bg.mosaic && (i-H)%(uint16(mosaicSize)) == 0) {
			charData := rowData[flipXTtable[px]]

			if bg.colorDepth == bpp8 && bg.isDirectColor {
				if charData == 0 {
					color = BG_BACKDROP_COLOR
				} else {
					red := ((charData & 0x07) << 2) | ((tile.paletteNum & 0x01) << 1)
					green := ((charData & 0x38) >> 1) | (tile.paletteNum & 0x02)
					blue := ((charData & 0xC0) >> 3) | (tile.paletteNum & 0x04)
					color = int(uint16(blue)<<10 | uint16(green)<<5 | uint16(red))
				}
			} else {
				pIndex := bg.getPaletteIndex(bg.layerId, bg.colorDepth, tile.paletteNum) + charData
				if pIndex&bg.paletteIndexMask == 0 {
					color = BG_BACKDROP_COLOR
				} else {
					color = int(cgram[pIndex])
				}
			}
			if i == H {
				ret = color
			}
		}

		cache := &bg.renderCache[i]
		cache.priority = tile.priority
		cache.color = color
		px++
	}
	return ret, tile.priority, true
}

func resolveOPTMode26(bg *Background, H, V uint16) (uint16, uint16) {
	HOFS := bg.hScroll + H
	VOFS := bg.vScroll + V

	layer := bg.layerId
	if layer != bg1 && layer != bg2 {
		return HOFS, VOFS
	}

	optM := bg.OPTMap
	vram := bg.ds.getVRAM()

	//hLookup := ((H - 8) + optM.hScroll) & 0xFFF8
	hLookup := HOFS&7 | (((H - 8) & 0xFFF8) + (optM.hScroll & 0xFFF8))
	vLookup := optM.vScroll

	_, _, _, hTileIndex := getTileIndexAndPixelCoordinates(
		optM.tileMapSize, optM.charTileSize, hLookup, vLookup)
	_, _, _, vTileIndex := getTileIndexAndPixelCoordinates(
		optM.tileMapSize, optM.charTileSize, hLookup, vLookup+8)

	hScrollData := vram[optM.tileMapAddress+hTileIndex]
	vScrollData := vram[optM.tileMapAddress+vTileIndex]

	checkBit := uint16(1 << (13 + layer))

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

	optM := bg.OPTMap

	//hLookup := ((H - 8) + optM.hScroll) & 0xFFF8
	hLookup := HOFS&7 | (((H - 8) & 0xFFF8) + (optM.hScroll & 0xFFF8))
	vLookup := optM.vScroll

	_, _, _, tileIndex := getTileIndexAndPixelCoordinates(
		optM.tileMapSize, optM.charTileSize, hLookup, vLookup)

	scrollData := bg.ds.getVRAM()[optM.tileMapAddress+tileIndex]

	checkBit := uint16(1 << (13 + layer))
	if scrollData&checkBit != 0 {
		if scrollData&0x8000 != 0 {
			VOFS = V + scrollData
		} else {
			HOFS = (HOFS & 7) | (H & 0xFFF8) + (scrollData & 0x3F8) // 0000001111111000
		}
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
	charTiles  [4]*CharTile

	lastRenderEpoch uint64
	bg              *Background
}

func (bt *BgTile) setup(tileIndex uint16, currentEpoch uint64) {
	params := bt.bg.ds.getVRAM()[(bt.bg.tileMapAddress+tileIndex)&0x7FFF]
	bt.flipIndex = byte((params >> 14) & 3)
	bt.priority = byte(params>>13) & 1
	bt.paletteNum = byte(params>>10) & 7
	charIndex := params & 0x3FF
	bt.charIndex = charIndex

	charTiles := &bt.bg.charTiles

	bt.charTiles[0] = &charTiles[bt.charIndex]
	if bt.bg.charTileSize == 1 {
		bt.charTiles[1] = &charTiles[charIndex+charMapIdToOffsetLUT[1]]
		bt.charTiles[2] = &charTiles[charIndex+charMapIdToOffsetLUT[2]]
		bt.charTiles[3] = &charTiles[charIndex+charMapIdToOffsetLUT[3]]
	} else if hires == 1 {
		bt.charTiles[1] = &charTiles[charIndex+charMapIdToOffsetLUT[1]]
	}

	bt.isValid = true
	bt.lastRenderEpoch = currentEpoch
}

func (tile *BgTile) GetVramTileWordIndex(tileIndex byte) uint16 {
	k := bits.TrailingZeros(uint(tile.bg.colorDepth << 2))
	return ((tile.charIndex+charMapIdToOffsetLUT[tileIndex])<<k + tile.bg.charTileAddressBase) & 0x7FFF
}

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
	currentEpoch := ct.layerEpoch.GetLayerSourceEpoch()
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

func (ct *CharTile) getRowAt(bitplanes colorDepth, addr VRAMAddressCalculator, tileId, row byte) *[8]byte {
	currentEpoch := ct.layerEpoch.GetLayerSourceEpoch()
	if ct.lastRenderEpoch != currentEpoch {
		ct.tileAddress = addr(tileId)
		goto RENDER_AND_CACHE
	}

	if !ct.isValid {
		goto RENDER_AND_CACHE
	}

	return &ct.resolvedData[row]

RENDER_AND_CACHE:
	ct.setup(bitplanes)
	ct.renderer(ct.ds.getVRAM(), ct.tileAddress, &ct.resolvedData)
	ct.isValid = true
	ct.lastRenderEpoch = currentEpoch

	return &ct.resolvedData[row]
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
