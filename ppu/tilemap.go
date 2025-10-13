package ppu

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
	tileMap        [0x1000]BgTile //4x400
	tileMapAddress uint16
	tileMapSize    uint16

	charTiles           map[uint16]*CharTile
	charTileAddressBase uint16
	charTileSize        byte
	colorDepth          byte
}

func NewBackground1() *Background1 {
	return &Background1{
		charTiles: make(map[uint16]*CharTile),
	}
}

func (bg1 *Background1) Invalidate(addr uint16) {
	if bg1.tileMapAddress <= addr && bg1.tileMapAddress+bg1.getTileMapWordCount() > addr {
		index := addr - bg1.tileMapAddress
		if index < uint16(len(bg1.tileMap)) {
			bg1.tileMap[index].isValid = false
		}
		return
	}

	if addr >= bg1.charTileAddressBase {
		wordsPerTile := uint16(bg1.colorDepth) * 4
		tileIndex := (addr - bg1.charTileAddressBase) / wordsPerTile
		if t, ok := bg1.charTiles[bg1.charTileAddressBase+(tileIndex*wordsPerTile)]; ok {
			t.isValid = false
		}
	}
}

func (bg1 *Background1) getTileMapWordCount() uint16 {
	switch bg1.tileMapSize {
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

func (bg1 *Background1) getDotAt(VRAM []uint16, CGRAM []uint16, H, V byte) uint16 {
	rowCnt := V / 8
	row := V % 8
	columnCnt := H / 8
	px := H % 8

	tile := bg1.tileMap[rowCnt*32+columnCnt]
	char := bg1.charTiles[tile.tileIndex*uint16(bg1.colorDepth*4)+bg1.charTileAddressBase]
	return CGRAM[char.getPixelAt(VRAM, px, row)+tile.paletteNum]
}

type BgTile struct {
	isValid                      bool
	verticalFlip, horizontalFlip bool

	priority   byte
	paletteNum byte
	tileIndex  uint16
}

// TODO chartile needs to be able to handle 16x16 tiles later on too
type CharTile struct {
	isValid bool

	renderer     bitPlaneRenderer
	resolvedData [8][8]byte

	tileIndex uint16
}

func (ct *CharTile) getPixelAt(VRAM []uint16, px, row byte) byte {
	if !ct.isValid {
		ct.renderer(VRAM, ct.tileIndex, &ct.resolvedData)
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
		w1 := VRAM[wordBase+uint16(row*2)]
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

// EVEN FASTER: Pre-compute lookup tables (trade memory for speed)
var bitplaneLUT [256][8]byte

func initBitplaneLUT() {
	// Pre-compute all possible byte -> 8 pixels mappings
	for b := range 256 {
		for px := range 8 {
			bitplaneLUT[b][px] = byte((b >> (7 - px)) & 1)
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
