package cpu

import "SNES_emulator/memory"

type CPU struct {
	r *registers

	E bool

	instructions       map[byte]Instruction
	currentInstruction Instruction

	bus *memory.Bus
}

func NewCPU(bus memory.Bus) *CPU {
	cpu := &CPU{
		bus:                &bus,
		r:                  &registers{},
		instructions:       NewInstructionMap(),
		currentInstruction: nil,
	}
	return cpu
}

func (c *CPU) Reset() {
	// set emulation flag
	c.E = true

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

func (c *CPU) stepCycle() {
	if c.currentInstruction == nil {
		opcode := c.fetchByte()
		c.currentInstruction = c.instructions[opcode]
		c.currentInstruction.Reset()
	} else if c.currentInstruction.Step(c) {
		c.currentInstruction = nil
	}
}

// mapAddress combines the Program Bank and Program Counter into a 24-bit address.
func (c *CPU) mapPCAddress() uint32 {
	return (uint32(c.r.PB) << 16) | uint32(c.r.PC)
}

// mapDataAddress combines the Data Bank and a 16-bit address into a 24-bit address.
func (c *CPU) mapDataAddress(addr uint16) uint32 {
	return (uint32(c.r.DB) << 16) | uint32(addr)
}

// fetchByte maps PC to 24 bit then goes and reads a byte from memory
// then increases PC by 1
func (c *CPU) fetchByte() byte {
	ret := c.bus.ReadByte(c.mapPCAddress())
	c.r.PC++

	return ret
}
