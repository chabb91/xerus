package apu

const (
	FlagC byte = 1 << 0 // Carry
	FlagZ byte = 1 << 1 // Zero
	FlagI byte = 1 << 2 // Interrupt  (unused??)
	FlagH byte = 1 << 3 // Half-carry
	FlagB byte = 1 << 4 // Break  (unused??)
	FlagP byte = 1 << 5 // Direct Page selector
	FlagV byte = 1 << 6 // Overflow
	FlagN byte = 1 << 7 // Negative
)

type registers struct {
	A   byte   //accumulator
	X   byte   //index
	Y   byte   //index
	SP  byte   //stack pointer
	PSW byte   //program status word
	PC  uint16 //program counter
}

func (r *registers) setFlag(flag byte, reset bool) {
	if !reset {
		r.PSW |= flag
	} else {
		r.PSW &= ^flag
	}
}

func (r *registers) hasFlag(flag byte) bool {
	return (r.PSW & flag) != 0
}

func (r *registers) getDirectPageNum() byte {
	return (r.PSW & FlagP) >> 5
}

func (r *registers) getStackAddr() uint16 {
	return 0x100 | uint16(r.SP)
}
