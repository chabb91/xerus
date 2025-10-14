package ppu

import "fmt"

type PPU struct {
	OAM   *OAMController
	VRAM  *VRAMController
	CGRAM *CGRAMController

	Bg1 *Background1

	FBlank, VBlank, HBlank bool
	screenBrightness       byte
}

func NewPPU() *PPU {
	ppu := &PPU{
		OAM:   NewOAM(),
		VRAM:  NewVRAM(),
		CGRAM: NewCGRAM(),
		Bg1:   NewBackground1(),
	}
	ppu.VRAM.ppu = ppu
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
	if addr == 0x212C {
		fmt.Println("TM: ", value)
	}
	if addr == 0x210B {
		fmt.Println("BG12NBA: ", value)
		ppu.Bg1.charTileAddressBase = (uint16(value&0xF) << 12) & 0x7FFF
	}
	if addr == 0x2107 {
		fmt.Println("BG1SC: ", value)
		ppu.Bg1.tileMapSize = uint16(value & 0x3)
		ppu.Bg1.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
	}
	if addr == 0x2105 {
		fmt.Println("BGMODE: ", value)
		ppu.Bg1.charTileSize = (value >> 4) & 1
		ppu.Bg1.colorDepth = 2
	}
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
	default:
		return fmt.Errorf("invalid PPU register write at $%04X", addr)
	}
	return nil
}
