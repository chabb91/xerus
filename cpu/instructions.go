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
	ret[0x5C] = &I5C{}
	ret[0x6C] = &I6C{}
	ret[0x7C] = &I7C{}
	ret[0xDC] = &IDC{}
	ret[0xFC] = &IFC{}
	ret[0x20] = &I20{}
	ret[0x22] = &I22{}

	ret[0x40] = &I40{}
	ret[0x6B] = &I6B{}
	ret[0x60] = &I60{}

	ret[0x80] = &I80{}
	ret[0x82] = &I82{}

	ret[0x10] = &I10{}
	ret[0x30] = &I30{}

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

// I5C represents the JMP $XXXXXX instruction (opcode 0x5C)
type I5C struct {
	state    int
	lowByte  byte
	highByte byte
	pbByte   byte
	address  uint16
}

// Step runs one cycle of the JMP instruction
func (i *I5C) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.state++
	case 2:
		i.pbByte = cpu.fetchByte()
		i.address = createWord(i.highByte, i.lowByte)
		cpu.r.PC = i.address
		cpu.r.PB = i.pbByte
		return true
	}
	return false
}

func (i *I5C) Reset() {
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

// I7C represents the JMP [nnnn+X] instruction (opcode 0x7C)
type I7C struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
func (i *I7C) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.pointerAddress += cpu.r.GetX()
		i.state++
	case 3:
		i.lowByte = cpu.bus.ReadByte(cpu.mapAddressToBank(cpu.r.PB, i.pointerAddress))
		i.state++
	case 4:
		i.highByte = cpu.bus.ReadByte(cpu.mapAddressToBank(cpu.r.PB, i.pointerAddress+1))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *I7C) Reset() {
	i.state = 0
}

// IDC represents the JMP FAR[nnnn] instruction
type IDC struct {
	state int

	lowByte  byte
	highByte byte
	pbByte   byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
func (i *IDC) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.lowByte = cpu.bus.ReadByte(uint32(i.pointerAddress))
		i.state++
	case 3:
		i.highByte = cpu.bus.ReadByte(uint32(i.pointerAddress + 1))
		i.state++
	case 4:
		i.pbByte = cpu.bus.ReadByte(uint32(i.pointerAddress + 2))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		cpu.r.PB = i.pbByte
		return true
	}
	return false
}

func (i *IDC) Reset() {
	i.state = 0
}

// I20 represents the CALL nnnn instruction
type I20 struct {
	state int

	lowByte  byte
	highByte byte

	pointerAddress uint16
}

// Step runs one cycle of the JMP instruction
// MLB active TODO
func (i *I20) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.highByte, i.lowByte = splitWord(cpu.r.PC - 1)
		cpu.PushByte(i.highByte)
		i.state++
	case 3:
		cpu.PushByte(i.lowByte)
		i.state++
	case 4:
		cpu.r.PC = i.pointerAddress
		return true
	}
	return false
}

func (i *I20) Reset() {
	i.state = 0
}

// I22 represents the CALL nnnnnn instruction
type I22 struct {
	state    int
	lowByte  byte
	highByte byte
	pbByte   byte
	address  uint16
}

// Step runs one cycle of the JMP instruction
// the emulation test case for this so 22.e.json seems to not wrap the stack pointer
// it also has a faulty test case on top of that. maybe im wrong but additional debug needed later TODO
// and im not even sure this is cycle accurate.
func (i *I22) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.address = createWord(i.highByte, i.lowByte)
		i.state++
	case 2:
		i.highByte, i.lowByte = splitWord(cpu.r.PC)
		cpu.PushByte(cpu.r.PB)
		i.state++
	case 3:
		i.state++
	case 4:
		i.pbByte = cpu.fetchByte()
		cpu.r.PC = i.address
		cpu.r.PB = i.pbByte
		i.state++
	case 5:
		cpu.PushByte(i.highByte)
		i.state++
	case 6:
		cpu.PushByte(i.lowByte)
		return true
	}
	return false
}

func (i *I22) Reset() {
	i.state = 0
}

// IFC represents the CALL [nnnn + X] instruction
type IFC struct {
	state int

	lowByte  byte
	highByte byte

	lowByteS byte

	pointerAddress uint16
}

func (i *IFC) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte, i.lowByteS = splitWord(cpu.r.PC)
		cpu.PushByte(i.highByte)
		i.state++
	case 2:
		cpu.PushByte(i.lowByteS)
		i.state++
	case 3:
		i.highByte = cpu.fetchByte()
		i.pointerAddress = createWord(i.highByte, i.lowByte)
		i.state++
	case 4:
		i.pointerAddress += cpu.r.GetX()
		i.state++
	case 5:
		i.lowByte = cpu.bus.ReadByte(cpu.mapAddressToBank(cpu.r.PB, i.pointerAddress))
		i.state++
	case 6:
		i.highByte = cpu.bus.ReadByte(cpu.mapAddressToBank(cpu.r.PB, i.pointerAddress+1))
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		return true
	}
	return false
}

func (i *IFC) Reset() {
	i.state = 0
}

// I40 represents the RTI or return from interrupt instruction
type I40 struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *I40) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		if cpu.r.E {
			i.lowByte |= 0x30 //m and x flags are always 1 in emulation mode
		}
		cpu.r.P = i.lowByte
		i.state++
	case 3:
		i.lowByte = cpu.PopByte()
		i.state++
	case 4:
		i.highByte = cpu.PopByte()
		cpu.r.PC = createWord(i.highByte, i.lowByte)
		if cpu.r.E {
			return true
		}
		i.state++
	case 5:
		cpu.r.PB = cpu.PopByte()
		return true
	}
	return false
}

func (i *I40) Reset() {
	i.state = 0
}

// I6B represents the RTL or return from subroutine long instruction
type I6B struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *I6B) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		i.state++
	case 3:
		i.highByte = cpu.PopByte()
		cpu.r.PC = createWord(i.highByte, i.lowByte) + 1
		i.state++
	case 4:
		cpu.r.PB = cpu.PopByte()
		return true
	}
	return false
}

func (i *I6B) Reset() {
	i.state = 0
}

// I60 represents the RTS or return from subroutine instruction
type I60 struct {
	state int

	lowByte  byte
	highByte byte
}

func (i *I60) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		i.state++
	case 3:
		i.highByte = cpu.PopByte()
		i.state++
	case 4:
		cpu.r.PC = createWord(i.highByte, i.lowByte) + 1
		return true
	}
	return false
}

func (i *I60) Reset() {
	i.state = 0
}

// I80 represents the BRA or branch always instruction
type I80 struct {
	state int

	pcTmp  uint16
	offset int8
}

func (i *I80) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offset = int8(cpu.fetchByte())
		i.state++
	case 1:
		i.pcTmp = cpu.r.PC
		cpu.r.PC += uint16(i.offset)
		if cpu.r.E && isPageBoundaryCrossed(i.pcTmp, cpu.r.PC) {
			i.state++
		} else {
			return true
		}
	case 2:
		return true
	}
	return false
}

func (i *I80) Reset() {
	i.state = 0
}

// I82 represents the BRL or branch always long instruction
type I82 struct {
	state int

	offset  int16
	offsetL byte
	offsetH byte
}

func (i *I82) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offsetL = cpu.fetchByte()
		i.state++
	case 1:
		i.offsetH = cpu.fetchByte()
		i.state++
	case 2:
		i.offset = int16(createWord(i.offsetH, i.offsetL))
		cpu.r.PC += uint16(i.offset)
		return true
	}
	return false
}

func (i *I82) Reset() {
	i.state = 0
}

// I10 represents the BPL or branch if positive instruction
type I10 struct {
	state int

	pcTmp  uint16
	offset int8
}

func (i *I10) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offset = int8(cpu.fetchByte())
		if cpu.r.hasFlag(FlagN) {
			return true
		}
		i.state++
	case 1:
		i.pcTmp = cpu.r.PC
		cpu.r.PC += uint16(i.offset)
		if cpu.r.E && isPageBoundaryCrossed(i.pcTmp, cpu.r.PC) {
			i.state++
		} else {
			return true
		}
	case 2:
		return true
	}
	return false
}

func (i *I10) Reset() {
	i.state = 0
}

// I30 represents the BMI or branch if not positive instruction
type I30 struct {
	state int

	pcTmp  uint16
	offset int8
}

func (i *I30) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.offset = int8(cpu.fetchByte())
		if !cpu.r.hasFlag(FlagN) {
			return true
		}
		i.state++
	case 1:
		i.pcTmp = cpu.r.PC
		cpu.r.PC += uint16(i.offset)
		if cpu.r.E && isPageBoundaryCrossed(i.pcTmp, cpu.r.PC) {
			i.state++
		} else {
			return true
		}
	case 2:
		return true
	}
	return false
}

func (i *I30) Reset() {
	i.state = 0
}
