// chasing the dream.
package cpu

const (
	FETCH = iota
	FETCH_OP_1
	FETCH_OP_2
	FETCH_OP_3
	EXECUTE
	WRITE_HI
	WRITE_LO

	EXTRA_CYCLE_W
	EXTRA_CYCLE_P

	REGISTER_READ

	READ_LO
	READ_HI
	READ_BANK

	RESOLVE_POINTER_LO
	RESOLVE_POINTER_HI
)

const (
	BASE_MODE = iota
	BASE_MODE_Y
	BASE_MODE_X
	INDIRECT              //()
	INDIRECT_LONG         //[]
	INDEXED_INDIRECT      //(,X)
	INDIRECT_INDEXED      //(),Y
	INDIRECT_LONG_INDEXED //[],Y
)

const (
	NO_IO = iota
	READ_RAM
	WRITE_RAM
)

func sta(_ uint16, _ int, cpu *CPU) (result uint16) {
	return cpu.r.GetA()
}

func lda(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	cpu.r.setFlag(FlagN, (1<<(width-1))&result == 0)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.SetA(result)
	return result
}

// represents all memry access modes. it abstracts away from the concrete instruction
// and resolves the write addresses and fetches the data so the outer state doesn't have to be aware
// of how the data is being obtained
type AccessMicroInstruction interface {
	Step(cpu *CPU, u *Umbrella) bool
	Reset(cpu *CPU)
	isPointer() bool
}

type Umbrella struct {
	state  int
	mode   int
	result uint16

	checkM        bool
	checkX        bool
	reverseWrites bool

	combineExecuteAndWrite bool

	addressHi, addressLo, addressBank uint32
	lowByte, highByte, bankByte       byte

	addressMode AccessMicroInstruction

	instructionFunc instructionFuncWith16BitReturn
}

func (i *Umbrella) Step(cpu *CPU) bool {
	switch i.state {
	case FETCH:
		if i.addressMode.Step(cpu, i) {
			if i.mode != READ_RAM {
				i.state = EXECUTE
			} else {
				if i.is8Bit(cpu) {
					i.instructionFunc(uint16(i.lowByte), 8, cpu)
				} else {
					i.result = i.instructionFunc(createWord(i.highByte, i.lowByte), 16, cpu)
				}
				return true
			}
		}
	case EXECUTE:
		if i.is8Bit(cpu) {
			i.result = i.instructionFunc(uint16(i.lowByte), 8, cpu)
			if i.mode == WRITE_RAM {
				if i.combineExecuteAndWrite {
					i.WriteLo(cpu)
					return true
				} else {
					i.state = WRITE_LO
				}

			} else {
				return true
			}
		} else {
			i.result = i.instructionFunc(createWord(i.highByte, i.lowByte), 16, cpu)
			if i.mode == WRITE_RAM {
				if i.combineExecuteAndWrite {
					if i.reverseWrites {
						i.WriteLo(cpu)
					} else {
						i.WriteHi(cpu)
					}
				} else {
					if i.reverseWrites {
						i.state = WRITE_LO
					} else {
						i.state = WRITE_HI
					}
				}

			} else {
				return true
			}

		}
	case WRITE_HI:
		i.WriteHi(cpu)
		if i.reverseWrites {
			return true
		}
	case WRITE_LO:
		i.WriteLo(cpu)
		if !i.reverseWrites || i.is8Bit(cpu) {
			return true
		}
	}

	return false
}

func (i *Umbrella) Reset(cpu *CPU) {
	i.state = FETCH
	i.addressMode.Reset(cpu)
}

func (i *Umbrella) is8Bit(cpu *CPU) bool {
	return (i.checkM && cpu.r.hasFlag(FlagM)) || (i.checkX && cpu.r.hasFlag(FlagX))
}

func (i *Umbrella) WriteHi(cpu *CPU) {
	cpu.bus.WriteByte(i.addressHi, getHighByte(i.result))
	i.state = WRITE_LO
}

func (i *Umbrella) WriteLo(cpu *CPU) {
	cpu.bus.WriteByte(i.addressLo, getLowByte(i.result))
	i.state = WRITE_HI
}

// the micro instruction for direct/direct, X/ diecct, Y
type DirXY struct {
	state  int
	mode   int
	isPEI  bool
	checkP bool

	register uint16
}

func (i *DirXY) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		if cpu.isW() {
			if i.isXY() {
				i.state = REGISTER_READ
			} else {
				i.state = READ_LO
			}
		} else {
			i.state = EXTRA_CYCLE_W
		}
	case EXTRA_CYCLE_W:
		if i.isXY() {
			i.state = REGISTER_READ
		} else {
			i.state = READ_LO
		}
	case REGISTER_READ:
		if i.mode == BASE_MODE_Y {
			i.register = cpu.r.GetY()
			i.state = READ_LO
			break
		}
		if i.mode == BASE_MODE_X || i.mode == INDEXED_INDIRECT {
			i.register = cpu.r.GetX()
			i.state = READ_LO
			break
		}
	case READ_LO:
		if i.isXY() {
			u.addressLo, u.addressHi = directPageXY(cpu, u.lowByte, i.register)
		}
		if i.mode == BASE_MODE || i.mode == INDIRECT || i.mode == INDIRECT_INDEXED {
			u.addressLo, u.addressHi = directPage(cpu, u.lowByte, i.isPEI)
		}
		if i.mode == INDIRECT_LONG || i.mode == INDIRECT_LONG_INDEXED {
			u.addressLo, u.addressHi, u.addressBank = directPageLong(cpu, u.lowByte)
		}

		u.lowByte = cpu.bus.ReadByte(u.addressLo)
		//TODO unsure how this will pan out. check later
		if u.is8Bit(cpu) && !i.isPointer() {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.bus.ReadByte(u.addressHi)
		if !i.isPointer() {
			return true
		}
		if i.mode == INDIRECT_INDEXED || i.mode == INDIRECT_LONG_INDEXED {
			i.register = cpu.r.GetY()
		} else {
			i.register = 0
		}
		if i.isIndirectLong() {
			i.state = READ_BANK
		} else {
			u.addressLo = mask24(createAddress(u.lowByte, u.highByte, cpu.r.DB) + uint32(i.register))
			u.addressHi = mask24(u.addressLo + 1)

			if i.checkP {
				if cpu.r.hasFlag(FlagX) {
					if isPageBoundaryCrossed(i.register, i.register+uint16(u.lowByte)) {
						i.state = EXTRA_CYCLE_P
						break
					}
				} else {
					//some weird phantom cycle in normal mode
					i.state = EXTRA_CYCLE_P
					break
				}
			}
			if u.mode != READ_RAM {
				return true
			} else {
				i.state = RESOLVE_POINTER_LO
			}
		}
	case RESOLVE_POINTER_LO:
		u.lowByte = cpu.bus.ReadByte(u.addressLo)
		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = RESOLVE_POINTER_HI
		}
	case RESOLVE_POINTER_HI:
		u.highByte = cpu.bus.ReadByte(u.addressHi)
		return true
	case READ_BANK:
		u.bankByte = cpu.bus.ReadByte(u.addressBank)
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, u.bankByte) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)
		if u.mode != READ_RAM {
			return true
		} else {
			i.state = RESOLVE_POINTER_LO
		}
	case EXTRA_CYCLE_P:
		if u.mode != READ_RAM {
			return true
		} else {
			i.state = RESOLVE_POINTER_LO
		}
	}
	return false
}

func (i *DirXY) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

func (i *DirXY) isXY() bool {
	return i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y || i.mode == INDEXED_INDIRECT
}

func (i *DirXY) isPointer() bool {
	return !(i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y || i.mode == BASE_MODE)
}

func (i *DirXY) isIndirectLong() bool {
	return i.mode == INDIRECT_LONG_INDEXED || i.mode == INDIRECT_LONG
}
