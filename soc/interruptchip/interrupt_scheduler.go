package interruptchip

import (
	"SNES_emulator/cpu"
	"SNES_emulator/memory"
	"SNES_emulator/ppu"
)

type irqMode func(*InterruptController) bool

type InterruptController struct {
	Htime, Vtime uint16
	rdnmi        byte
	Timeup       byte
	hvbjoy       byte

	cpu *cpu.CPU
	ppu *ppu.PPU

	nmi bool

	autoJoypad bool

	JOY1 uint16
	JOY2 uint16
	JOY3 uint16
	JOY4 uint16

	bus memory.Bus
}

func NewInterruptController(bus memory.Bus, cpu *cpu.CPU, ppu *ppu.PPU) *InterruptController {
	return &InterruptController{
		cpu:    cpu,
		ppu:    ppu,
		bus:    bus,
		rdnmi:  0x02,
		hvbjoy: 0x0,
		Htime:  0x1FF,
		Vtime:  0x1FF,
	}
}

func (ic *InterruptController) FireNmi() {
	if ic.nmi {
		ic.cpu.NmiSignal = true
	}
}

func (ic *InterruptController) FireIrq() {
	ic.cpu.IrqSignal = true
}

func (ic *InterruptController) SetHvbjoyV(on bool) {
	if on {
		ic.hvbjoy |= 0x80
	} else {
		ic.hvbjoy &= 0x7F
	}
}

func (ic *InterruptController) SetHvbjoyH(on bool) {
	if on {
		ic.hvbjoy |= 0x40
	} else {
		ic.hvbjoy &= 0xBF
	}
}

func (ic *InterruptController) SetHvbjoyA(inProgress bool) {
	if ic.autoJoypad {
		if inProgress {
			ic.hvbjoy |= 1
			ic.performAutoJoypadRead()
		} else {
			ic.hvbjoy &= 0xFE
		}
	}
}

func (ic *InterruptController) ReadHvbjoy() byte {
	return (ic.bus.GetOpenBus() & 0x3E) | ic.hvbjoy
}

// TODO this should be done periodically over 1056 dots or 4 times of that master cycles
func (ic *InterruptController) performAutoJoypadRead() {
	ic.bus.WriteByte(0x4016, 1)
	ic.bus.WriteByte(0x4016, 0)

	ic.JOY1, ic.JOY2, ic.JOY3, ic.JOY4 = 0, 0, 0, 0

	for i := range 16 {
		ca := uint16(ic.bus.ReadByte(0x4016))
		db := uint16(ic.bus.ReadByte(0x4017))
		ic.JOY1 |= (ca & 1) << i
		ic.JOY3 |= ((ca >> 1) & 1) << i
		ic.JOY2 |= (db & 1) << i
		ic.JOY4 |= ((db >> 1) & 1) << i
	}
}

func (ic *InterruptController) SetNmitimen(value byte) {
	if value >= 0x80 {
		ic.nmi = true
	} else {
		ic.nmi = false
	}

	switch (value >> 4) & 3 {
	case 0:
		ic.ppu.IrqFunc = nil
	case 1:
		ic.ppu.IrqFunc = func() bool { return irq10(ic, ic.ppu) }
	case 2:
		ic.ppu.IrqFunc = func() bool { return irq01(ic, ic.ppu) }
	case 3:
		ic.ppu.IrqFunc = func() bool { return irq11(ic, ic.ppu) }
	}

	if value&1 == 1 {
		ic.autoJoypad = true
	} else {
		ic.autoJoypad = false
	}
}

func (ic *InterruptController) SetHtimeL(value byte) {
	ic.Htime = (ic.Htime & 0x1F00) | uint16(value)
}

func (ic *InterruptController) SetHtimeH(value byte) {
	ic.Htime = (ic.Htime & 0xFF) | ((uint16(value) << 8) & 1)
}

func (ic *InterruptController) SetVtimeL(value byte) {
	ic.Vtime = (ic.Vtime & 0x1F00) | uint16(value)
}

func (ic *InterruptController) SetVtimeH(value byte) {
	ic.Vtime = (ic.Vtime & 0xFF) | ((uint16(value) << 8) & 1)
}

// used as the actual register
func (ic *InterruptController) ReadRdnmi() byte {
	ret := (ic.rdnmi & 0x8F) | (ic.bus.GetOpenBus() & 0x70)
	ic.rdnmi &= 0x7F

	return ret
}

// used by the ppu to set/unset the registers nmi indicator as needed
func (ic *InterruptController) SetRdnmi(nmiOn bool) {
	if nmiOn {
		ic.rdnmi |= 0x80
	} else {
		ic.rdnmi &= 0x7F
	}
}

func (ic *InterruptController) SetTimeUp() {
	ic.Timeup = 0x80
}

func (ic *InterruptController) ReadTimeUp() byte {
	ret := (ic.Timeup & 0x80) | (ic.bus.GetOpenBus() & 0x7F)
	ic.Timeup = 0

	return ret
}

func irq01(ic *InterruptController, ppu *ppu.PPU) bool {
	return int(ic.Vtime+2) == ppu.V
}

func irq10(ic *InterruptController, ppu *ppu.PPU) bool {
	return int(ic.Htime+3) == ppu.H
}

func irq11(ic *InterruptController, ppu *ppu.PPU) bool {
	return int(ic.Htime+3) == ppu.H && int(ic.Vtime+2) == ppu.V
}
