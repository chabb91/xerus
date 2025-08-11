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
