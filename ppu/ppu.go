package ppu

import "fmt"

type PPU struct {
	VRAM  []uint16 //video RAM
	CGRAM []byte   //Color/Paletter RAM
	OAM   []byte   //object attribute memory/sprites

	vmain *VMAIN
	vmadd uint16

	//absolute cringe VERY speshul case for VRAM register reads
	vmLatchedValue uint16

	FBlank, VBlank, HBlank bool
}

func NewPPU() *PPU {
	return &PPU{
		vmain: newVMAIN(),
		VRAM:  make([]uint16, 0x8000),
		CGRAM: make([]byte, 0x200),
		OAM:   make([]byte, 0x220)}
}

// Some of these registers can only be read and written to at specific times defined by the blanking periods
// TODO
func (ppu *PPU) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x2139:
		ret := byte(ppu.vmLatchedValue)
		remapped_addr := ppu.vmain.remap(ppu.vmadd) & 0x7FFF
		ppu.vmLatchedValue = ppu.VRAM[remapped_addr]

		if !ppu.vmain.incrementOnHighByte {
			ppu.vmadd += ppu.vmain.incrementAmount
		}

		return ret, nil
	case 0x213A:
		ret := byte(ppu.vmLatchedValue >> 8)
		remapped_addr := ppu.vmain.remap(ppu.vmadd) & 0x7FFF
		ppu.vmLatchedValue = ppu.VRAM[remapped_addr]

		if ppu.vmain.incrementOnHighByte {
			ppu.vmadd += ppu.vmain.incrementAmount
		}

		return ret, nil
	default:
		return 0, fmt.Errorf("invalid PPU register read at $%04X", addr)
	}
}

func (ppu *PPU) Write(addr uint16, value byte) error {
	switch addr {
	case 0x2115:
		ppu.vmain.Setup(value)
	case 0x2116:
		ppu.vmadd = (ppu.vmadd & 0xFF00) | uint16(value)
	case 0x2117:
		ppu.vmadd = (ppu.vmadd & 0xFF) | (uint16(value) << 8)
	case 0x2118:
		remapped_addr := ppu.vmain.remap(ppu.vmadd) & 0x7FFF
		ppu.VRAM[remapped_addr] = (ppu.VRAM[remapped_addr] & 0xFF00) | uint16(value)

		if !ppu.vmain.incrementOnHighByte {
			ppu.vmadd += ppu.vmain.incrementAmount
		}
	case 0x2119:
		remapped_addr := ppu.vmain.remap(ppu.vmadd) & 0x7FFF
		ppu.VRAM[remapped_addr] = (ppu.VRAM[remapped_addr] & 0x00FF) | (uint16(value) << 8)

		if ppu.vmain.incrementOnHighByte {
			ppu.vmadd += ppu.vmain.incrementAmount
		}
	default:
		return fmt.Errorf("invalid PPU register write at $%04X", addr)
	}
	return nil
}
