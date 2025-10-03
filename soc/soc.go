package soc

import (
	"SNES_emulator/cpu"
	"SNES_emulator/dma"
	"SNES_emulator/memory"
	"SNES_emulator/soc/muldivchip"
	"fmt"
)

type SoC struct {
	mulDiv muldivchip.MulDiv
	dma    dma.Dma
	cpu    cpu.CPU

	bus memory.Bus
}

func (soc *SoC) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x4214:
		return soc.mulDiv.Rddivl, nil
	case 0x4215:
		return soc.mulDiv.Rddivh, nil
	case 0x4216:
		return soc.mulDiv.Rdmpyl, nil
	case 0x4217:
		return soc.mulDiv.Rdmpyh, nil
	default:
		return 0, fmt.Errorf("invalid internal CPU register read at $%04X", addr)
	}
}

func (soc *SoC) Write(addr uint16, value byte) error {
	switch addr {
	case 0x4202:
		soc.mulDiv.Wrmpya = value
	case 0x4203:
		soc.mulDiv.SetMultiplicandB(value)
	case 0x4204:
		soc.mulDiv.Wrdivl = value
	case 0x4205:
		soc.mulDiv.Wrdivh = value
	case 0x4206:
		soc.mulDiv.SetDivisorB(value)
	case 0x420B:
		soc.dma.Mdmaen = value
	case 0x420C:
		soc.dma.Hdmaen = value
	default:
		return fmt.Errorf("invalid internal CPU register write at $%04X", addr)
	}
	return nil
}
