package interruptchip

import (
	"SNES_emulator/cpu"
	"SNES_emulator/memory"
)

type irqMode func(*InterruptController) bool

type InterruptController struct {
	Htime, Vtime uint16
	rdnmi        byte
	Timeup       byte
	//todo this is just part of the ppu fr fr
	Hvbjoy byte

	cpu *cpu.CPU
	//TODO this also has access to joypad and ppu but of course those dont exist yet.

	autoJoypad       bool
	autoJoypadStatus bool

	nmi bool
	irq irqMode

	bus memory.Bus
}

func NewInterruptController(bus memory.Bus, cpu *cpu.CPU) *InterruptController {
	return &InterruptController{
		cpu:    cpu,
		bus:    bus,
		rdnmi:  0x02,
		Hvbjoy: 0x0,
	}
}

func (ic *InterruptController) FireNmi() {
	if ic.nmi {
		ic.cpu.NmiSignal = true
	}
}

func (ic *InterruptController) SetHvbjoyV(on bool) {
	if on {
		ic.Hvbjoy |= 0x80
	} else {
		ic.Hvbjoy &= 0x7F
	}
}

func (ic *InterruptController) SetHvbjoyH(on bool) {
	if on {
		ic.Hvbjoy |= 0x40
	} else {
		ic.Hvbjoy &= 0xBF
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
		ic.irq = irq00
	case 1:
		ic.irq = irq10
	case 2:
		ic.irq = irq01
	case 3:
		ic.irq = irq11
	}

	if value&1 == 1 {
		ic.autoJoypad = true
	} else {
		ic.autoJoypad = false
	}
}

func (ic *InterruptController) SetHtimeL(value byte) {
	ic.Htime = (ic.Htime & 0xFF00) | uint16(value)
}

func (ic *InterruptController) SetHtimeH(value byte) {
	ic.Htime = (ic.Htime & 0xFF) | uint16(value)<<8
}

func (ic *InterruptController) SetVtimeL(value byte) {
	ic.Vtime = (ic.Vtime & 0xFF00) | uint16(value)
}

func (ic *InterruptController) SetVtimeH(value byte) {
	ic.Vtime = (ic.Vtime & 0xFF) | uint16(value)<<8
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

func irq00(_ *InterruptController) bool {
	return false
}

func irq01(ic *InterruptController) bool {
	//TODO 123=ppu.V
	return ic.Vtime == 123
}

func irq10(ic *InterruptController) bool {
	//TODO 123=ppu.H
	return ic.Htime == 123
}

func irq11(ic *InterruptController) bool {
	return irq01(ic) && irq10(ic)
}
