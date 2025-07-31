package cpu

const (
	FlagC byte = 1 << 0 // Carry                   (0=No Carry, 1=Carry)
	FlagZ byte = 1 << 1 // Zero                    (0=Nonzero, 1=Zero)
	FlagI byte = 1 << 2 // IRQ Disable             (0=IRQ Enable, 1=IRQ Disable)
	FlagD byte = 1 << 3 // Decimal Mode            (0=Normal, 1=BCD Mode for ADC/SBC opcodes)
	FlagX byte = 1 << 4 // Index Register Size     (e: 0=IRQ/NMI, 1=BRK/PHP opcode)(0=Index registers 16 bit, 1=Index registers 8 bit)
	FlagM byte = 1 << 5 // Memory/Accumulator Size (e: Always 1) (0=A is 16 bit, 1=A is 8 bit)
	FlagV byte = 1 << 6 // Overflow                (0=No Overflow, 1=Overflow)
	FlagN byte = 1 << 7 // Negative                (0=Positive, 1=Negative)
)

type registers struct {
	A  uint16 //accumulator
	X  uint16 //index
	PC uint16 //program counter
	S  uint16 //stack pointer
	P  byte   //processor status register, holds the first 8 flags using masks
	D  uint16 //zeropage offset      ;expands 8bit  [nn]   to 16bit [00:nn+D]
	DB byte   //Data Bank            ;expands 16bit [nnnn] to 24bit [DB:nnnn]
	PB byte   //Program Counter Bank ;expands 16bit PC     to 24bit PB:PC
}

func (c *registers) setFlag(flag byte, reset bool) {
	if !reset {
		c.P |= flag
	} else {
		c.P &= ^flag
	}
}

func (c *registers) hasFlag(flag byte) bool {
	return (c.P & flag) != 0
}

// SetLowByte takes a 16-bit value and an 8-bit value,
// and returns a new 16-bit value with the low byte updated.
// The high byte of the original value is preserved.
func SetLowByte(original uint16, newLowByte byte) uint16 {
	return (original & 0xFF00) | uint16(newLowByte)
}

// SetHighByte takes a 16-bit value and an 8-bit value,
// and returns a new 16-bit value with the high byte updated.
// The low byte of the original value is preserved.
func SetHighByte(original uint16, newHighByte byte) uint16 {
	return (original & 0x00FF) | (uint16(newHighByte) << 8)
}

// high byte=AB of ABCD
func getHighByte(fullValue uint16) byte {
	return byte((0xFF00 & fullValue) >> 8)

}

// low byte=CD of ABCD
func getLowByte(fullValue uint16) byte {
	return byte(0x00FF & fullValue)
}
