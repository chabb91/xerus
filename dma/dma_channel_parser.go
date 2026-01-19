package dma

import (
	"SNES_emulator/memory"
)

const (
	fixed     = 0
	decrement = -1
	increment = 1

	direct   = 2
	indirect = 3
)

type direction func(busA uint32, busB byte, validB bool, bus memory.Bus)

// this function is from BSNES
func isValidA(address uint32) bool {
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

	valid := busB != 0x80 || ((busA&0xfe0000) != 0x7e0000 && (busA&0x40e000) != 0x0000) //transfers from WRAM to WRAM are invalid -- from BSNES
	direction(busA, busB, valid, bus)
}

type DmaOperation struct {
	bus     memory.Bus
	channel *DmaChannel

	transferMode     byte
	transferIndex    byte
	transferUnitSize byte
	direction        direction
	step             int
}

func (op *DmaOperation) setup(channel *DmaChannel) *DmaOperation {
	op.channel = channel
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
	return op
}

func (op *DmaOperation) stepCycle() bool {
	channel := op.channel

	transfer(op.transferMode, op.transferIndex, channel.a1b|uint32(channel.a1w), channel.bbad, op.direction, op.bus)
	op.transferIndex++
	if op.transferIndex == op.transferUnitSize {
		op.transferIndex = 0
	}

	channel.a1w += uint16(op.step)
	channel.dasw--

	if channel.dasw != 0 {
		return false
	} else {
		return true
	}
}

type HdmaOperation struct {
	//injected from the parent dma struct
	bus     memory.Bus
	Hdmaen  *byte
	channel *DmaChannel

	transferMode     byte
	transferIndex    byte
	transferUnitSize byte
	direction        direction
	addressingMode   int

	currentAddressPointer *uint16
	currentBankPointer    *uint32

	doTransfer   bool
	isTerminated bool
}

func (op *HdmaOperation) reload() uint64 {
	cycles := uint64(0)
	channel := op.channel

	channel.a2w = channel.a1w
	channel.ntlrx = op.bus.ReadByte(channel.a1b | uint32(channel.a2w))
	channel.a2w++

	op.doTransfer = true
	op.isTerminated = channel.ntlrx == 0

	if op.addressingMode == indirect {
		cycles += op.loadIndirectAddress(channel.ntlrx)
	}
	return cycles
}

func (op *HdmaOperation) setup() {
	channel := op.channel

	op.transferIndex = 0
	op.transferMode = channel.dmap & 7
	switch op.transferMode {
	case 0:
		op.transferUnitSize = 1
	case 1, 2, 6:
		op.transferUnitSize = 2
	case 3, 4, 5, 7:
		op.transferUnitSize = 4
	}

	switch (channel.dmap >> 7) & 1 {
	case 0:
		op.direction = CpuToIo
	case 1:
		op.direction = IoToCpu
	}

	switch (channel.dmap & 0x40) >> 6 {
	case 0:
		op.addressingMode = direct
		op.currentAddressPointer = &op.channel.a2w
		op.currentBankPointer = &op.channel.a1b
	case 1:
		op.addressingMode = indirect
		op.currentAddressPointer = &op.channel.dasw
		op.currentBankPointer = &op.channel.dasb
	}
}

func (op *HdmaOperation) stepCycle() bool {
	if op.transferIndex < op.transferUnitSize {
		transfer(op.transferMode, op.transferIndex, *op.currentBankPointer|uint32(*op.currentAddressPointer), op.channel.bbad, op.direction, op.bus)
		op.transferIndex++
		*op.currentAddressPointer++
	}
	if op.transferIndex == op.transferUnitSize {
		op.transferIndex = 0
		return true
	} else {
		return false
	}
}

func (op *HdmaOperation) stepLineCounter() uint64 {
	cycles := uint64(0)
	ntlrx := &op.channel.ntlrx
	*ntlrx--
	if *ntlrx&0x7F == 0 {
		*ntlrx = op.bus.ReadByte(op.channel.a1b | uint32(op.channel.a2w))
		op.channel.a2w++
		if op.addressingMode == indirect {
			cycles += op.loadIndirectAddress(*ntlrx)
		}
		op.doTransfer = true
		op.isTerminated = *ntlrx == 0
	} else {
		//TODO this might have to go to stepcycle instead but that would break the midframe hdma test rom so who knows
		op.doTransfer = *ntlrx >= 0x80
	}

	return cycles
}

// returns the indirect table load cycle cost and nothing else
func (op *HdmaOperation) loadIndirectAddress(ntlrx byte) uint64 {
	cycles := CYCLE_8
	channel := op.channel
	lo := op.bus.ReadByte(channel.a1b | uint32(channel.a2w))
	channel.a2w++
	if ntlrx == 0 && getNextActiveChannel(*op.Hdmaen, channel.id) == -1 {
		channel.dasw = uint16(lo) << 8
	} else {
		hi := op.bus.ReadByte(channel.a1b | uint32(channel.a2w))
		channel.a2w++
		channel.dasw = uint16(hi)<<8 | uint16(lo)
		cycles <<= 1
	}
	return cycles
}
