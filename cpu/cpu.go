package cpu

import (
	"SNES_emulator/memory"
)

type CPU struct {
	r *registers

	instructions       map[byte]Instruction
	currentInstruction Instruction

	bus memory.Bus
}

func NewCPU(bus memory.Bus) *CPU {
	cpu := &CPU{
		bus:                bus,
		r:                  &registers{},
		instructions:       NewInstructionMap(),
		currentInstruction: nil,
	}
	return cpu
}

// the 4 hardware interrupts
func (c *CPU) Reset() {
	// set emulation flag
	c.r.E = true

	c.r.PB = 0x00
	c.r.DB = 0x00
	c.r.D = 0x0000

	// the default stack head in emulation mode
	c.r.S = 0x01FF

	// set M X and I to 1
	c.r.P = 0x34

	// read the 16-bit Reset Vector from the bus
	c.r.PC = createWord(c.bus.ReadByte(0x00FFFD), c.bus.ReadByte(0x00FFFC))
}

func (c *CPU) IRQ() {
	if !c.r.E {
		c.PushByte(c.r.PB)
	}
	c.PushWord(c.r.PC)

	if c.r.E {
		c.PushByte(c.r.P & ^FlagX)
	} else {
		c.PushByte(c.r.P)
	}

	c.r.PB = 0x00

	if c.r.E {
		c.r.PC = createWord(c.bus.ReadByte(0x00FFFF), c.bus.ReadByte(0x00FFFE))
	} else {
		c.r.PC = createWord(c.bus.ReadByte(0x00FFEF), c.bus.ReadByte(0x00FFEE))
	}

	c.r.setFlag(FlagD, true)
	c.r.setFlag(FlagI, false)
}

func (c *CPU) stepCycle() bool {
	if c.currentInstruction == nil {
		opcode := c.fetchByte()
		c.currentInstruction = c.instructions[opcode]
		c.currentInstruction.Reset()
	} else if c.currentInstruction.Step(c) {
		c.currentInstruction = nil
		return true
	}
	return false
}

// mapAddress combines the Program Bank and Program Counter into a 24-bit address.
func (c *CPU) mapPCAddress() uint32 {
	return c.mapAddressToBank(c.r.PB, c.r.PC)
}

// mapDataAddress combines the Data Bank and a 16-bit address into a 24-bit address.
func (c *CPU) mapDataAddress(addr uint16) uint32 {
	return c.mapAddressToBank(c.r.DB, addr)
}

// maps a 2 byte address to a 8 bit bank returning a 24 bit full memory address
func (c *CPU) mapAddressToBank(bank byte, addr uint16) uint32 {
	return (uint32(bank) << 16) | uint32(addr)
}

// fetchByte maps PC to 24 bit then goes and reads a byte from memory
// then increases PC by 1
func (c *CPU) fetchByte() byte {
	ret := c.bus.ReadByte(c.mapPCAddress())
	c.r.PC++

	return ret
}

// PushByte pushes one byte onto the stack and updates SP.
func (cpu *CPU) PushByte(val byte) {
	addr := cpu.r.GetStackAddr()
	cpu.bus.WriteByte(addr, val)
	cpu.r.S--
}

// PopByte pops one byte from the stack and updates SP.
func (cpu *CPU) PopByte() byte {
	cpu.r.S++
	addr := cpu.r.GetStackAddr()
	return cpu.bus.ReadByte(addr)
}

// PushWord pushes a 16-bit word onto the stack (high byte first).
func (cpu *CPU) PushWord(val uint16) {
	high, low := splitWord(val)
	cpu.PushByte(high)
	cpu.PushByte(low)
}

// PopWord pops a 16-bit word from the stack (low byte first).
func (cpu *CPU) PopWord() uint16 {
	low := cpu.PopByte()
	high := cpu.PopByte()
	return createWord(high, low)
}
