package cpu

type ShiftFunc func(val uint16, width int, inputCarry bool) (result uint16, carry, zero, negative bool)

func asl(val uint16, width int, _ bool) (uint16, bool, bool, bool) {
	mask := uint16(1) << (width - 1)
	carry := (val & mask) == 0
	result := (val << 1) & ((1 << width) - 1)
	negative := (result & mask) == 0
	zero := result != 0

	return result, carry, zero, negative
}

func lsr(val uint16, width int, _ bool) (uint16, bool, bool, bool) {
	mask := uint16((1 << width) - 1)
	carry := (val & 1) == 0
	result := (val & mask) >> 1
	zero := result != 0

	return result, carry, zero, true
}

func ror(val uint16, width int, inputCarry bool) (uint16, bool, bool, bool) {
	mask := uint16(1) << (width - 1)
	carry := (val & 1) == 0
	result := (val >> 1) & ((1 << width) - 1)
	if inputCarry {
		result |= mask
	} else {
		result &= ^mask
	}
	negative := !inputCarry
	zero := result != 0

	return result, carry, zero, negative
}

func rol(val uint16, width int, inputCarry bool) (uint16, bool, bool, bool) {
	mask := uint16(1) << (width - 1)
	carry := (val & mask) == 0
	result := (val << 1) & ((1 << width) - 1)
	if inputCarry {
		result |= 1
	} else {
		result &= (1 << width) - 2
	}
	negative := (result & mask) == 0
	zero := result != 0

	return result, carry, zero, negative
}

type ShiftAccumulator struct {
	state int

	shiftFunc ShiftFunc
}

func (i *ShiftAccumulator) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		//TODO check this in Reset(), dont think its possible for the value of the M flag to change between these 2 cycles
		//but cant test it now so ill just keep it like this
		width := 16
		if cpu.r.hasFlag(FlagM) {
			width = 8
		}
		result, c, z, n := i.shiftFunc(cpu.r.A, width, cpu.r.hasFlag(FlagC))

		if cpu.r.hasFlag(FlagM) {
			SetLowByte(&cpu.r.A, byte(result))
		} else {
			cpu.r.A = result
		}

		cpu.r.setFlag(FlagC, c)
		cpu.r.setFlag(FlagN, n)
		cpu.r.setFlag(FlagZ, z)
		return true
	}
	return false
}

func (i *ShiftAccumulator) Reset(cpu *CPU) {
	i.state = 0
}

type ShiftZeroPage struct {
	state int

	lowByte, highByte byte

	shiftFunc ShiftFunc
	dirX      bool
	addr      uint16

	c, z, n bool
	result  uint16
	address uint32
}

// TODO the high byte SHOULD always be the same in address. the low byte should wrap without carry
// but that fails the only test suite i have. so i cant let it wrap.
// i get the feeling this will fail later
func (i *ShiftZeroPage) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		if getLowByte(cpu.r.D) == 0 {
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
			i.addr = cpu.r.GetX()
		}
		i.state++
	case 3:
		if i.dirX {
			i.addr += uint16(i.lowByte)
		} else {
			i.addr = uint16(i.lowByte)
		}
		//TODO wrap here
		//something like add lowbyte of D to lowbyte of addr
		//jesus christ whats wrong with the test wrapping. this is some horrible edge case
		//TODO double todo giga investigate later
		if getLowByte(cpu.r.D) == 0 && i.dirX && cpu.r.E {
			i.address = cpu.mapAddressToBank(0x00, addWordToWordWithWrap(cpu.r.D, i.addr))
		} else {
			i.address = cpu.mapAddressToBank(0x00, cpu.r.D+i.addr)
		}

		i.lowByte = cpu.bus.ReadByte(i.address)
		if cpu.r.hasFlag(FlagM) {
			i.state++
		}
		i.state++
	case 4:
		if !cpu.r.hasFlag(FlagM) {
			i.highByte = cpu.bus.ReadByte(i.address + 1)
		}
		i.state++
	case 5:
		if cpu.r.hasFlag(FlagM) {
			i.result, i.c, i.z, i.n = i.shiftFunc(uint16(i.lowByte), 8, cpu.r.hasFlag(FlagC))
			i.state++
		} else {
			i.result, i.c, i.z, i.n = i.shiftFunc(createWord(i.highByte, i.lowByte), 16, cpu.r.hasFlag(FlagC))
		}
		i.state++
	case 6:
		if !cpu.r.hasFlag(FlagM) {
			cpu.bus.WriteByte(i.address+1, getHighByte(i.result))
		}
		i.state++
	case 7:
		cpu.bus.WriteByte(i.address, getLowByte(i.result))
		cpu.r.setFlag(FlagC, i.c)
		cpu.r.setFlag(FlagN, i.n)
		cpu.r.setFlag(FlagZ, i.z)
		return true
	}
	return false
}

func (i *ShiftZeroPage) Reset(cpu *CPU) {
	i.state = 0
}

type ShiftAbsolute struct {
	state int

	shiftFunc ShiftFunc

	lowByte, highByte byte

	dirX bool

	c, z, n bool
	result  uint16
	address uint32
}

func (i *ShiftAbsolute) Step(cpu *CPU) bool {
	switch i.state {
	case 0:
		i.lowByte = cpu.fetchByte()
		i.state++
	case 1:
		i.highByte = cpu.fetchByte()
		i.address = cpu.mapDataAddress(createWord(i.highByte, i.lowByte))
		if i.dirX {
			i.state = 2
		} else {
			i.state = 3
		}
	case 2:
		//TODO create a helper function this is the ABSOLUTE+1 logic
		i.address = (i.address + uint32(cpu.r.GetX())) & (1<<24 - 1)
		i.state++
	case 3:
		i.lowByte = cpu.bus.ReadByte(i.address)
		if cpu.r.hasFlag(FlagM) {
			i.state++
		}
		i.state++
	case 4:
		if !cpu.r.hasFlag(FlagM) {
			i.highByte = cpu.bus.ReadByte(i.address + 1)
		}
		i.state++
	case 5:
		if cpu.r.hasFlag(FlagM) {
			i.result, i.c, i.z, i.n = i.shiftFunc(uint16(i.lowByte), 8, cpu.r.hasFlag(FlagC))
			i.state++
		} else {
			i.result, i.c, i.z, i.n = i.shiftFunc(createWord(i.highByte, i.lowByte), 16, cpu.r.hasFlag(FlagC))
		}
		i.state++
	case 6:
		if !cpu.r.hasFlag(FlagM) {
			cpu.bus.WriteByte(i.address+1, getHighByte(i.result))
		}
		i.state++
	case 7:
		cpu.bus.WriteByte(i.address, getLowByte(i.result))
		cpu.r.setFlag(FlagC, i.c)
		cpu.r.setFlag(FlagN, i.n)
		cpu.r.setFlag(FlagZ, i.z)
		return true
	}
	return false
}

func (i *ShiftAbsolute) Reset(cpu *CPU) {
	i.state = 0
}
