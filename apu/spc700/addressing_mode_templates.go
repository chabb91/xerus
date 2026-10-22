package spc700

type InstructionState int
type AddressingMode int

const (
	DEFAULT AddressingMode = iota
	X_INDEXED
	Y_INDEXED
	INDEXED_INDIRECT      //[+X]
	INDIRECT_INDEXED      //[]+Y
	INDIRECT_INDEXED_LAST //[]+Y but as last step (cringe)
	BIT

	ACCUMULATOR
	REGISTER_X
	REGISTER_Y
	STACK_POINTER
)

const (
	FETCH_BYTE1 InstructionState = iota
	FETCH_BYTE2
	INDEX_DATA
	RESOLVE_ADDRESS1
	RESOLVE_ADDRESS2
	RESOLVE_INDIRECTION
)

type AddressMode interface {
	step(*CPU) *ResolvedAddress
	reset()
}

type AmResultTypeTag int

const (
	MEMORY_REGION AmResultTypeTag = iota
	REGISTER_REGION
)

type ResolvedAddress struct {
	tag AmResultTypeTag

	//Memory variant
	value   byte
	address uint16

	//Register variant
	register *byte
}

func newMemoryResult(value byte, address uint16) *ResolvedAddress {
	return &ResolvedAddress{
		tag:     MEMORY_REGION,
		value:   value,
		address: address,
	}
}

func newRegisterResult(register *byte) *ResolvedAddress {
	return &ResolvedAddress{
		tag:      REGISTER_REGION,
		register: register,
	}
}

func (amr *ResolvedAddress) isMemory() bool {
	return amr.tag == MEMORY_REGION
}

func (amr *ResolvedAddress) isRegister() bool {
	return amr.tag == REGISTER_REGION
}

func (amr *ResolvedAddress) setRegister(value byte) bool {
	isReg := amr.isRegister()
	if isReg {
		*amr.register = value
	}
	return isReg
}

func (amr *ResolvedAddress) getValueAtAddress() byte {
	if amr.isMemory() {
		return amr.value
	} else {
		return *amr.register
	}
}

func (amr *ResolvedAddress) writeData(cpu *CPU, value byte) {
	if amr.isRegister() {
		*amr.register = value
	} else {
		cpu.psram.Write8(amr.address, value)
	}
}

func (amr *ResolvedAddress) dummyRead(cpu *CPU) {
	if amr.isMemory() {
		cpu.psram.Read8(amr.address)
	}
}

type DirectPage struct {
	mode  AddressingMode
	state InstructionState

	lo byte

	bitOp func(byte) byte

	addr uint16
	reg  uint16

	skipRead bool

	indexAndResolve bool
	autoIncrement   byte

	//fields for optimization
	afterFetch InstructionState
	isIndirect bool
}

func (dp *DirectPage) init() *DirectPage {
	switch dp.mode {
	case X_INDEXED, Y_INDEXED, INDEXED_INDIRECT, INDIRECT_INDEXED:
		dp.afterFetch = INDEX_DATA
	case DEFAULT, BIT, INDIRECT_INDEXED_LAST:
		dp.afterFetch = RESOLVE_ADDRESS1
	}

	dp.isIndirect = dp.mode == INDEXED_INDIRECT || dp.mode == INDIRECT_INDEXED || dp.mode == INDIRECT_INDEXED_LAST

	return dp
}

func (dp *DirectPage) step(cpu *CPU) *ResolvedAddress {
	switch dp.state {
	case FETCH_BYTE1:
		dp.lo = cpu.fetchByte()
		dp.state = dp.afterFetch
	case INDEX_DATA:
		switch dp.mode {
		case X_INDEXED, INDEXED_INDIRECT:
			dp.lo += cpu.r.X
		case Y_INDEXED:
			dp.lo += cpu.r.Y
		case INDIRECT_INDEXED:
			dp.reg = uint16(cpu.r.Y)
		case INDIRECT_INDEXED_LAST:
			dp.reg = uint16(cpu.r.Y)
			dp.state = RESOLVE_INDIRECTION
			return nil
		case REGISTER_X:
			dp.lo = cpu.r.X
			cpu.r.X += dp.autoIncrement
			if dp.indexAndResolve {
				dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)
				dp.lo = cpu.psram.Read8(dp.addr)
				return newMemoryResult(dp.lo, dp.addr)
			}
		case REGISTER_Y:
			dp.lo = cpu.r.Y
			cpu.r.Y += dp.autoIncrement
			if dp.indexAndResolve {
				dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)
				dp.lo = cpu.psram.Read8(dp.addr)
				return newMemoryResult(dp.lo, dp.addr)
			}
		}
		dp.state = RESOLVE_ADDRESS1
	case RESOLVE_ADDRESS1:
		dp.addr = uint16(cpu.r.getDirectPageNum())<<8 | uint16(dp.lo)

		if dp.skipRead {
			return newMemoryResult(dp.lo, dp.addr)
		}

		dp.lo = cpu.psram.Read8(dp.addr)
		if dp.mode == BIT {
			dp.lo = dp.bitOp(dp.lo)
		}

		if !(dp.isIndirect) {
			return newMemoryResult(dp.lo, dp.addr)
		} else {
			dp.state = RESOLVE_ADDRESS2
		}
	case RESOLVE_ADDRESS2:
		addr := uint16(cpu.r.getDirectPageNum())<<8 | ((dp.addr + 1) & 0xFF)
		dp.addr = (uint16(cpu.psram.Read8(addr))<<8 | uint16(dp.lo))

		if dp.mode == INDIRECT_INDEXED_LAST {
			dp.state = INDEX_DATA
		} else {
			dp.state = RESOLVE_INDIRECTION
		}
	case RESOLVE_INDIRECTION:
		dp.addr += dp.reg
		dp.lo = cpu.psram.Read8(dp.addr)
		return newMemoryResult(dp.lo, dp.addr)
	}
	return nil
}

func (dp *DirectPage) reset() {
	if dp.mode == REGISTER_X || dp.mode == REGISTER_Y {
		dp.state = INDEX_DATA
	} else {
		dp.state = FETCH_BYTE1
	}
}

type Absolute struct {
	mode  AddressingMode
	state InstructionState

	hi, lo, reg byte
	addr        uint16
}

func (a *Absolute) step(cpu *CPU) *ResolvedAddress {
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

		a.lo = cpu.psram.Read8(a.addr)
		return newMemoryResult(a.lo, a.addr)

	}
	return nil
}

func (a *Absolute) reset() {
	a.state = FETCH_BYTE1
}

type Immediate struct {
}

func (r *Immediate) step(cpu *CPU) *ResolvedAddress {
	pc := cpu.r.PC //probably unnecessary but for correctness
	return newMemoryResult(cpu.fetchByte(), pc)
}

func (r *Immediate) reset() {
}

type AccessRegister struct {
	mode AddressingMode
}

func (r *AccessRegister) step(cpu *CPU) *ResolvedAddress {
	switch r.mode {
	case REGISTER_X:
		return newRegisterResult(&cpu.r.X)
	case REGISTER_Y:
		return newRegisterResult(&cpu.r.Y)
	case STACK_POINTER:
		return newRegisterResult(&cpu.r.SP)
	default:
		//accumulator
		return newRegisterResult(&cpu.r.A)
	}
}

func (r *AccessRegister) reset() {
}
