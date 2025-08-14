package cpu

func asl(val uint16, width int, cpu *CPU) uint16 {
	mask := uint16(1) << (width - 1)
	result := (val << 1) & ((1 << width) - 1)

	cpu.r.setFlag(FlagC, (val&mask) == 0)
	cpu.r.setFlag(FlagN, (result&mask) == 0)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

func lsr(val uint16, width int, cpu *CPU) uint16 {
	mask := uint16((1 << width) - 1)
	result := (val & mask) >> 1

	cpu.r.setFlag(FlagC, (val&1) == 0)
	cpu.r.setFlag(FlagN, true)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

func ror(val uint16, width int, cpu *CPU) uint16 {
	inputCarry := cpu.r.hasFlag(FlagC)
	mask := uint16(1) << (width - 1)

	result := (val >> 1) & ((1 << width) - 1)
	if inputCarry {
		result |= mask
	} else {
		result &= ^mask
	}

	cpu.r.setFlag(FlagC, (val&1) == 0)
	cpu.r.setFlag(FlagN, !inputCarry)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

func rol(val uint16, width int, cpu *CPU) uint16 {
	inputCarry := cpu.r.hasFlag(FlagC)
	mask := uint16(1) << (width - 1)

	result := (val << 1) & ((1 << width) - 1)
	if inputCarry {
		result |= 1
	} else {
		result &= (1 << width) - 2
	}

	cpu.r.setFlag(FlagC, (val&mask) == 0)
	cpu.r.setFlag(FlagN, (result&mask) == 0)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}
