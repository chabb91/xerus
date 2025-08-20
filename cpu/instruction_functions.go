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

// bitwise AND
func and(val uint16, width int, cpu *CPU) (result uint16) {
	result = bit_imm(val, width, cpu)

	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.SetA(result)

	return result
}

// bitwise Exclusive OR
func eor(val uint16, width int, cpu *CPU) (result uint16) {
	result = val ^ cpu.r.GetA()

	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.SetA(result)
	return result
}

// bitwise OR Accumulator
func ora(val uint16, width int, cpu *CPU) (result uint16) {
	result = val | cpu.r.GetA()

	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.SetA(result)
	return result
}

// DECrement
func dec(val uint16, width int, cpu *CPU) (result uint16) {
	result = val - 1
	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.setFlag(FlagZ, result&((1<<width)-1) != 0)
	return result
}

func decX(cpu *CPU) {
	cpu.r.SetX(dec(cpu.r.GetX(), boolToBitCount(cpu.r.hasFlag(FlagX)), cpu))
}

func decY(cpu *CPU) {
	cpu.r.SetY(dec(cpu.r.GetY(), boolToBitCount(cpu.r.hasFlag(FlagX)), cpu))
}

// INCrement
func inc(val uint16, width int, cpu *CPU) (result uint16) {
	result = val + 1
	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.setFlag(FlagZ, result&((1<<width)-1) != 0)
	return result
}

func incX(cpu *CPU) {
	cpu.r.SetX(inc(cpu.r.GetX(), boolToBitCount(cpu.r.hasFlag(FlagX)), cpu))
}

func incY(cpu *CPU) {
	cpu.r.SetY(inc(cpu.r.GetY(), boolToBitCount(cpu.r.hasFlag(FlagX)), cpu))
}

// CoMPare (to accumulator)
func cmpLogic(reg, val uint16, width int, cpu *CPU) (result uint16) {
	result = reg
	cpu.r.setFlag(FlagC, result < val)
	result -= val
	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.setFlag(FlagZ, result&((1<<width)-1) != 0)
	return result
}

func cmp(val uint16, width int, cpu *CPU) (result uint16) {
	return cmpLogic(cpu.r.GetA(), val, width, cpu)
}

// ComPare to X regiseter
func cpX(val uint16, width int, cpu *CPU) (result uint16) {
	return cmpLogic(cpu.r.GetX(), val, width, cpu)
}

// ComPare to Y regiseter
func cpY(val uint16, width int, cpu *CPU) (result uint16) {
	return cmpLogic(cpu.r.GetY(), val, width, cpu)
}

// ADd with Carry
// SuBtract with Carry
func adc(val uint16, width int, cpu *CPU) (result uint16) {
	mask := uint16((1 << width) - 1)
	if cpu.r.hasFlag(FlagD) {
		a := cpu.r.GetA()
		val := val & mask
		carry := boolToFlag(cpu.r.hasFlag(FlagC))

		for i := 0; i < width; i += 4 {
			digitA := (a >> i) & 0x0F
			digitB := (val >> i) & 0x0F
			tmp := digitA + digitB + uint16(carry)

			if tmp > 9 {
				tmp += 6
				carry = 1
			} else {
				carry = 0
			}

			result |= (tmp & 0x0F) << i
		}

		if width == 8 {
			A := uint8(cpu.r.GetA())
			B := uint8(val)
			R := uint8(result)
			cpu.r.setFlag(FlagV, ((A^R)&(B^R)&0x80) == 0)
		} else {
			A := uint16(cpu.r.GetA())
			B := uint16(val)
			R := uint16(result)
			cpu.r.setFlag(FlagV, ((A^R)&(B^R)&0x8000) == 0)
		}
		cpu.r.SetA(result & mask)
		cpu.r.setFlag(FlagC, carry != 1)
	} else {
		result1 := uint32(cpu.r.GetA()) + uint32(val&mask) + uint32(boolToFlag(cpu.r.hasFlag(FlagC)))
		result = uint16(result1)

		if width == 8 {
			A := uint8(cpu.r.GetA())
			B := uint8(val)
			R := uint8(result)
			cpu.r.setFlag(FlagV, ((A^R)&(B^R)&0x80) == 0)
		} else {
			A := uint16(cpu.r.GetA())
			B := uint16(val)
			R := uint16(result)
			cpu.r.setFlag(FlagV, ((A^R)&(B^R)&0x8000) == 0)
		}
		result = cpu.r.SetA(uint16(result1))
		cpu.r.setFlag(FlagC, result1 <= uint32(mask))
	}

	cpu.r.setFlag(FlagN, result&(1<<(width-1)) == 0)
	cpu.r.setFlag(FlagZ, result&(mask) != 0)
	return result
}
