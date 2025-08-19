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
