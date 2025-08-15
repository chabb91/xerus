package cpu

type instructionFuncWith16BitReturn func(val uint16, width int, cpu *CPU) (result uint16)

// this template represents the Abs and AbsX instructions in (8-2*m and 9-2*m)
// modes respectively
// set absX true for absX mode
type AbsAbsXRW struct {
	state int

	instructionFunc instructionFuncWith16BitReturn

	lowByte, highByte    byte
	addressLo, addressHi uint32

	absX bool

	register uint16

	result uint16
}

func (i *AbsAbsXRW) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		if i.absX {
			i.state = 2
		} else {
			i.state = 3
		}
	case 2:
		i.register = cpu.r.GetX()
		i.state++
	case 3:
		if !i.absX {
			i.addressLo, i.addressHi = absolute(cpu.r.DB, i.highByte, i.lowByte)
		} else {
			i.addressLo, i.addressHi = absoluteXY(cpu.r.DB, i.highByte, i.lowByte, i.register)
		}

		i.lowByte = cpu.bus.ReadByte(i.addressLo)
		if cpu.r.hasFlag(FlagM) {
			i.state = 5
		} else {
			i.state = 4
		}
	case 4:
		if !cpu.r.hasFlag(FlagM) {
			i.highByte = cpu.bus.ReadByte(i.addressHi)
		}
		i.state++
	case 5:
		if cpu.r.hasFlag(FlagM) {
			i.result = i.instructionFunc(uint16(i.lowByte), 8, cpu)
			i.state = 7
		} else {
			i.result = i.instructionFunc(createWord(i.highByte, i.lowByte), 16, cpu)
			i.state = 6
		}
	case 6:
		if !cpu.r.hasFlag(FlagM) {
			cpu.bus.WriteByte(i.addressHi, getHighByte(i.result))
		}
		i.state++
	case 7:
		cpu.bus.WriteByte(i.addressLo, getLowByte(i.result))
		return true
	}
	return false
}

func (i *AbsAbsXRW) Reset(cpu *CPU) {
	i.state = 0
}

// this template represents the Dir and Dir,X instructions in (7-2*m+w and 8-2*m+w)
// modes respectively
// set dirX true for dirX mode
type DirDirXRW struct {
	state int

	instructionFunc instructionFuncWith16BitReturn

	lowByte, highByte    byte
	addressLo, addressHi uint32

	dirX bool

	register uint16

	result uint16
}

func (i *DirDirXRW) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		if cpu.isW() {
			i.state++
			if !i.dirX {
				i.state++
			}
		}
		i.state++
	case 1:
		i.state++
		if !i.dirX {
			i.state++
		}
	case 2:
		if i.dirX {
			i.register = cpu.r.GetX()
		}
		i.state++
	case 3:
		if i.dirX {
			i.addressLo, i.addressHi = directPageXY(cpu, i.lowByte, i.register)
		} else {
			i.addressLo, i.addressHi = directPage(cpu, i.lowByte, false)
		}
		i.lowByte = cpu.bus.ReadByte(i.addressLo)
		if cpu.r.hasFlag(FlagM) {
			i.state = 5
		} else {
			i.state = 4
		}
	case 4:
		if !cpu.r.hasFlag(FlagM) {
			i.highByte = cpu.bus.ReadByte(i.addressHi)
		}
		i.state++
	case 5:
		if cpu.r.hasFlag(FlagM) {
			i.result = i.instructionFunc(uint16(i.lowByte), 8, cpu)
			i.state = 7
		} else {
			i.result = i.instructionFunc(createWord(i.highByte, i.lowByte), 16, cpu)
			i.state = 6
		}
	case 6:
		if !cpu.r.hasFlag(FlagM) {
			cpu.bus.WriteByte(i.addressHi, getHighByte(i.result))
		}
		i.state++
	case 7:
		cpu.bus.WriteByte(i.addressLo, getLowByte(i.result))
		return true
	}
	return false
}

func (i *DirDirXRW) Reset(cpu *CPU) {
	i.state = 0
}

type Accumulator struct {
	state  int
	result uint16

	instructionFunc instructionFuncWith16BitReturn
}

func (i *Accumulator) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		//TODO check this in Reset(), dont think its possible for the value of the M flag to change between these 2 cycles
		//but cant test it now so ill just keep it like this
		width := 16
		if cpu.r.hasFlag(FlagM) {
			width = 8
		}
		i.result = i.instructionFunc(cpu.r.A, width, cpu)

		if cpu.r.hasFlag(FlagM) {
			SetLowByte(&cpu.r.A, byte(i.result))
		} else {
			cpu.r.A = i.result
		}
		return true
	}
	return false
}

func (i *Accumulator) Reset(cpu *CPU) {
	i.state = 0
}
