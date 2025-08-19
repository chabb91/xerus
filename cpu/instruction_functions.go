package cpu

// here are the core functions for all the basic instructions (so non implied or out of the ordinary)
// that can be used separately from the addressing modes.

// the functuion type that all these insctructions can hopefully use
type instructionFuncWith16BitReturn func(val uint16, width int, cpu *CPU) (result uint16)

// LoaD Accumulator/X/Y
func sta(_ uint16, _ int, cpu *CPU) uint16 {
	return cpu.r.GetA()
}

func stx(_ uint16, _ int, cpu *CPU) uint16 {
	return cpu.r.GetX()
}

func sty(_ uint16, _ int, cpu *CPU) uint16 {
	return cpu.r.GetY()
}

func stz(_ uint16, _ int, _ *CPU) uint16 {
	return 0
}

// STore Accumulator/X/Y/Zero
func lda(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, (1<<(width-1))&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetA(result)
	return result
}

func ldx(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, (1<<(width-1))&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetX(result)
	return result
}

func ldy(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, (1<<(width-1))&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetY(result)
	return result
}

// Test and Reset/Set Bits
func trb(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	for v := range width {
		if (cpu.r.A>>v)&1 == 1 {
			result &= ^(1 << v)
		}
	}
	cpu.r.setFlag(FlagZ, (val&cpu.r.A) != 0)
	return result
}

func tsb(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	for v := range width {
		if (cpu.r.A>>v)&1 == 1 {
			result |= 1 << v
		}
	}
	cpu.r.setFlag(FlagZ, (val&cpu.r.A) != 0)
	return result
}

// Arithmetic Shift Left
func asl(val uint16, width int, cpu *CPU) uint16 {
	mask := uint16(1) << (width - 1)
	result := (val << 1) & ((1 << width) - 1)

	cpu.r.setFlag(FlagC, (val&mask) == 0)
	cpu.r.setFlag(FlagN, (result&mask) == 0)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

// Logical Shift Right
func lsr(val uint16, width int, cpu *CPU) uint16 {
	mask := uint16((1 << width) - 1)
	result := (val & mask) >> 1

	cpu.r.setFlag(FlagC, (val&1) == 0)
	cpu.r.setFlag(FlagN, true)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

// Rotate Right
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

// Rotate Left
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

// test BITs
// the only instruction that behaves differently based on addressing mode
func bit_imm(val uint16, _ int, cpu *CPU) (result uint16) {
	result = val & cpu.r.GetA()

	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

func bit(val uint16, width int, cpu *CPU) (result uint16) {
	cpu.r.setFlag(FlagN, val&(1<<(width-1)) == 0)
	cpu.r.setFlag(FlagV, val&(1<<(width-2)) == 0)

	return bit_imm(val, width, cpu)
}
