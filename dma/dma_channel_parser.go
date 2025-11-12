package dma

import (
	"SNES_emulator/memory"
)

const (
	increment = iota
	decrement
	fixed
	direct
	indirect
)

type direction func(busA uint32, busB byte, validB bool, bus memory.Bus)

func isValidA(address uint32) bool {
	//TODO unsure logic is from BSNES
	//A-bus cannot access the B-bus or CPU I/O registers
	if (address & 0x40ff00) == 0x2100 {
		return false //00-3f,80-bf:2100-21ff
	}
	if (address & 0x40fe00) == 0x4000 {
		return false //00-3f,80-bf:4000-41ff
	}
	if (address & 0x40ffe0) == 0x4200 {
		return false //00-3f,80-bf:4200-421f
	}
	if (address & 0x40ff80) == 0x4300 {
		return false //00-3f,80-bf:4300-437f
	}
	return true
}

func CpuToIo(busA uint32, busB byte, validB bool, bus memory.Bus) {
	if validB {
		if isValidA(busA & 0xFFFFFF) {
			bus.WriteByte(0x2100+uint32(busB), bus.ReadByte(busA&0xFFFFFF))
		} else {
			bus.WriteByte(0x2100+uint32(busB), 0)
		}
	}
}

func IoToCpu(busA uint32, busB byte, validB bool, bus memory.Bus) {
	if isValidA(busA & 0xFFFFFF) {
		if validB {
			bus.WriteByte(busA&0xFFFFFF, bus.ReadByte(0x2100+uint32(busB)))
		} else {
			bus.WriteByte(busA&0xFFFFFF, 0)
		}
	}
}

// thx byuu
func transfer(mode byte, index byte, busA uint32, busB byte, direction direction, bus memory.Bus) {
	switch mode {
	case 1, 5:
		busB += (index & 1)
	case 3, 7:
		busB += (index & 0b10) >> 1
	case 4:
		busB += index
	default:
		//busB unchanged
	}

	//transfers from WRAM to WRAM are invalid
	//TODO unsure logic is from BSNES
	valid := busB != 0x80 || ((busA&0xfe0000) != 0x7e0000 && (busA&0x40e000) != 0x0000)
	direction(busA, busB, valid, bus)
}

type DmaOperation struct {
	bus memory.Bus

	transferMode     byte
	transferIndex    byte
	transferUnitSize byte
	direction        direction
	step             int

	busA uint32
	busB byte

	size uint16
}

func (op *DmaOperation) setup(channel DmaChannel) {
	op.transferIndex = 0
	op.transferMode = channel.dmap & 0b111
	switch op.transferMode {
	case 0:
		op.transferUnitSize = 1
	case 1, 2, 6:
		op.transferUnitSize = 2
	case 3, 4, 5, 7:
		op.transferUnitSize = 4
	}

	switch (channel.dmap & 0b11000) >> 3 {
	case 0:
		op.step = increment
	case 2:
		op.step = decrement
	case 1, 3:
		op.step = fixed
	}

	switch (channel.dmap >> 7) & 1 {
	case 0:
		op.direction = CpuToIo
	case 1:
		op.direction = IoToCpu
	}

	op.busA = uint32(channel.a1b)<<16 | uint32(channel.a1th)<<8 | uint32(channel.a1tl)
	op.busB = channel.bbad

	op.size = uint16(channel.dash)<<8 | uint16(channel.dasl)
}

func (op *DmaOperation) stepCycle() bool {
	if op.size > 0 {
		transfer(op.transferMode, op.transferIndex, op.busA, op.busB, op.direction, op.bus)
		op.size--
		op.transferIndex++

		if op.transferIndex >= op.transferUnitSize {
			op.transferIndex = 0
		}

		switch op.step {
		case decrement:
			op.busA = (op.busA - 1) & 0xFFFFFF
		case increment:
			op.busA = (op.busA + 1) & 0xFFFFFF
		default:
			//fixed
		}
	}

	if op.size > 0 {
		return false
	} else {
		return true
	}
}

type HdmaOperation struct {
	bus memory.Bus

	transferMode     byte
	transferIndex    byte
	transferUnitSize byte
	direction        direction
	addressingMode   int

	//table start address
	//the bank is constant, the mid and lo are the reload values for 43x8/9
	//i believe in indirect mode the table reads two bytes and loads it into indirectAddr which then is used for the actual write
	tableAddr        uint32
	indirectAddr     uint32
	indirectBank     uint32
	tableCurrentAddr uint32
	busB             byte

	currentAddressPointer *uint32

	lineCounter byte
	repeat      bool

	doTransfer   bool
	isTerminated bool
}

func (op *HdmaOperation) reload(channel DmaChannel) {
	op.setup(channel)
	ntlrx := op.bus.ReadByte(op.tableAddr)
	op.tableAddr++
	op.lineCounter = ntlrx & 0x7F
	op.repeat = ntlrx&0x80 != 0

	op.doTransfer = true
	op.isTerminated = ntlrx == 0

	if op.addressingMode == indirect {
		lo := op.bus.ReadByte(op.tableAddr)
		op.tableAddr++
		hi := op.bus.ReadByte(op.tableAddr)
		op.tableAddr++
		op.indirectAddr = op.indirectBank | uint32(hi)<<8 | uint32(lo)
	}
}

func (op *HdmaOperation) setup(channel DmaChannel) {
	op.transferIndex = 0
	op.transferMode = channel.dmap & 0b111
	switch op.transferMode {
	case 0:
		op.transferUnitSize = 1
	case 1, 2, 6:
		op.transferUnitSize = 2
	case 3, 4, 5, 7:
		op.transferUnitSize = 4
	}

	op.tableAddr = uint32(channel.a1b)<<16 | uint32(channel.a1th)<<8 | uint32(channel.a1tl)
	op.indirectBank = uint32(channel.dasb) << 16
	op.indirectAddr = op.indirectBank | uint32(channel.dash)<<8 | uint32(channel.dasl)
	op.tableCurrentAddr = uint32(channel.a1b)<<16 | uint32(channel.a2ah)<<8 | uint32(channel.a2al)
	op.busB = channel.bbad

	op.lineCounter = channel.ntlrx & 0x7F
	op.repeat = channel.ntlrx&0x80 != 0 || channel.ntlrx == 0 //at setup ntlrx == 0 is treated as 0x80 instead

	op.isTerminated = false
	op.doTransfer = false

	switch (channel.dmap & 0x40) >> 6 {
	case 0:
		op.addressingMode = direct
	case 1:
		op.addressingMode = indirect
	}

	switch (channel.dmap >> 7) & 1 {
	case 0:
		op.direction = CpuToIo
	case 1:
		op.direction = IoToCpu
	}
}

func (op *HdmaOperation) stepCycle() bool {
	if op.transferIndex < op.transferUnitSize {
		if op.addressingMode == direct {
			transfer(op.transferMode, op.transferIndex, op.tableAddr, op.busB, op.direction, op.bus)
			op.transferIndex++
			op.tableAddr++
		} else {
			transfer(op.transferMode, op.transferIndex, op.indirectAddr, op.busB, op.direction, op.bus)
			op.transferIndex++
			op.indirectAddr++
		}
	}
	if op.transferIndex == op.transferUnitSize {
		op.doTransfer = op.repeat
		op.transferIndex = 0
		op.stepLineCounter()
		return true
	} else {
		return false
	}
}

func (op *HdmaOperation) stepLineCounter() {
	op.lineCounter--
	if op.lineCounter == 0 {
		ntlrx := op.bus.ReadByte(op.tableAddr)
		op.lineCounter = ntlrx & 0x7F
		op.repeat = ntlrx&0x80 != 0
		op.tableAddr++
		if op.addressingMode == indirect {
			lo := op.bus.ReadByte(op.tableAddr)
			op.tableAddr++
			hi := op.bus.ReadByte(op.tableAddr)
			op.tableAddr++
			op.indirectAddr = op.indirectBank | uint32(hi)<<8 | uint32(lo)
		}
		op.doTransfer = true
		op.isTerminated = ntlrx == 0
	}

	//handling the midframe hdma. something that repeats has to always do the transfer
	if op.repeat && !op.doTransfer {
		op.doTransfer = true
	}
}
