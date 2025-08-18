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

	//immediate
	CHECK_PARENT
	LOCKED_8
	LOCKED_16
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
}

type Umbrella struct {
	state  int
	mode   int
	result uint16

	checkM        bool
	checkX        bool
	reverseWrites bool

	combineExecuteAndWrite bool
	executeInFetch         bool

	addressHi, addressLo, addressBank uint32
	lowByte, highByte, bankByte       byte

	addressMode AccessMicroInstruction

	instructionFunc instructionFuncWith16BitReturn
}

func (i *Umbrella) Step(cpu *CPU) bool {
	switch i.state {
	case FETCH:
		if i.addressMode.Step(cpu, i) {
			if i.mode == WRITE_RAM {
				if !i.executeInFetch {
					i.state = EXECUTE
					break
				}
			}
			return i.Execute(cpu)
		}
	case EXECUTE:
		return i.Execute(cpu)
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

func (i *Umbrella) Execute(cpu *CPU) bool {
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
	return false
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

			//this is slightly incorrect.
			//the real hardware tries to read the Y regisger this cycle but it can only do it if X flag is 1 and
			//the page isnt crossed. what i do instead is just get Y and stall a cycle if needed.
			//this MIGHT be inaccurate i dont have the brainpower to know for sure but if it only affects internal state
			//so no memory reads then its completely fine
			if i.checkP {
				if cpu.r.hasFlag(FlagX) {
					if isPageBoundaryCrossed(i.register, i.register+uint16(u.lowByte)) {
						i.state = EXTRA_CYCLE_P
						break
					}
				} else {
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

// the micro instruction ABSOLUTE and I do mean ALL OF ABSOLUTE
// I mean every single instruction that has abs in its access mode this will service it.
// no kap on a stack
// the jumps are UNTESTED
type Absolute struct {
	state  int
	mode   int
	isPEI  bool
	checkP bool

	//this is intended to differentiate between the normal abs and normal abs JMP
	//there are other jump abs instructions but they are all jump only so its implied
	isJMP bool

	register uint16
}

func (i *Absolute) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		i.state = FETCH_OP_2
	case FETCH_OP_2:
		u.highByte = cpu.fetchByte()
		if i.isXY() || i.mode == INDEXED_INDIRECT {
			if i.checkP && cpu.r.hasFlag(FlagX) {
				//in some instructions if X/Y is 8 bit the register read can be done without consuming a cycle,
				//but only if the page isnt crossed.
				if i.mode == BASE_MODE_Y {
					i.register = cpu.r.GetY()
				}
				if i.mode == BASE_MODE_X {
					i.register = cpu.r.GetX()
				}
				u.result = createWord(u.highByte, u.lowByte)
				if isPageBoundaryCrossed(u.result, u.result+i.register) {
					i.state = EXTRA_CYCLE_P
				} else {
					i.state = READ_LO
				}
			} else {
				i.state = REGISTER_READ
			}
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
			u.addressLo, u.addressHi = absoluteXY(cpu.r.DB, u.highByte, u.lowByte, i.register)
		}
		if i.mode == BASE_MODE {
			if i.isJMP {
				u.addressLo = createAddress(u.lowByte, u.highByte, cpu.r.PB)
			} else {
				u.addressLo, u.addressHi = absolute(cpu.r.DB, u.highByte, u.lowByte)
			}
		}
		if i.mode == INDEXED_INDIRECT {
			u.result = createWord(u.highByte, u.lowByte) + i.register
			u.addressLo = mapOffsetToBank(cpu.r.PB, u.result)
			u.addressHi = mapOffsetToBank(cpu.r.PB, u.result+1)
		}
		if i.mode == INDIRECT || i.mode == INDIRECT_LONG {
			u.result = createWord(u.highByte, u.lowByte)
			u.addressLo = mapOffsetToBank(0x00, u.result)
			u.addressHi = mapOffsetToBank(0x00, u.result+1)
			u.addressBank = mapOffsetToBank(0x00, u.result+2)
		}

		//abs executes the instruction here if its not a pointer and we are in write mode
		if (u.executeInFetch && u.mode == WRITE_RAM && !i.isPointer()) || i.isJMP {
			return true
		}

		u.lowByte = cpu.bus.ReadByte(u.addressLo)

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
		if i.isIndirectLong() {
			i.state = READ_BANK
		} else {
			i.state = RESOLVE_POINTER_LO
			//return true
		}
	case RESOLVE_POINTER_LO:
		switch i.mode {
		case INDIRECT, INDEXED_INDIRECT:
			u.addressLo = createAddress(u.lowByte, u.highByte, cpu.r.PB)
		case INDIRECT_LONG:
			u.addressLo = createAddress(u.lowByte, u.highByte, u.bankByte)
		}
		return true
	case READ_BANK:
		u.bankByte = cpu.bus.ReadByte(u.addressBank)
		i.state = RESOLVE_POINTER_LO
		//return true
	case EXTRA_CYCLE_P:
		i.state = READ_LO
	}
	return false
}

func (i *Absolute) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

func (i *Absolute) isXY() bool {
	return i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y
}

func (i *Absolute) isPointer() bool {
	return !(i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y || i.mode == BASE_MODE)
}

func (i *Absolute) isIndirectLong() bool {
	return i.mode == INDIRECT_LONG_INDEXED || i.mode == INDIRECT_LONG
}

// the micro instruction for Long, just the normal one not the one for jump
// no point in including jump instructions under the umbrella
type Long struct {
	state int
	mode  int

	register uint16
}

func (i *Long) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		i.state = FETCH_OP_2
	case FETCH_OP_2:
		u.highByte = cpu.fetchByte()
		i.state = FETCH_OP_3
	case FETCH_OP_3:
		u.bankByte = cpu.fetchByte()

		if i.mode == BASE_MODE {
			i.register = 0
		}
		if i.mode == BASE_MODE_X {
			i.register = cpu.r.GetX()
		}
		i.state = READ_LO
	case READ_LO:
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, u.bankByte) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)
		//long executes the instruction here if we are in write mode
		if u.executeInFetch && u.mode == WRITE_RAM {
			return true
		}

		u.lowByte = cpu.bus.ReadByte(u.addressLo)
		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.bus.ReadByte(u.addressHi)
		return true
	}
	return false
}

func (i *Long) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

type Immediate struct {
	state int
	mode  int

	register uint16
}

func (i *Immediate) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		u.addressLo = mapOffsetToBank(cpu.r.PB, cpu.r.PC)
		if i.mode == LOCKED_8 || (i.mode == CHECK_PARENT && u.is8Bit(cpu)) {
			return true
		}
		i.state = FETCH_OP_2
	case FETCH_OP_2:
		u.highByte = cpu.fetchByte()
		u.addressHi = mapOffsetToBank(cpu.r.PB, cpu.r.PC)
		return true
	}
	return false
}

func (i *Immediate) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

type StackS struct {
	state int
	mode  int

	register uint16
}

func (i *StackS) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		i.state = REGISTER_READ
	case REGISTER_READ:
		i.register = uint16(u.lowByte) + cpu.r.S
		i.state = READ_LO
	case EXTRA_CYCLE_P:
		//its really just another register read no fancy page trickery but didnt want to create another needless enum for it
		i.register = cpu.r.GetY()
		i.state = RESOLVE_POINTER_LO
	case READ_LO:
		u.addressLo = mapOffsetToBank(0x00, i.register)
		u.addressHi = mapOffsetToBank(0x00, i.register+1)

		//StackS executes the instruction here if its not a pointer and we are in write mode
		if u.executeInFetch && u.mode == WRITE_RAM && !i.isPointer() {
			return true
		}
		u.lowByte = cpu.bus.ReadByte(u.addressLo)

		if !i.isPointer() && u.is8Bit(cpu) {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.bus.ReadByte(u.addressHi)
		if !i.isPointer() {
			return true
		}
		i.state = EXTRA_CYCLE_P
	case RESOLVE_POINTER_LO:
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, cpu.r.DB) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)

		if u.mode == WRITE_RAM {
			return true
		}

		u.lowByte = cpu.bus.ReadByte(u.addressLo)
		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = RESOLVE_POINTER_HI

		}
	case RESOLVE_POINTER_HI:
		u.highByte = cpu.bus.ReadByte(u.addressHi)
		return true
	}
	return false
}

func (i *StackS) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

func (i *StackS) isPointer() bool {
	return i.mode == INDIRECT_INDEXED
}
