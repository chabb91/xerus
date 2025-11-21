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
	JoypadController    *JoypadController
	InterruptController *interruptchip.InterruptController
	MulDiv              *muldivchip.MulDiv
	Dma                 *dma.Dma
	Cpu                 *cpu.CPU
	Ppu                 *ppu.PPU

	bus memory.Bus
}

func NewSoC(framebuffer *ui.Framebuffer) *SoC {
	romData, err := cartridge.Load("/home/chabb/Downloads/jonasquinn/nmi_irq/nmi_pf/test_nmi.smc")
	if err != nil {
		panic(err)
	}
	bus := memory.NewBus(cartridge.NewCartridge(romData, cartridge.NewLoRom()))
	soc := &SoC{
		MulDiv: muldivchip.NewMulDiv(),
		Dma:    dma.NewDma(bus),
		Cpu:    cpu.NewCPU(bus),
		Ppu:    ppu.NewPPU(bus),
		bus:    bus,
	}
	soc.JoypadController = NewJoypadController(bus)
	soc.InterruptController = interruptchip.NewInterruptController(bus, soc.Cpu, soc.Ppu)
	soc.Ppu.InterruptScheduler = soc.InterruptController
	soc.Ppu.HdmaScheduler = soc.Dma
	soc.Ppu.Framebuffer = framebuffer
	soc.Ppu.Wrio = &soc.InterruptController.WRIO
	soc.Ppu.Init()

	bus.RegisterRange(0x4200, 0x421F, soc, "internal CPU")
	bus.RegisterRange(0x2100, 0x213F, soc.Ppu, "PPU")
	bus.RegisterRange(0x4016, 0x4017, soc.JoypadController, "Joypad")

	return soc
}

func (soc *SoC) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x4210:
		return soc.InterruptController.ReadRdnmi(), nil
	case 0x4211:
		return soc.InterruptController.ReadTimeUp(), nil
	case 0x4212:
		return soc.InterruptController.ReadHvbjoy(), nil
	case 0x4214:
		return soc.MulDiv.Rddivl, nil
	case 0x4215:
		return soc.MulDiv.Rddivh, nil
	case 0x4216:
		return soc.MulDiv.Rdmpyl, nil
	case 0x4217:
		return soc.MulDiv.Rdmpyh, nil
	case 0x4218:
		return byte(soc.InterruptController.JOY1), nil
	case 0x4219:
		return byte(soc.InterruptController.JOY1 >> 8), nil
	case 0x421A:
		return byte(soc.InterruptController.JOY2), nil
	case 0x421B:
		return byte(soc.InterruptController.JOY2 >> 8), nil
	case 0x421C:
		return byte(soc.InterruptController.JOY3), nil
	case 0x421D:
		return byte(soc.InterruptController.JOY3 >> 8), nil
	case 0x421E:
		return byte(soc.InterruptController.JOY4), nil
	case 0x421F:
		return byte(soc.InterruptController.JOY4 >> 8), nil
	default:
		return 0, fmt.Errorf("invalid internal CPU register read at $%04X", addr)
	}
}

func (soc *SoC) Write(addr uint16, value byte) error {
	switch addr {
	case 0x4200:
		fmt.Println("NMITIMEN: ", value)
		soc.InterruptController.SetNmitimen(value)
	case 0x4201:
		//TODO add Lightgun High-to-Low transition support
		fmt.Println("WRIO: ", value)
		wrio := &soc.InterruptController.WRIO
		if *wrio&0x80 != 0 && value&0x80 == 0 {
			soc.Ppu.LatchHV()
		}

		*wrio = value
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
		soc.InterruptController.SetHtimeL(value)
	case 0x4208:
		fmt.Println("HTIMEH: ", value)
		soc.InterruptController.SetHtimeH(value)
	case 0x4209:
		fmt.Println("VTIMEL: ", value)
		soc.InterruptController.SetVtimeL(value)
	case 0x420A:
		fmt.Println("VTIMEH: ", value)
		soc.InterruptController.SetVtimeH(value)
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
