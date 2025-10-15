package ppu

type colorDepth uint16

const (
	bpp2 colorDepth = 2
	bpp4 colorDepth = 4
	bpp8 colorDepth = 8
)

type bitPlaneRenderer func([]uint16, uint16, *[8][8]byte)

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
}

func NewBackground1(ds tileDataSource) *Background1 {
	return &Background1{
		ds:        ds,
		charTiles: make(map[uint16]*CharTile),
	}
}

func (bg1 *Background1) Invalidate(addr uint16) {
	if bg1.tileMapAddress <= addr && bg1.tileMapAddress+getTileMapWordCount(bg1.tileMapSize) > addr {
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

func getTileMapWordCount(tileMapSize uint16) uint16 {
	switch tileMapSize {
	case 0:
		return 0x400 // 32x32
	case 1:
		return 0x800 // 64x32
	case 2:
		return 0x800 // 32x64
	case 3:
		return 0x1000 // 64x64
	default:
		return 0x400
	}
}

// TODO this can be optimized like crazy
// save the char reference in the tile
// save the char address in the chartile
// basically free pixels
// the previously read tile can also be cached so its only 1 tile lookup instead of 64 per tile
func (bg1 *Background1) GetDotAt(H, V byte) uint16 {
	rowCnt := uint16(V / 8)
	row := V % 8
	columnCnt := uint16(H / 8)
	px := H % 8

	tileIndex := rowCnt*32 + columnCnt
	tile := bg1.tileMap[tileIndex]
	if !tile.isValid {
		tile.setup(bg1.ds.getVRAM()[bg1.tileMapAddress+uint16(tileIndex)])
	}
	//TODO use lookuptables for this
	if tile.horizontalFlip {
		px = 7 - px
	}
	if tile.verticalFlip {
		row = 7 - row
	}

	charAddress := tile.tileIndex*uint16(bg1.colorDepth*4) + bg1.charTileAddressBase
	char := bg1.charTiles[charAddress]
	if char == nil {
		char = &CharTile{isValid: false, ds: bg1.ds, tileAddress: charAddress}
		bg1.charTiles[charAddress] = char
	}

	return bg1.ds.getCGRAM()[char.getPixelAt(bg1.colorDepth, px, row)+tile.paletteNum*(1<<bg1.colorDepth)]
}

type BgTile struct {
	isValid                      bool
	verticalFlip, horizontalFlip bool

	priority   byte
	paletteNum byte
	tileIndex  uint16
}

func (bt *BgTile) setup(params uint16) {
	bt.verticalFlip = (params>>15)&1 == 1
	bt.horizontalFlip = (params>>14)&1 == 1
	bt.priority = byte(params>>13) & 1
	bt.paletteNum = byte(params>>10) & 7
	bt.tileIndex = params & 0x3FF

	bt.isValid = true
}

// TODO chartile needs to be able to handle 16x16 tiles later on too
type CharTile struct {
	isValid bool

	renderer     bitPlaneRenderer
	resolvedData [8][8]byte

	tileAddress uint16
	ds          tileDataSource
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
	if !ct.isValid {
		ct.setup(bitplanes)
		ct.renderer(ct.ds.getVRAM(), ct.tileAddress, &ct.resolvedData)
		ct.isValid = true
	}
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
