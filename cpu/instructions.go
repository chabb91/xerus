package cpu

// Instruction represents a single CPU instruction, executed one cycle at a time.
type Instruction interface {
	// Step performs one cycle of the instruction's execution.
	// It returns true if the instruction is complete, false otherwise.
	Step(cpu *CPU) bool
	Reset()
}

func NewInstructionMap() map[byte]Instruction {
	ret := make(map[byte]Instruction)

	ret[0x4C] = &I4C{}
	ret[0x6C] = &I6C{}

	return ret
}

// I4C represents the JMP $XXXX instruction (opcode 0x4C)
type I4C struct {
	state    int
	lowByte  byte
	highByte byte
	address  uint16
}

// Step runs one cycle of the JMP instruction
func (i *I4C) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.address = createWord(i.highByte, i.lowByte)
		cpu.r.PC = i.address
		return true
	}
	return false
}

func (i *I4C) Reset() {
	i.state = 0
}

// I6C represents the JMP [nnnn] instruction (opcode 0x6C)
type I6C struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
func (i *I6C) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		//this read doesnt have to be mapped because it defaults to bank 0x00
		i.lowByte = cpu.bus.ReadByte(uint32(i.pointerAddress))
		i.state++
	case 3:
		var highByteAddress uint32
		/* the tests are passing without the glitch so its removed for now
		if i.pointerAddress&0x00FF == 0x00FF {
			// The glitch! The high byte is fetched from the start of the same page.
			highByteAddress = uint32(i.pointerAddress & 0xFF00)
		} else {
		*/
		highByteAddress = uint32(i.pointerAddress + 1)
		//}
		i.highByte = cpu.bus.ReadByte(highByteAddress)

		finalAddress := createWord(i.highByte, i.lowByte)
		cpu.r.PC = finalAddress

		return true
	}
	return false
}

func (i *I6C) Reset() {
	i.state = 0
}
