// chasing the dream.
package cpu

const (
	FETCH = iota
	FETCH_OP_1
	FETCH_OP_2
	RESOLVE_POINTER
	RESOLVE_POINTER_HI
	READ_POINTER_LO
	READ_POINTER_HI
	EXECUTE
	WRITE_HI
	WRITE_LO

	EXTRA_CYCLE
	REGISTER_READ

	READ_LO
	READ_HI
	READ_BANK
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

func sta(_ uint16, _ int, cpu *CPU) (result uint16) {
	return cpu.r.GetA()
}

// represents all memry access modes. it abstracts away from the concrete instruction
// and resolves the write addresses and fetches the data so the outer state doesn't have to be aware
// of how the data is being obtained
type AccessMicroInstruction interface {
	Step(cpu *CPU, u *Umbrella) bool
	Reset(cpu *CPU)
}

type Umbrella struct {
	state  int
	result uint16

	write         bool
	checkM        bool
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
			i.state = EXECUTE
		}
	case EXECUTE:
		if i.checkM && cpu.r.hasFlag(FlagM) {
			i.result = i.instructionFunc(uint16(i.lowByte), 8, cpu)
			if i.write && i.combineExecuteAndWrite {
				i.WriteLo(cpu)
				return true
			} else {
				i.state = WRITE_LO
			}
		} else {
			i.result = i.instructionFunc(createWord(i.highByte, i.lowByte), 16, cpu)
			if i.write && i.combineExecuteAndWrite {
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
		}
		if !i.write {
			return true
		}
	case WRITE_HI:
		i.WriteHi(cpu)
		if i.reverseWrites {
			return true
		}
	case WRITE_LO:
		i.WriteLo(cpu)
		if !i.reverseWrites || (i.checkM && cpu.r.hasFlag(FlagM)) {
			return true
		}
	}

	return false
}

func (i *Umbrella) Reset(cpu *CPU) {
	i.state = FETCH
	i.addressMode.Reset(cpu)
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
	state int
	mode  int
	isPEI bool

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
			i.state = EXTRA_CYCLE
		}
	case EXTRA_CYCLE:
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
		if i.mode == INDIRECT_INDEXED || i.mode == INDIRECT_LONG_INDEXED {
			i.register = cpu.r.GetY()
			i.state = RESOLVE_POINTER
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
		if cpu.r.hasFlag(FlagM) && !i.isPointer() && u.checkM {
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
			return true
		}
	case READ_BANK:
		u.bankByte = cpu.bus.ReadByte(u.addressBank)
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, u.bankByte) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)
		return true
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
