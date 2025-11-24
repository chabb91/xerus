package interruptchip

import (
	"SNES_emulator/cpu"
	"SNES_emulator/memory"
	"SNES_emulator/ppu"
)

const CHIP_5A22_VERSION = byte(2)

type InterruptController struct {
	Htime, Vtime uint16
	rdnmi        byte
	Timeup       byte
	hvbjoy       byte

	cpu *cpu.CPU
	ppu *ppu.PPU

	nmi       bool
	rdnmiRead bool //reading $4210 will prevent an NMI from firing

	autoJoypad bool

	WRIO byte //not sure where to put it its related to controllers ig

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
		rdnmi:  CHIP_5A22_VERSION,
		hvbjoy: 0x0,
		Htime:  0x1FF,
		Vtime:  0x1FF,
		WRIO:   0xFF,
	}
}

func (ic *InterruptController) FireNmi() {
	if ic.nmi && !ic.rdnmiRead {
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
		// strobing (lowering and raising) $4200 triggers an NMI after vblank -byuu
		if ic.hvbjoy >= 0x80 && !ic.nmi && !ic.rdnmiRead {
			ic.cpu.NmiSignal = true
		}
		ic.nmi = true
	} else {
		ic.nmi = false
	}

	switch (value >> 4) & 3 {
	case 0:
		ic.ppu.IrqFunc = nil
		ic.Timeup = 0
		ic.cpu.IrqSignal = false
	case 1:
		ic.ppu.IrqFunc = func() bool { return irqX(ic, ic.ppu) }
	case 2:
		ic.ppu.IrqFunc = func() bool { return irqY(ic, ic.ppu) }
	case 3:
		ic.ppu.IrqFunc = func() bool { return irqXY(ic, ic.ppu) }
	}

	if value&1 == 1 {
		ic.autoJoypad = true
	} else {
		ic.autoJoypad = false
	}
}

// TODO apparently setting Vtime to a number the ppu is currently on should re fire irq but that fails the only test i have for it
func (ic *InterruptController) SetHtimeL(value byte) {
	ic.Htime = (ic.Htime & 0x1F00) | uint16(value)
}

func (ic *InterruptController) SetHtimeH(value byte) {
	ic.Htime = (ic.Htime & 0xFF) | (uint16(value&1) << 8)
}

func (ic *InterruptController) SetVtimeL(value byte) {
	ic.Vtime = (ic.Vtime & 0x1F00) | uint16(value)
}

func (ic *InterruptController) SetVtimeH(value byte) {
	ic.Vtime = (ic.Vtime & 0xFF) | (uint16(value&1) << 8)
}

// TODO incpmlete
// 7 is shared with port2 pin6
// 6 is port1 pin6
// and the other bits are random external devices
func (ic *InterruptController) ReadRdio() byte {
	return ic.ppu.LatchFlag << 7 & (^ic.WRIO)
}

// used as the actual register
func (ic *InterruptController) ReadRdnmi() byte {
	ret := (ic.rdnmi & 0x8F) | (ic.bus.GetOpenBus() & 0x70)
	ic.rdnmi &= 0x7F
	if !ic.rdnmiRead {
		ic.rdnmiRead = ret >= 0x80
	}

	return ret
}

// used by the ppu to set/unset the registers nmi indicator as needed
func (ic *InterruptController) SetRdnmi(nmiOn bool) {
	if nmiOn {
		ic.rdnmi |= 0x80
	} else {
		ic.rdnmi &= 0x7F
		ic.rdnmiRead = false
	}
}

func (ic *InterruptController) SetTimeUp() {
	ic.Timeup = 0x80
}

func (ic *InterruptController) ReadTimeUp() byte {
	ret := (ic.Timeup & 0x80) | (ic.bus.GetOpenBus() & 0x7F)
	//if ic.ppu.IrqFunc == nil || !ic.ppu.IrqFunc() {
	ic.Timeup = 0
	ic.cpu.IrqSignal = false
	//}

	return ret
}

// TODO create a mechanic that properly counts the LONGLINE dot number in PAL mode and
// adjusts <340 to 341 accordingly
func irqY(ic *InterruptController, ppu *ppu.PPU) bool {
	return int(ic.Vtime) == ppu.V && ppu.H == 0
}

// irq cannot be latched over H=339 only in PAL longline. i count dots to <341 but
// the real snes counts <340 where 4 dots are 5 master cycles long
func irqX(ic *InterruptController, ppu *ppu.PPU) bool {
	return ppu.H < 340 && int(ic.Htime) == ppu.H
}

func irqXY(ic *InterruptController, ppu *ppu.PPU) bool {
	return ppu.H < 340 && int(ic.Htime) == ppu.H && int(ic.Vtime) == ppu.V
}
