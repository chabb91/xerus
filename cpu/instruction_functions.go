package cpu

// here are the core functions for all the basic instructions (so non implied or out of the ordinary)
// that can be used separately from the addressing modes.

type iWidth int

func (w iWidth) msbMask() uint16 {
	return 1 << (w - 1)
}

func (w iWidth) maxValue() uint16 {
	return (1 << w) - 1
}

const W_8 iWidth = 8
const W_16 iWidth = 16

// the functuion type that all these insctructions can hopefully use
type instructionFuncWith16BitReturn func(val uint16, w iWidth, cpu *CPU) (result uint16)

// LoaD Accumulator/X/Y
func sta(_ uint16, _ iWidth, cpu *CPU) uint16 {
	return cpu.r.GetA()
}

func stx(_ uint16, _ iWidth, cpu *CPU) uint16 {
	return cpu.r.GetX()
}

func sty(_ uint16, _ iWidth, cpu *CPU) uint16 {
	return cpu.r.GetY()
}

func stz(_ uint16, _ iWidth, _ *CPU) uint16 {
	return 0
}

// STore Accumulator/X/Y/Zero
func lda(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, w.msbMask()&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetA(result)
	return result
}

func ldx(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, w.msbMask()&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetX(result)
	return result
}

func ldy(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, w.msbMask()&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetY(result)
	return result
}

// Test and Reset/Set Bits
func trb(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = (cpu.r.A) & w.maxValue()
	result = val & ^result
	cpu.r.setFlag(FlagZ, (val&cpu.r.A) != 0)
	return
}

func tsb(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = (cpu.r.A) & w.maxValue()
	result = val | result
	cpu.r.setFlag(FlagZ, (val&cpu.r.A) != 0)
	return
}

// Arithmetic Shift Left
func asl(val uint16, w iWidth, cpu *CPU) uint16 {
	mask := w.msbMask()
	result := (val << 1) & w.maxValue()

	cpu.r.setFlag(FlagC, (val&mask) == 0)
	cpu.r.setFlag(FlagN, (result&mask) == 0)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

// Logical Shift Right
func lsr(val uint16, w iWidth, cpu *CPU) uint16 {
	result := (val & w.maxValue()) >> 1

	cpu.r.setFlag(FlagC, (val&1) == 0)
	cpu.r.setFlag(FlagN, true)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

// Rotate Right
func ror(val uint16, w iWidth, cpu *CPU) uint16 {
	inputCarry := cpu.r.hasFlag(FlagC)

	result := (val & w.maxValue()) >> 1
	if inputCarry {
		result |= w.msbMask()
	}
	cpu.r.setFlag(FlagC, (val&1) == 0)
	cpu.r.setFlag(FlagN, !inputCarry)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

// Rotate Left
func rol(val uint16, w iWidth, cpu *CPU) uint16 {
	inputCarry := cpu.r.hasFlag(FlagC)
	mask := w.msbMask()

	result := (val << 1) & w.maxValue()
	if inputCarry {
		result |= 1
	}
	cpu.r.setFlag(FlagC, (val&mask) == 0)
	cpu.r.setFlag(FlagN, (result&mask) == 0)
	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

// test BITs
// the only instruction that behaves differently based on addressing mode
func bit_imm(val uint16, _ iWidth, cpu *CPU) (result uint16) {
	result = val & cpu.r.GetA()

	cpu.r.setFlag(FlagZ, result != 0)

	return result
}

func bit(val uint16, w iWidth, cpu *CPU) (result uint16) {
	cpu.r.setFlag(FlagN, val&w.msbMask() == 0)
	cpu.r.setFlag(FlagV, val&(w.msbMask()>>1) == 0)

	return bit_imm(val, w, cpu)
}

// bitwise AND
func and(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = bit_imm(val, w, cpu)

	cpu.r.setFlag(FlagN, result&w.msbMask() == 0)
	cpu.r.SetA(result)

	return result
}

// bitwise Exclusive OR
func eor(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val ^ cpu.r.GetA()

	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.setFlag(FlagN, result&w.msbMask() == 0)
	cpu.r.SetA(result)
	return result
}

// bitwise OR Accumulator
func ora(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val | cpu.r.GetA()

	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.setFlag(FlagN, result&w.msbMask() == 0)
	cpu.r.SetA(result)
	return result
}

// DECrement
func dec(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val - 1
	cpu.r.setFlag(FlagN, result&w.msbMask() == 0)
	cpu.r.setFlag(FlagZ, result&w.maxValue() != 0)
	return result
}

func decX(cpu *CPU) {
	cpu.r.SetX(dec(cpu.r.GetX(), getInstructionWidth(cpu.r.hasFlag(FlagX)), cpu))
}

func decY(cpu *CPU) {
	cpu.r.SetY(dec(cpu.r.GetY(), getInstructionWidth(cpu.r.hasFlag(FlagX)), cpu))
}

// INCrement
func inc(val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = val + 1
	cpu.r.setFlag(FlagN, result&w.msbMask() == 0)
	cpu.r.setFlag(FlagZ, result&w.maxValue() != 0)
	return result
}

func incX(cpu *CPU) {
	cpu.r.SetX(inc(cpu.r.GetX(), getInstructionWidth(cpu.r.hasFlag(FlagX)), cpu))
}

func incY(cpu *CPU) {
	cpu.r.SetY(inc(cpu.r.GetY(), getInstructionWidth(cpu.r.hasFlag(FlagX)), cpu))
}

// CoMPare (to accumulator)
func cmpLogic(reg, val uint16, w iWidth, cpu *CPU) (result uint16) {
	result = reg
	cpu.r.setFlag(FlagC, result < val)
	result -= val
	cpu.r.setFlag(FlagN, result&w.msbMask() == 0)
	cpu.r.setFlag(FlagZ, result&w.maxValue() != 0)
	return result
}

func cmp(val uint16, w iWidth, cpu *CPU) (result uint16) {
	return cmpLogic(cpu.r.GetA(), val, w, cpu)
}

// ComPare to X register
func cpX(val uint16, w iWidth, cpu *CPU) (result uint16) {
	return cmpLogic(cpu.r.GetX(), val, w, cpu)
}

// ComPare to Y register
func cpY(val uint16, w iWidth, cpu *CPU) (result uint16) {
	return cmpLogic(cpu.r.GetY(), val, w, cpu)
}

// ADd with Carry
func adc(val uint16, w iWidth, cpu *CPU) (result uint16) {
	mask1 := w.maxValue()
	mask2 := w.msbMask()
	a := cpu.r.GetA()
	c := min(cpu.r.P&FlagC, 1)

	if cpu.r.hasFlag(FlagD) {
		width := int(w)
		for i := 0; i < width; i += 4 {
			tmp := (a>>i)&0xF + (val>>i)&0xF + uint16(c)
			result = (result &^ (0xF << i)) | ((tmp & 0xF) << i)

			if width == i+4 {
				cpu.r.setFlag(FlagV, ((a^result)&(val^result)&mask2) == 0)
			}

			if tmp > 9 {
				result += 6 << i
				c = 1
			} else {
				c = 0
			}
		}

		result = cpu.r.SetA(result & mask1)
		cpu.r.setFlag(FlagC, c != 1)
	} else {
		result32 := uint32(a) + uint32(val&mask1) + uint32(c)
		result = cpu.r.SetA(uint16(result32))

		cpu.r.setFlag(FlagV, ((a^result)&(val^result)&mask2) == 0)
		cpu.r.setFlag(FlagC, result32 <= uint32(mask1))
	}

	cpu.r.setFlag(FlagN, result&(mask2) == 0)
	cpu.r.setFlag(FlagZ, result&(mask1) != 0)
	return result
}

// SuBtract with Carry
// carry flag is set on underflow
// overflow is calculated differently
func sbc(val uint16, w iWidth, cpu *CPU) (result uint16) {
	mask1 := w.maxValue()
	mask2 := w.msbMask()
	a := cpu.r.GetA()
	c := min(cpu.r.P&FlagC, 1)

	if cpu.r.hasFlag(FlagD) {
		width := int(w)
		for i := 0; i < width; i += 4 {
			tmp := int16((a>>i)&0xF) - int16((val>>i)&0xF) - 1 + int16(c)
			result = (result &^ (0xF << i)) | (uint16(tmp&0xF) << i)

			if width == i+4 {
				cpu.r.setFlag(FlagV, ((a^val)&(a^result)&mask2) == 0)
			}

			if tmp < 0 {
				result -= 6 << i
				c = 0
			} else {
				c = 1
			}
		}

		result = cpu.r.SetA(result & mask1)
		cpu.r.setFlag(FlagC, c != 1)
	} else {
		result32 := uint32(a) - uint32(val&mask1) - 1 + uint32(c)
		result = cpu.r.SetA(uint16(result32))

		cpu.r.setFlag(FlagV, ((a^val)&(a^result)&mask2) == 0)
		cpu.r.setFlag(FlagC, result32 > uint32(mask1))
	}

	cpu.r.setFlag(FlagN, result&(mask2) == 0)
	cpu.r.setFlag(FlagZ, result&(mask1) != 0)
	return result
}

func transferFlagHelper(hasFlag bool, register uint16, cpu *CPU) {
	if hasFlag {
		cpu.r.setFlag(FlagN, register&0x80 == 0)
		cpu.r.setFlag(FlagZ, register&0xFF != 0)
	} else {
		cpu.r.setFlag(FlagN, register&0x8000 == 0)
		cpu.r.setFlag(FlagZ, register != 0)
	}
}

// the "logic" for pei and pea. its a bit slow because umbrella creates a word for no reason that gets split right after.
func peAI(val uint16, _ iWidth, _ *CPU) uint16 {
	return val
}

// Push Effective Relative address
func per(val uint16, _ iWidth, cpu *CPU) uint16 {
	return cpu.r.PC + rel16(val)
}

// Transfer Accumulator to X register
func tax(cpu *CPU) {
	cpu.r.SetX(cpu.r.A)
	transferFlagHelper(cpu.r.hasFlag(FlagX), cpu.r.GetX(), cpu)
}

// Transfer Accumulator to Y register
func tay(cpu *CPU) {
	cpu.r.SetY(cpu.r.A)
	transferFlagHelper(cpu.r.hasFlag(FlagX), cpu.r.GetY(), cpu)
}

// Transfer Stack to X register
func tsx(cpu *CPU) {
	cpu.r.SetX(cpu.r.GetStack())
	transferFlagHelper(cpu.r.hasFlag(FlagX), cpu.r.GetX(), cpu)
}

// Transfer Accumulator to Y register
func txa(cpu *CPU) {
	cpu.r.SetA(cpu.r.GetX())
	transferFlagHelper(cpu.r.hasFlag(FlagM), cpu.r.GetA(), cpu)
}

// Transfer X register to Stack
func txs(cpu *CPU) {
	cpu.r.SetStack(cpu.r.GetX())
}

// Transfer X register to Y register
func txy(cpu *CPU) {
	cpu.r.SetY(cpu.r.GetX())
	transferFlagHelper(cpu.r.hasFlag(FlagX), cpu.r.GetY(), cpu)
}

// Transfer Y register to Accumulator
func tya(cpu *CPU) {
	cpu.r.SetA(cpu.r.GetY())
	transferFlagHelper(cpu.r.hasFlag(FlagM), cpu.r.GetA(), cpu)
}

// Transfer Y register to Accumulator
func tyx(cpu *CPU) {
	cpu.r.SetX(cpu.r.GetY())
	transferFlagHelper(cpu.r.hasFlag(FlagX), cpu.r.GetX(), cpu)
}

// Transfer 16-bit Accumulator (C) to Direct register
func tcd(cpu *CPU) {
	cpu.r.D = cpu.r.A
	transferFlagHelper(false, cpu.r.D, cpu)
}

// Transfer 16-bit Accumulator (C) Stack pointer
func tcs(cpu *CPU) {
	cpu.r.SetStack(cpu.r.A)
}

// Transfer Direct register to 16-bit Accumulator (C)
func tdc(cpu *CPU) {
	cpu.r.A = cpu.r.D
	transferFlagHelper(false, cpu.r.A, cpu)
}

// Transfer Stack pointer to 16-bit Accumulator (C)
func tsc(cpu *CPU) {
	cpu.r.A = cpu.r.GetStack()
	transferFlagHelper(false, cpu.r.A, cpu)
}

// the XCE or eXchange Carry and Emulation instruction
// the only instruction that can swap modes
func xce(cpu *CPU) {
	tmp := cpu.r.hasFlag(FlagC)
	cpu.r.setFlag(FlagC, !cpu.r.E)
	cpu.r.E = tmp

	if cpu.r.E {
		cpu.r.P |= 0x30
		cpu.r.X &= 0xFF
		cpu.r.Y &= 0xFF
		cpu.r.S = 0x0100 | (cpu.r.S & 0xFF)
	}
}
