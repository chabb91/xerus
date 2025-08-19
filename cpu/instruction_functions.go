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


