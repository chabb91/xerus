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
	Y  uint16 //index
	PC uint16 //program counter
	S  uint16 //stack pointer
	P  byte   //processor status register, holds the first 8 flags using masks
	D  uint16 //zeropage offset      ;expands 8bit  [nn]   to 16bit [00:nn+D]
	DB byte   //Data Bank            ;expands 16bit [nnnn] to 24bit [DB:nnnn]
	PB byte   //Program Counter Bank ;expands 16bit PC     to 24bit PB:PC

	//its the emulation flag. bit cringe for it to be grouped with the registers but it makes the code cleaner
	E bool
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

func (r *registers) GetStackAddr() uint32 {
	return uint32(r.GetStack())
}

// It's paramount to interact with the values that can be locked to 8 bit through a getter/setter
// this prevents the return of a wrong value in emulation mode/with certain registers set
func (r *registers) GetStack() uint16 {
	if r.E {
		// Emulation mode: $01SS
		return 0x0100 | maskHighByte(r.S)
	}
	// Native mode: $SSSS
	return r.S
}

func (r *registers) SetStack(val uint16) {
	if r.E {
		r.S = 0x0100 | maskHighByte(val)
	} else {
		r.S = val
	}
}

func (r *registers) EmulationON() {
	if !r.E {
		r.E = true
		r.P |= 0x30
		r.S = 0x0100 | maskHighByte(r.S)
	}
}

func (r *registers) GetX() uint16 {
	if r.E || r.hasFlag(FlagX) {
		return maskHighByte(r.X)
	} else {
		return r.X
	}
}

func (r *registers) SetX(val uint16) {
	if r.E || r.hasFlag(FlagX) {
		r.X = maskHighByte(val)
	} else {
		r.X = val
	}
}
