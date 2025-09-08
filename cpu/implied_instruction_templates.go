package cpu

// accumulator address mode is unique enough to not be part of umbrella
type Accumulator struct {
	result uint16

	instructionFunc instructionFuncWith16BitReturn
}

func (i *Accumulator) Step(cpu *CPU) bool {
	width := 16
	if cpu.r.hasFlag(FlagM) {
		width = 8
	}
	i.result = i.instructionFunc(cpu.r.A, width, cpu)

	cpu.r.SetA(i.result)
	return true
}
func (i *Accumulator) Reset(cpu *CPU) {
}

// 2 cycle implied struct. there are many instructions that just fetch opcode and execute a function.
type TwoCycleImplied struct {
	instructionFunc func(cpu *CPU)
}

func (i *TwoCycleImplied) Step(cpu *CPU) bool {
	i.instructionFunc(cpu)

	return true
}
func (i *TwoCycleImplied) Reset(cpu *CPU) {
}

// stack Push/Pull implied instructions
// PulL Direct register
type Ipld struct {
	state    int
	lowByte  byte
	highByte byte
}

func (i *Ipld) Step(cpu *CPU) bool {
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
		cpu.r.D = createWord(i.highByte, i.lowByte)
		cpu.r.setFlag(FlagN, cpu.r.D&0x8000 == 0)
		cpu.r.setFlag(FlagZ, cpu.r.D != 0)
		return true
	}
	return false
}

func (i *Ipld) Reset(cpu *CPU) {
	i.state = 0
}

// PulL Processor status register
type Iplp struct {
	state int
}

func (i *Iplp) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		cpu.r.P = cpu.PopByte()
		if cpu.r.E {
			cpu.r.P |= 0x30
		}
		return true
	}
	return false
}

func (i *Iplp) Reset(cpu *CPU) {
	i.state = 0
}

// PulL data Bank register
type Iplb struct {
	state int
}

func (i *Iplb) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		cpu.r.DB = cpu.PopByte()
		cpu.r.setFlag(FlagN, cpu.r.DB&0x80 == 0)
		cpu.r.setFlag(FlagZ, cpu.r.DB != 0)
		return true
	}
	return false
}

func (i *Iplb) Reset(cpu *CPU) {
	i.state = 0
}

// PusH data Bank register
type Iphb struct {
	state int
}

func (i *Iphb) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		cpu.PushByte(cpu.r.DB)
		return true
	}
	return false
}

func (i *Iphb) Reset(cpu *CPU) {
	i.state = 0
}

// PusH Direct register
type Iphd struct {
	state             int
	lowByte, highByte byte
}

func (i *Iphd) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.highByte, i.lowByte = splitWord(cpu.r.D)
		cpu.PushByte(i.highByte)
		i.state++
	case 2:
		cpu.PushByte(i.lowByte)
		return true
	}
	return false
}

func (i *Iphd) Reset(cpu *CPU) {
	i.state = 0
}

// PusH K(PB) register
type Iphk struct {
	state int
}

func (i *Iphk) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		cpu.PushByte(cpu.r.PB)
		return true
	}
	return false
}

func (i *Iphk) Reset(cpu *CPU) {
	i.state = 0
}

// PusH Processor status register
type Iphp struct {
	state int
}

func (i *Iphp) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		cpu.PushByte(cpu.r.P)
		return true
	}
	return false
}

func (i *Iphp) Reset(cpu *CPU) {
	i.state = 0
}

// PusH Accumulator/X register/ Y register
type PushAXY struct {
	flag     byte
	register func(*CPU) uint16

	state int
}

func (i *PushAXY) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
		if cpu.r.hasFlag(i.flag) {
			i.state++
		}
	case 1:
		cpu.PushByte(getHighByte(i.register(cpu)))
		i.state++
	case 2:
		cpu.PushByte(getLowByte(i.register(cpu)))
		return true
	}
	return false
}

func (i *PushAXY) Reset(cpu *CPU) {
	i.state = 0
}

// PulL Accumulator/X register/ Y register
type PullAXY struct {
	flag     byte
	register func(uint16, *CPU) uint16

	state int

	lowByte byte
	address uint16
}

func (i *PullAXY) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.state++
	case 1:
		i.state++
	case 2:
		i.lowByte = cpu.PopByte()
		if cpu.r.hasFlag(i.flag) {
			cpu.r.setFlag(FlagN, i.lowByte&0x80 == 0)
			cpu.r.setFlag(FlagZ, i.lowByte != 0)
			i.register(uint16(i.lowByte), cpu)
			return true
		}
		i.state++
	case 3:
		i.address = createWord(cpu.PopByte(), i.lowByte)
		cpu.r.setFlag(FlagN, i.address&0x8000 == 0)
		cpu.r.setFlag(FlagZ, i.address != 0)
		i.register(i.address, cpu)
		return true
	}
	return false
}

func (i *PullAXY) Reset(cpu *CPU) {
	i.state = 0
}
