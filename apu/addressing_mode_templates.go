package apu

const (
	DEFAULT = iota
	X_INDEXED
	Y_INDEXED
	BIT

	ACCUMULATOR
	REGISTER_X
	REGISTER_Y
)

const (
	FETCH_BYTE1 = iota
	INDEX_DATA
	RESOLVE_ADDRESS
)

const (
	READ_RAM = iota
	WRITE_RAM
)

type AddressMode interface {
	step(*CPU) (bool, byte, uint16)
	reset()
}

type DirectPage struct {
	io    int
	mode  int
	state int

	lo byte

	bitOp func(byte) byte

	addr uint16
}

func (dp *DirectPage) step(cpu *CPU) (bool, byte, uint16) {
	switch dp.state {
	case FETCH_BYTE1:
		dp.lo = cpu.fetchByte()

		switch dp.mode {
		case X_INDEXED, Y_INDEXED:
			dp.state = INDEX_DATA
		case DEFAULT, BIT:
			dp.state = RESOLVE_ADDRESS
		}
	case INDEX_DATA:
		if dp.mode == X_INDEXED {
			dp.lo += cpu.r.X
		}
		if dp.mode == Y_INDEXED {
			dp.lo += cpu.r.Y
		}
		dp.state = RESOLVE_ADDRESS
	case RESOLVE_ADDRESS:
		dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)

		if dp.io == WRITE_RAM {
			return true, dp.lo, dp.addr
		}

		dp.lo = cpu.psram.Read8(dp.addr)
		if dp.mode == BIT {
			dp.lo = dp.bitOp(dp.lo)
		}
		return true, dp.lo, dp.addr
	}
	return false, 0, 0
}

func (dp *DirectPage) reset() {
	dp.state = FETCH_BYTE1
}

type AccessRegister struct {
	mode int
}

func (r *AccessRegister) step(cpu *CPU) (bool, byte, uint16) {
	switch r.mode {
	case REGISTER_X:
		return true, cpu.r.X, 0
	case REGISTER_Y:
		return true, cpu.r.Y, 0
	default:
		//accumulator
		return true, cpu.r.A, 0
	}
}

func (dp *AccessRegister) reset() {
}
