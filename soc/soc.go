package soc

import (
	"SNES_emulator/cartridge"
	"SNES_emulator/cpu"
	"SNES_emulator/dma"
	"SNES_emulator/memory"
	"SNES_emulator/ppu"
	"SNES_emulator/soc/interruptchip"
	"SNES_emulator/soc/muldivchip"
	"SNES_emulator/ui"
	"fmt"
)

type SoC struct {
	MulDiv              *muldivchip.MulDiv
	InterruptController *interruptchip.InterruptController
	Dma                 *dma.Dma
	Cpu                 *cpu.CPU
	Ppu                 *ppu.PPU

	bus memory.Bus
}

func NewSoC(framebuffer *ui.Framebuffer) *SoC {
	romData, err := cartridge.Load("/home/chabb/Downloads/hdma-2100-glitch-2ch-0a.sfc")
	if err != nil {
		panic(err)
	}
	bus := memory.NewBus(cartridge.NewCartridge(romData, cartridge.NewLoRom()))
	soc := &SoC{
		MulDiv: muldivchip.NewMulDiv(),
		Dma:    dma.NewDma(bus),
		Cpu:    cpu.NewCPU(bus),
		Ppu:    ppu.NewPPU(),
		bus:    bus,
	}
	soc.InterruptController = interruptchip.NewInterruptController(bus, soc.Cpu)
	soc.Ppu.InterruptScheduler = soc.InterruptController
	soc.Ppu.HdmaScheduler = soc.Dma
	soc.Ppu.Framebuffer = framebuffer

	bus.RegisterRange(0x4200, 0x4217, soc, "internal CPU")
	bus.RegisterRange(0x2100, 0x213F, soc.Ppu, "PPU")

	return soc
}

func (soc *SoC) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x4210:
		//return 0x82, fmt.Errorf("internal CPU register $%04X is unimplemented", addr)
		ret := soc.InterruptController.ReadRdnmi()
		return ret, nil
	case 0x4212:
		//TODO the JOY part of this is not implemented and its missing the specific timings in the ppu
		//just using the general hblank/vblank for now
		return soc.InterruptController.Hvbjoy, nil
	case 0x4214:
		return soc.MulDiv.Rddivl, nil
	case 0x4215:
		return soc.MulDiv.Rddivh, nil
	case 0x4216:
		return soc.MulDiv.Rdmpyl, nil
	case 0x4217:
		return soc.MulDiv.Rdmpyh, nil
	default:
		return 0, fmt.Errorf("invalid internal CPU register read at $%04X", addr)
	}
}

func (soc *SoC) Write(addr uint16, value byte) error {
	switch addr {
	case 0x4200:
		fmt.Println("NMITIMEN: ", value)
		//TODO THIS IS OMEGA EXPERIMENTAL
		soc.InterruptController.SetNmitimen(value)
	case 0x4202:
		soc.MulDiv.Wrmpya = value
	case 0x4203:
		soc.MulDiv.SetMultiplicandB(value)
	case 0x4204:
		soc.MulDiv.Wrdivl = value
	case 0x4205:
		soc.MulDiv.Wrdivh = value
	case 0x4206:
		soc.MulDiv.SetDivisorB(value)
	case 0x4207:
		fmt.Println("HTIMEL: ", value)
	case 0x420B:
		soc.Dma.Mdmaen = value
	case 0x420C:
		//soc.Dma.Hdmaen = value
		soc.Dma.SetHdmaen(value)
	case 0x420D:
		soc.bus.SetMEMSEL(value)
	default:
		return fmt.Errorf("invalid internal CPU register write at $%04X", addr)
	}
	return nil
}
