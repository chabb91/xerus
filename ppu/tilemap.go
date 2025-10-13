package ppu

type Background interface {
	GetTileMapAddress() uint16
	GetTileMapSize()
	GetCharTileAddress() byte
	GetCharTileSize() byte
	GetColorDepth() byte
	IsOffsetPerTile() bool
}

type TileMap struct {
	background *Background
	tiles      [0x400]*BgTile
}

type BgTile struct {
	isValid                      bool
	verticalFlip, horizontalFlip bool

	priority   byte
	paletteNum byte
	tileIndex  uint16

	charTile *CharTile
}

type CharTile struct {
	isValid bool

	resolvedData [8][8]byte

	tileIndex  uint16
	colorDepth byte
}

func (ct *CharTile) resolve(VRAM []uint16) {

}

func resolveWordBitPlanePixel(word uint16, px int) byte {
	return byte(((word >> (7 - px)) & 1) | (((word >> (15 - px)) & 1) << 1))
}

func RenderTile2bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row*2)]
		for px := range 8 {
			bit0 := (w1 >> (7 - px)) & 1
			bit1 := (w1 >> (15 - px)) & 1
			color := byte(bit0 | (bit1 << 1))
			out[row][px] = color
		}
	}
}

func RenderTile4bpp(VRAM []uint16, wordBase uint16, out *[8][8]byte) {
	for row := range 8 {
		w1 := VRAM[wordBase+uint16(row*2)]
		w2 := VRAM[wordBase+uint16(row*2)+1]
		for px := range 8 {
			bit0 := (w1 >> (7 - px)) & 1
			bit1 := (w1 >> (15 - px)) & 1
			bit2 := (w2 >> (7 - px)) & 1
			bit3 := (w2 >> (15 - px)) & 1
			color := byte(bit0 | (bit1 << 1) | (bit2 << 2) | (bit3 << 3))
			out[row][px] = color
		}
	}
}
