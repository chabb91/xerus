package ppu

import "fmt"

type tileDataSource interface {
	getOAMLow() []byte
	getOAMHigh() []byte
	getVRAM() []uint16
	getCGRAM() []uint16
}

type tileValidator interface {
	tryInvalidate(addr uint16)
	//TODO
	//invalidateBgTiles(bg Background)
	//invalidateEverything()
}

type PPU struct {
	OAM   *OAMController
	VRAM  *VRAMController
	CGRAM *CGRAMController

	Bg1     *Background1
	BGxnOFS *BGxnOFS

	FBlank, VBlank, HBlank bool
	screenBrightness       byte

	H, V uint16

	bgEpochs [5]uint64 //1 2 3 4 and mode7

	InterruptScheduler InterruptScheduler
}

func NewPPU() *PPU {
	ppu := &PPU{
		OAM:     NewOAM(),
		CGRAM:   NewCGRAM(),
		BGxnOFS: &BGxnOFS{},
	}
	ppu.Bg1 = NewBackground1(ppu, &ppu.bgEpochs[0])
	ppu.VRAM = NewVRAM(ppu)
	return ppu
}

// Some of these registers can only be read and written to at specific times defined by the blanking periods
// TODO
func (ppu *PPU) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x2138:
		return ppu.OAM.ReadOAMData(), nil
	case 0x2139:
		return ppu.VRAM.ReadDataLow(), nil
	case 0x213A:
		return ppu.VRAM.ReadDataHigh(), nil
	case 0x213B:
		return ppu.CGRAM.ReadData(), nil
	default:
		return 0, fmt.Errorf("invalid PPU register read at $%04X", addr)
	}
}

func (ppu *PPU) Write(addr uint16, value byte) error {
	switch addr {
	case 0x2100:
		//TODO writing this register the first line of vlblank causes an oam address reset
		ppu.FBlank = (value>>7)&1 == 1
		ppu.screenBrightness = value & 0xF
		fmt.Println("INIDISP")
	case 0x2101:
		ppu.OAM.obsel.Setup(value)
	case 0x2102:
		ppu.OAM.SetAddWordLow(value)
	case 0x2103:
		ppu.OAM.SetAddWordHigh(value)
	case 0x2104:
		ppu.OAM.WriteOAMData(value)
	case 0x2105:
		fmt.Println("BGMODE: ", value)
		ppu.Bg1.charTileSize = (value >> 4) & 1
		ppu.Bg1.colorDepth = bpp2
		//TODO should invalidate everything
		ppu.InvalidateBG(0)
	case 0x2107:
		fmt.Println("BG1SC: ", value)
		ppu.Bg1.tileMapSize = uint16(value & 0x3)
		ppu.Bg1.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.InvalidateBG(0)
	case 0x210B:
		fmt.Println("BG12NBA: ", value)
		ppu.Bg1.charTileAddressBase = (uint16(value&0xF) << 12) & 0x7FFF
		ppu.InvalidateBG(0)
	//TODO add mode 7 scrolling
	case 0x210D:
		ppu.Bg1.hScroll = ppu.BGxnOFS.hFormula(value)
	case 0x210E:
		ppu.Bg1.vScroll = ppu.BGxnOFS.vFormula(value)
	case 0x2115:
		ppu.VRAM.vmain.Setup(value)
		fmt.Println("VMAIN: ", value)
	case 0x2116:
		ppu.VRAM.UpdateAddressLow(value)
		fmt.Println("VMADDLOW: ", value)
	case 0x2117:
		ppu.VRAM.UpdateAddressHigh(value)
		fmt.Println("VMADDHIGH: ", value)
	case 0x2118:
		ppu.VRAM.WriteDataLow(value)
	case 0x2119:
		ppu.VRAM.WriteDataHigh(value)
	case 0x2121:
		ppu.CGRAM.SetAddWord(value)
	case 0x2122:
		ppu.CGRAM.WriteData(value)
	case 0x212C:
		fmt.Println("TM: ", value)
	default:
		return fmt.Errorf("invalid PPU register write at $%04X", addr)
	}
	return nil
}

func (ppu *PPU) getOAMLow() []byte {
	return ppu.OAM.LowTable
}

func (ppu *PPU) getOAMHigh() []byte {
	return ppu.OAM.HighTable
}

func (ppu *PPU) getVRAM() []uint16 {
	return ppu.VRAM.VRAM
}

func (ppu *PPU) getCGRAM() []uint16 {
	return ppu.CGRAM.CGRAM
}

func (ppu *PPU) tryInvalidate(addr uint16) {
	//maybe return true or something if theres a hit so it can stop checking the rest of the bgs
	ppu.Bg1.Invalidate(addr)
}

func (ppu *PPU) InvalidateBG(bgIndex int) {
	if bgIndex >= 0 && bgIndex < len(ppu.bgEpochs) {
		ppu.bgEpochs[bgIndex]++
	}
}
