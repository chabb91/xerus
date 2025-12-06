package apu

const (
	DEFAULT = iota
	X_INDEXED
	Y_INDEXED
	INDEXED_INDIRECT //[+X]
	INDIRECT_INDEXED //[]+Y
	BIT

	ACCUMULATOR
	REGISTER_X
	REGISTER_Y
)

const (
	FETCH_BYTE1 = iota
	FETCH_BYTE2
	INDEX_DATA
	RESOLVE_ADDRESS1
	RESOLVE_ADDRESS2
	RESOLVE_INDIRECTION
)

const (
	READ_RAM = iota
	WRITE_RAM
)

type AddressMode interface {
	step(*CPU) (bool, byte, uint16, *byte)
	reset()
}

type DirectPage struct {
	io    int
	mode  int
	state int

	lo byte

	bitOp func(byte) byte

	addr uint16
	reg  uint16

	indexAndResolve bool
	autoIncrement   byte
}

func (dp *DirectPage) step(cpu *CPU) (bool, byte, uint16, *byte) {
	switch dp.state {
	case FETCH_BYTE1:
		dp.lo = cpu.fetchByte()

		switch dp.mode {
		case X_INDEXED, Y_INDEXED, INDEXED_INDIRECT, INDIRECT_INDEXED:
			dp.state = INDEX_DATA
		case DEFAULT, BIT:
			dp.state = RESOLVE_ADDRESS1
		}
	case INDEX_DATA:
		if dp.mode == X_INDEXED || dp.mode == INDEXED_INDIRECT {
			dp.lo += cpu.r.X
		}
		if dp.mode == Y_INDEXED {
			dp.lo += cpu.r.Y
		}
		if dp.mode == INDIRECT_INDEXED {
			dp.reg = uint16(cpu.r.Y)
		}
		if dp.mode == REGISTER_X {
			dp.lo = cpu.r.X
			cpu.r.X += dp.autoIncrement
			if dp.indexAndResolve {
				dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)
				dp.lo = cpu.psram.Read8(dp.addr)
				return true, dp.lo, dp.addr, nil
			}
		}
		if dp.mode == REGISTER_Y {
			dp.lo = cpu.r.Y
			cpu.r.Y += dp.autoIncrement
			if dp.indexAndResolve {
				dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)
				dp.lo = cpu.psram.Read8(dp.addr)
				return true, dp.lo, dp.addr, nil
			}
		}
		dp.state = RESOLVE_ADDRESS1
	case RESOLVE_ADDRESS1:
		dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)

		if dp.io == WRITE_RAM && !(dp.mode == INDEXED_INDIRECT || dp.mode == INDIRECT_INDEXED) {
			return true, dp.lo, dp.addr, nil
		}

		dp.lo = cpu.psram.Read8(dp.addr)
		if dp.mode == BIT {
			dp.lo = dp.bitOp(dp.lo)
		}

		if !(dp.mode == INDEXED_INDIRECT || dp.mode == INDIRECT_INDEXED) {
			return true, dp.lo, dp.addr, nil
		} else {
			dp.state = RESOLVE_ADDRESS2
		}
	case RESOLVE_ADDRESS2:
		addr := uint16(cpu.r.getDirectPageNum())<<8 | ((dp.addr + 1) & 0xFF)
		dp.addr = (uint16(cpu.psram.Read8(addr))<<8 | uint16(dp.lo)) + dp.reg

		dp.state = RESOLVE_INDIRECTION
	case RESOLVE_INDIRECTION:
		dp.lo = cpu.psram.Read8(dp.addr)
		return true, dp.lo, dp.addr, nil
	}
	return false, 0, 0, nil
}

func (dp *DirectPage) reset() {
	if dp.mode == REGISTER_X || dp.mode == REGISTER_Y {
		dp.state = INDEX_DATA
	} else {
		dp.state = FETCH_BYTE1
	}
}

type Absolute struct {
	io    int
	mode  int
	state int

	hi, lo, reg byte
	addr        uint16
}

func (a *Absolute) step(cpu *CPU) (bool, byte, uint16, *byte) {
	switch a.state {
	case FETCH_BYTE1:
		a.lo = cpu.fetchByte()
		a.state = FETCH_BYTE2
	case FETCH_BYTE2:
		a.hi = cpu.fetchByte()

		switch a.mode {
		case X_INDEXED, Y_INDEXED:
			a.state = INDEX_DATA
		case DEFAULT:
			a.reg = 0
			a.state = RESOLVE_ADDRESS1
		}
	case INDEX_DATA:
		if a.mode == X_INDEXED {
			a.reg = cpu.r.X
		}
		if a.mode == Y_INDEXED {
			a.reg = cpu.r.Y
		}
		a.state = RESOLVE_ADDRESS1
	case RESOLVE_ADDRESS1:
		a.addr = (uint16(a.hi)<<8 | uint16(a.lo)) + uint16(a.reg)

		if a.io == WRITE_RAM {
			return true, a.lo, a.addr, nil
		}

		a.lo = cpu.psram.Read8(a.addr)
		return true, a.lo, a.addr, nil

	}
	return false, 0, 0, nil
}

func (a *Absolute) reset() {
	a.state = FETCH_BYTE1
}

type Immediate struct {
}

func (r *Immediate) step(cpu *CPU) (bool, byte, uint16, *byte) {
	return true, cpu.fetchByte(), 0, nil
}

func (r *Immediate) reset() {
}

type AccessRegister struct {
	mode int
}

func (r *AccessRegister) step(cpu *CPU) (bool, byte, uint16, *byte) {
	switch r.mode {
	case REGISTER_X:
		return true, cpu.r.X, 0, &cpu.r.X
	case REGISTER_Y:
		return true, cpu.r.Y, 0, &cpu.r.Y
	default:
		//accumulator
		return true, cpu.r.A, 0, &cpu.r.A
	}
}

func (r *AccessRegister) reset() {
}
