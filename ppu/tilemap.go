package ppu

import (
	"fmt"
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

func (cd *colorDepth) vramSizeShift() int {
	// V=2 (0010) k=1, V=4 (0100) k=2, V=8 (1000), k=3
	switch *cd {
	case 2:
		return 3
	case 4:
		return 4
	case 8:
		return 5
	default:
		return bits.TrailingZeros(uint(*cd)) + 2
	}
}

func (cd *colorDepth) transparencyMask() byte {
	switch *cd {
	case 2:
		return 0x3
	case 4:
		return 0xF
	case 8:
		return 0xFF
	default:
		return (1 << *cd) - 1 //uint16 to uint8 ??
	}
}

func (cd *colorDepth) getRenderer() bitPlaneRenderer {
	switch *cd {
	case 2:
		return RenderTile2bppLUT
	case 4:
		return RenderTile4bppLUT
	case 8:
		return RenderTile8bppLUT
	default:
		panic(fmt.Errorf("PPU: Could not set up assign a bitplane renderer."))
	}
}

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
	VRAM  []uint16
	CGRAM []uint16

	tileMap        [0x1000]BgTile //4x400
	tileMapAddress uint16
	tileMapSize    uint16

	charTiles           [0x400]CharTile //each bg can reference 0x400 charTiles at most. wraps
	charTileAddressBase uint16
	charTileSize        byte
	colorDepth          colorDepth
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
		VRAM:    ds.getVRAM(),
		CGRAM:   ds.getCGRAM(),
		layerId: layer,
	}

	for i := range bg.tileMap {
		bg.tileMap[i].bg = bg
	}

	for i := range len(bg.charTiles) {
		bg.charTiles[i].layerEpoch = bg
		bg.charTiles[i].VRAM = ds.getVRAM()
		bg.charTiles[i].isValid = false
	}

	return bg
}

func (bg *Background) isActive() bool {
	return bg.enabledOnMainScreen || bg.enabledOnSubScreen
}

func (bg *Background) GetLayerSourceEpoch() uint64 {
	return bg.currentEpoch
}

func (bg *Background) Invalidate(addr uint16) {
	if bg.tileMapAddress <= addr && bg.tileMapAddress+tileMapDimensionsLUT[bg.tileMapSize].wordSize > addr {
		index := addr - bg.tileMapAddress
		if index < uint16(len(bg.tileMap)) {
			bg.tileMap[index].isValid = false
		}
		return
	}

	if addr >= bg.charTileAddressBase {
		tileIndex := (addr - bg.charTileAddressBase) >> bg.colorDepth.vramSizeShift()
		if tileIndex < uint16(len(bg.charTiles)) {
			bg.charTiles[tileIndex].isValid = false
		}
	}
}

func getTileIndexAndPixelCoordinates(tileMapSize uint16, charTileSize byte, H, V uint16) (byte, byte, byte, uint16) {
	charTileSize = hires<<1 | charTileSize
	tileDimensions := tileMapDimensionsLUT[tileMapSize]
	charDimensions := charTileSizeLUT[charTileSize]
	row := byte(V & charDimensions.modMaskH)
	px := byte(H & charDimensions.modMaskW)
	columnCnt := (H >> charDimensions.divMaskW) & tileDimensions.modMaskW
	rowCnt := (V >> charDimensions.divMaskH) & tileDimensions.modMaskH
	tileMapID := (rowCnt>>5)<<tileDimensions.mapsPerRowMinusOne + columnCnt>>5
	tileIndex := tileMapID<<10 | (rowCnt&31)<<5 | columnCnt&31
	if charTileSize == 0 {
		return px, row, 0, tileIndex
	}
	charMapID := (row>>3)<<1 | (px >> 3)
	row &= 7
	px &= 7

	return px, row, charMapID, tileIndex
}

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

	if bg.charTileSize|hires == 1 {
		//figured out mapId ^ flipIndex accurately calculates flip without any lut. took me a while
		charMapID ^= tile.flipIndex & (3 >> (hires & (bg.charTileSize ^ 1)))
	}

	var ret, color int
	charTile := tile.charTiles[charMapID]
	row ^= tile.verticalFlipMask
	rowData := charTile.getRowAt(bg.colorDepth, tile.GetVramTileWordIndex, charMapID, row)
	cgram := bg.CGRAM
	transparencyMask := bg.colorDepth.transparencyMask()

	for i := H; i < bg.renderCacheEnd; i++ {
		if !bg.mosaic || (bg.mosaic && (i-H)%(uint16(mosaicSize)) == 0) {
			charData := rowData[px^tile.horizontalFlipMask]

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
				paletteIdx := bg.getPaletteIndex(bg.layerId, bg.colorDepth, tile.paletteNum) + charData
				if paletteIdx&transparencyMask == 0 {
					color = BG_BACKDROP_COLOR
				} else {
					color = int(cgram[paletteIdx])
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
	vram := bg.VRAM

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

	scrollData := bg.VRAM[optM.tileMapAddress+tileIndex]

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
	isValid                              bool
	verticalFlipMask, horizontalFlipMask byte
	flipIndex                            byte

	priority   byte
	paletteNum byte
	charIndex  uint16
	charTiles  [4]*CharTile

	lastRenderEpoch uint64
	bg              *Background
}

func (bt *BgTile) setup(tileIndex uint16, currentEpoch uint64) {
	params := bt.bg.VRAM[(bt.bg.tileMapAddress+tileIndex)&0x7FFF]
	bt.flipIndex = byte((params >> 14) & 3)
	bt.horizontalFlipMask = -(bt.flipIndex & 1) & 7 //0 or 7
	bt.verticalFlipMask = -(bt.flipIndex >> 1) & 7  //0 or 7
	bt.priority = byte(params>>13) & 1
	bt.paletteNum = byte(params>>10) & 7
	charIndex := params & 0x3FF
	bt.charIndex = charIndex

	charTiles := &bt.bg.charTiles

	bt.charTiles[0] = &charTiles[charIndex]
	if bt.bg.charTileSize == 1 {
		bt.charTiles[1] = &charTiles[(charIndex+0x01)&0x3FF]
		bt.charTiles[2] = &charTiles[(charIndex+0x10)&0x3FF]
		bt.charTiles[3] = &charTiles[(charIndex+0x11)&0x3FF]
	} else if hires == 1 {
		bt.charTiles[1] = &charTiles[(charIndex+0x01)&0x3FF]
	}

	bt.isValid = true
	bt.lastRenderEpoch = currentEpoch
}

func (tile *BgTile) GetVramTileWordIndex(tileIndex byte) uint16 {
	k := tile.bg.colorDepth.vramSizeShift()
	/*
		if tileIndex == 0 {
			return (tile.charIndex<<k + tile.bg.charTileAddressBase) & 0x7FFF
		}
	*/
	offset := uint16((tileIndex&2)<<3 | tileIndex&1) // 0, 1, 0x10, 0x11
	return (((tile.charIndex+offset)&0x3FF)<<k + tile.bg.charTileAddressBase) & 0x7FFF
}

type CharTile struct {
	isValid bool

	renderer     bitPlaneRenderer
	resolvedData [8][8]byte

	tileAddress uint16
	VRAM        []uint16

	lastRenderEpoch uint64
	layerEpoch      LayerEpochSource
}

func (ct *CharTile) getPixelAt(bitplanes colorDepth, addr VRAMAddressCalculator, tileId, px, row byte) byte {
	currentEpoch := ct.layerEpoch.GetLayerSourceEpoch()
	if ct.lastRenderEpoch != currentEpoch {
		ct.tileAddress = addr(tileId)
		ct.renderer = bitplanes.getRenderer()
		goto RENDER_AND_CACHE
	}

	if !ct.isValid {
		goto RENDER_AND_CACHE
	}

	return ct.resolvedData[row][px]

RENDER_AND_CACHE:
	ct.renderer(ct.VRAM, ct.tileAddress, &ct.resolvedData)
	ct.isValid = true
	ct.lastRenderEpoch = currentEpoch

	return ct.resolvedData[row][px]
}

func (ct *CharTile) getRowAt(bitplanes colorDepth, addr VRAMAddressCalculator, tileId, row byte) *[8]byte {
	currentEpoch := ct.layerEpoch.GetLayerSourceEpoch()
	if ct.lastRenderEpoch != currentEpoch {
		ct.tileAddress = addr(tileId)
		ct.renderer = bitplanes.getRenderer()
		goto RENDER_AND_CACHE
	}

	if !ct.isValid {
		goto RENDER_AND_CACHE
	}

	return &ct.resolvedData[row]

RENDER_AND_CACHE:
	ct.renderer(ct.VRAM, ct.tileAddress, &ct.resolvedData)
	ct.isValid = true
	ct.lastRenderEpoch = currentEpoch

	return &ct.resolvedData[row]
}

func RenderTile2bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w01 := VRAM[wordBase+uint16(row)]
		for px := range 8 {
			out[row][px] = byte(((w01 >> (7 - px)) & 1) | (((w01 >> (15 - px)) & 1) << 1))
		}
	}
}

func RenderTile4bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		currentRow := wordBase + uint16(row)
		w01 := VRAM[currentRow]   // bitplanes 0-1
		w23 := VRAM[currentRow+8] // bitplanes 2-3

		for px := range uint16(8) {
			offset8 := 7 - px
			offset16 := 15 - px
			out[row][px] = byte((w01>>offset8)&1<<0 |
				(w01>>offset16)&1<<1 |
				(w23>>offset8)&1<<2 |
				(w23>>offset16)&1<<3)
		}
	}
}

// EVERYTHING IS BASED ON 2bpp. in memory 8bpp is just 2bpp 2bpp 2bpp 2bpp
func RenderTile8bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		currentRow := wordBase + uint16(row)
		w01 := VRAM[currentRow]    // bitplanes 0-1
		w23 := VRAM[currentRow+8]  // bitplanes 2-3
		w45 := VRAM[currentRow+16] // bitplanes 4-5
		w67 := VRAM[currentRow+24] // bitplanes 6-7

		for px := range uint16(8) {
			offset8 := 7 - px
			offset16 := 15 - px
			out[row][px] = byte((w01>>offset8)&1<<0 |
				(w01>>offset16)&1<<1 |
				(w23>>offset8)&1<<2 |
				(w23>>offset16)&1<<3 |
				(w45>>offset8)&1<<4 |
				(w45>>offset16)&1<<5 |
				(w67>>offset8)&1<<6 |
				(w67>>offset16)&1<<7)
		}
	}
}

func RenderTile2bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		word := VRAM[wordBase+uint16(row)]

		lowBits := bitplaneLUT[byte(word)]
		highBits := bitplaneLUT[byte(word>>8)]

		for px := range 8 {
			out[row][px] = lowBits[px] | (highBits[px] << 1)
		}
	}
}

func RenderTile4bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range uint16(8) {
		currentRow := wordBase + row
		w01 := VRAM[currentRow]   // bitplanes 0-1
		w23 := VRAM[currentRow+8] // bitplanes 2-3

		p0 := bitplaneLUT[byte(w01)]
		p1 := bitplaneLUT[byte(w01>>8)]
		p2 := bitplaneLUT[byte(w23)]
		p3 := bitplaneLUT[byte(w23>>8)]

		for px := range 8 {
			out[row][px] = p0[px] |
				p1[px]<<1 |
				p2[px]<<2 |
				p3[px]<<3
		}
	}
}

func RenderTile8bppLUT(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range uint16(8) {
		currentRow := wordBase + row
		w01 := VRAM[currentRow]    // bitplanes 0-1
		w23 := VRAM[currentRow+8]  // bitplanes 2-3
		w45 := VRAM[currentRow+16] // bitplanes 4-5
		w67 := VRAM[currentRow+24] // bitplanes 6-7

		p0 := bitplaneLUT[byte(w01)]
		p1 := bitplaneLUT[byte(w01>>8)]
		p2 := bitplaneLUT[byte(w23)]
		p3 := bitplaneLUT[byte(w23>>8)]
		p4 := bitplaneLUT[byte(w45)]
		p5 := bitplaneLUT[byte(w45>>8)]
		p6 := bitplaneLUT[byte(w67)]
		p7 := bitplaneLUT[byte(w67>>8)]

		for px := range 8 {
			out[row][px] = p0[px] |
				p1[px]<<1 |
				p2[px]<<2 |
				p3[px]<<3 |
				p4[px]<<4 |
				p5[px]<<5 |
				p6[px]<<6 |
				p7[px]<<7
		}
	}
}
