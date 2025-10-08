package ppu

import "fmt"

type PPU struct {
	OAM   *OAMController
	VRAM  *VRAMController
	CGRAM *CGRAMController

	FBlank, VBlank, HBlank bool
	screenBrightness       byte
}

func NewPPU() *PPU {
	return &PPU{
		OAM:   NewOAM(),
		VRAM:  NewVRAM(),
		CGRAM: NewCGRAM(),
	}
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
	case 0x2116:
		ppu.VRAM.UpdateAddressLow(value)
	case 0x2117:
		ppu.VRAM.UpdateAddressHigh(value)
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
