package dma

import "SNES_emulator/memory"

const (
	increment = iota
	decrement
	fixed
)

type direction func(busA uint32, busB byte, bus memory.Bus)

type Transfer func(busA uint32, busB byte, direction direction, bus memory.Bus)

func CpuToIo(busA uint32, busB byte, bus memory.Bus) {
	bus.WriteByte(0x2100+uint32(busB), bus.ReadByte(busA&0xFFFFFF))
}

func IoToCpu(busA uint32, busB byte, bus memory.Bus) {
	bus.WriteByte(busA&0xFFFFFF, bus.ReadByte(0x2100+uint32(busB)))
}

func TransferMode0(busA uint32, busB byte, direction direction, bus memory.Bus) {
	direction(busA, busB, bus)
}

func TransferMode1(busA uint32, busB byte, direction direction, bus memory.Bus) {
	direction(busA, busB, bus)
	direction(busA+1, busB+1, bus)
}

func TransferMode2(busA uint32, busB byte, direction direction, bus memory.Bus) {
	direction(busA, busB, bus)
	direction(busA+1, busB, bus)
}

func TransferMode3(busA uint32, busB byte, direction direction, bus memory.Bus) {
	direction(busA, busB, bus)
	direction(busA+1, busB, bus)

	direction(busA+2, busB+1, bus)
	direction(busA+3, busB+1, bus)
}

func TransferMode4(busA uint32, busB byte, direction direction, bus memory.Bus) {
	for i := range uint32(4) {
		direction(busA+i, busB+byte(i), bus)
	}
}

func TransferMode5(busA uint32, busB byte, direction direction, bus memory.Bus) {
	direction(busA, busB, bus)
	direction(busA+1, busB+1, bus)

	direction(busA+2, busB, bus)
	direction(busA+3, busB+1, bus)
}

type DmaOperation struct {
	bus memory.Bus

	direction           direction
	transfer            Transfer
	step                int
	transferUnitSize    uint32
	masterCyclesPerUnit uint32

	busA uint32
	busB byte

	size uint16
}

func (op *DmaOperation) setup(channel DmaChannel) {
	switch channel.dmap & 0b111 {
	case 0:
		op.transfer = TransferMode0
		op.transferUnitSize = 1
	case 1:
		op.transfer = TransferMode1
		op.transferUnitSize = 2
	case 2, 6:
		op.transfer = TransferMode2
		op.transferUnitSize = 2
	case 3, 7:
		op.transfer = TransferMode3
		op.transferUnitSize = 4
	case 4:
		op.transfer = TransferMode4
		op.transferUnitSize = 4
	case 5:
		op.transfer = TransferMode5
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

	op.masterCyclesPerUnit = op.transferUnitSize * 8
}

func (op *DmaOperation) stepCycle() bool {
	if op.size > 0 {
		op.transfer(op.busA, op.busB, op.direction, op.bus)
		op.size--

		switch op.step {
		case decrement:
			op.busA -= uint32(op.transferUnitSize)
		case increment:
			op.busA += uint32(op.transferUnitSize)
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
