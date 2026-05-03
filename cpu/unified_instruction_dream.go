// chasing the dream.
package cpu

type InstructionState int
type AddressingMode int
type IOType int

const (
	FETCH InstructionState = iota
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
	BASE_MODE AddressingMode = iota
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
	NO_IO IOType = iota
	READ_RAM
	WRITE_RAM
)

// represents all memry access modes. it abstracts away from the concrete instruction
// and resolves the write addresses and fetches the data so the outer state doesn't have to be aware
// of how the data is being obtained
type AccessMicroInstruction interface {
	Setup(*Umbrella)
	Step(cpu *CPU, u *Umbrella) bool
	Reset(cpu *CPU)
}

type Umbrella struct {
	state  InstructionState
	mode   IOType
	result uint16

	is8Bit func(*CPU) bool

	reverseWrites          bool
	combineExecuteAndWrite bool
	executeInFetch         bool
	stackWrite             bool

	addressHi, addressLo, addressBank uint32
	lowByte, highByte, bankByte       byte

	addressMode AccessMicroInstruction

	instructionFunc instructionFuncWith16BitReturn
}

func NewUmbrellaWrite(ifunc instructionFuncWith16BitReturn, am AccessMicroInstruction,
	reverseWrites, combineExecuteAndWrite, executeInFetch, stackWrite bool,
	is8Bit func(*CPU) bool) *Umbrella {

	ret := &Umbrella{
		mode:                   WRITE_RAM,
		instructionFunc:        ifunc,
		addressMode:            am,
		reverseWrites:          reverseWrites,
		combineExecuteAndWrite: combineExecuteAndWrite,
		executeInFetch:         executeInFetch,
		stackWrite:             stackWrite,
		is8Bit:                 is8Bit,
	}
	ret.addressMode.Setup(ret)
	return ret

}

func NewUmbrellaRead(ifunc instructionFuncWith16BitReturn, am AccessMicroInstruction,
	is8Bit func(*CPU) bool) *Umbrella {

	ret := &Umbrella{
		mode:                   READ_RAM,
		instructionFunc:        ifunc,
		addressMode:            am,
		reverseWrites:          false,
		combineExecuteAndWrite: false,
		executeInFetch:         false,
		stackWrite:             false,
		is8Bit:                 is8Bit,
	}
	ret.addressMode.Setup(ret)
	return ret

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

func (i *Umbrella) WriteHi(cpu *CPU) {
	if i.stackWrite {
		//STACKWRITE is specific to PEA PEI PER and is always 16 bit and is like this
		cpu.r.SetStack(cpu.r.S)
		cpu.PushByteNewOpCode(getHighByte(i.result))
	} else {
		cpu.writeByte(i.addressHi, getHighByte(i.result))
	}
	i.state = WRITE_LO
}

func (i *Umbrella) WriteLo(cpu *CPU) {
	if i.stackWrite {
		//STACKWRITE is specific to PEA PEI PER and is always 16 bit and is like this
		cpu.PushByteNewOpCode(getLowByte(i.result))
		cpu.r.SetStack(cpu.r.S)
	} else {
		cpu.writeByte(i.addressLo, getLowByte(i.result))
	}
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

func is8BitM(cpu *CPU) bool {
	return cpu.r.hasFlag(FlagM)
}

func is8BitX(cpu *CPU) bool {
	return cpu.r.hasFlag(FlagX)
}

func isNot8Bit(cpu *CPU) bool {
	return false
}

// the micro instruction for direct/direct, X/ diecct, Y
type Direct struct {
	state InstructionState
	mode  AddressingMode
	isPEI bool

	register uint16

	addressResolver func(cpu *CPU, op byte) (addressLo, addressHi, addressBank uint32)
	isXY            bool
	isPointer       bool
	checkP          bool
	isIndirectLong  bool
}

func (i *Direct) Setup(u *Umbrella) {
	i.isXY = i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y || i.mode == INDEXED_INDIRECT
	i.checkP = i.mode == INDIRECT_INDEXED && u.mode == READ_RAM
	i.isPointer = !(i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y || i.mode == BASE_MODE)
	i.isIndirectLong = i.mode == INDIRECT_LONG_INDEXED || i.mode == INDIRECT_LONG

	switch i.mode {
	case BASE_MODE_X, BASE_MODE_Y, INDEXED_INDIRECT:
		i.addressResolver = i.directPageXY
	case BASE_MODE, INDIRECT, INDIRECT_INDEXED:
		i.addressResolver = i.directPage
	case INDIRECT_LONG, INDIRECT_LONG_INDEXED:
		i.addressResolver = i.directPageLong
	}
}

func (i *Direct) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		if cpu.isW() {
			if i.isXY {
				i.state = REGISTER_READ
			} else {
				i.state = READ_LO
			}
		} else {
			i.state = EXTRA_CYCLE_W
		}
	case EXTRA_CYCLE_W:
		if i.isXY {
			i.state = REGISTER_READ
		} else {
			i.state = READ_LO
		}
	case REGISTER_READ:
		switch i.mode {
		case BASE_MODE_X, INDEXED_INDIRECT:
			i.register = cpu.r.GetX()
		case BASE_MODE_Y:
			i.register = cpu.r.GetY()
		}
		i.state = READ_LO
	case READ_LO:
		u.addressLo, u.addressHi, u.addressBank = i.addressResolver(cpu, u.lowByte)

		//executeinFetch can only be true in write mode if using the constructor
		if u.executeInFetch && !i.isPointer {
			return true
		}

		u.lowByte = cpu.readByte(u.addressLo)
		if u.is8Bit(cpu) && !i.isPointer {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.readByte(u.addressHi)
		if !i.isPointer {
			return true
		}
		if i.mode == INDIRECT_INDEXED || i.mode == INDIRECT_LONG_INDEXED {
			i.register = cpu.r.GetY()
		} else {
			i.register = 0
		}
		if i.isIndirectLong {
			i.state = READ_BANK
		} else {
			u.addressLo = mask24(createAddress(u.lowByte, u.highByte, cpu.r.DB) + uint32(i.register))
			u.addressHi = mask24(u.addressLo + 1)

			if u.mode == WRITE_RAM {
				return true
			}

			//this is slightly incorrect.
			//the real hardware tries to read the Y regisger this cycle
			//but it can only do it if X flag is 1 and
			//the page isnt crossed. what i do instead is just get Y and stall a cycle if needed.
			if i.checkP {
				if !cpu.r.hasFlag(FlagX) ||
					isPageBoundaryCrossed(i.register, i.register+uint16(u.lowByte)) {
					i.state = EXTRA_CYCLE_P
					break
				}
			}
			i.state = RESOLVE_POINTER_LO
		}
	case RESOLVE_POINTER_LO:
		u.lowByte = cpu.readByte(u.addressLo)
		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = RESOLVE_POINTER_HI
		}
	case RESOLVE_POINTER_HI:
		u.highByte = cpu.readByte(u.addressHi)
		return true
	case READ_BANK:
		u.bankByte = cpu.readByte(u.addressBank)
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, u.bankByte) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)
		if u.mode != READ_RAM {
			return true
		} else {
			i.state = RESOLVE_POINTER_LO
		}
	case EXTRA_CYCLE_P:
		//only happens in read mode, checked it before
		i.state = RESOLVE_POINTER_LO
	}
	return false
}

func (i *Direct) directPage(cpu *CPU, op byte) (addressLo, addressHi, _ uint32) {
	addressLo, addressHi = directPageLogic(cpu, op, 0, i.isPEI)
	return
}

func (i *Direct) directPageXY(cpu *CPU, op byte) (addressLo, addressHi, _ uint32) {
	addressLo, addressHi = directPageLogic(cpu, op, i.register, false)
	//little hardware quirk
	//the last part is just the !isW function
	if i.mode == INDEXED_INDIRECT && cpu.r.E && (cpu.r.D&0xFF != 0) {
		addressHi = (addressLo+1)&0xFF | addressLo&0xFFFF00
	}
	return
}

func (i *Direct) directPageLong(cpu *CPU, op byte) (addressLo, addressHi, addressBank uint32) {
	offset := cpu.r.D + uint16(op)
	addressLo = mapOffsetToBank(0x00, offset)
	addressHi = mapOffsetToBank(0x00, offset+1)
	addressBank = mapOffsetToBank(0x00, offset+2)

	return
}

func (i *Direct) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

// the micro instruction for the non jmp ABSOLUTE. (base(X/Y))
type Absolute struct {
	state InstructionState
	mode  AddressingMode

	register             uint16
	checkP               bool
	isXY, isIndirectLong bool
}

func (i *Absolute) Setup(u *Umbrella) {
	i.isXY = i.mode == BASE_MODE_X || i.mode == BASE_MODE_Y
	i.isIndirectLong = i.mode == INDIRECT_LONG_INDEXED || i.mode == INDIRECT_LONG

	i.checkP = u.mode == READ_RAM && i.isXY
	i.register = 0
}

func (i *Absolute) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		i.state = FETCH_OP_2
	case FETCH_OP_2:
		u.highByte = cpu.fetchByte()
		if i.isXY {
			//in read mode if X == 1 the register read can be done without consuming a cycle,
			//but only if the page isnt crossed.
			if i.checkP && cpu.r.hasFlag(FlagX) {
				switch i.mode {
				case BASE_MODE_Y:
					i.register = cpu.r.GetY()
				case BASE_MODE_X:
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
		switch i.mode {
		case BASE_MODE_Y:
			i.register = cpu.r.GetY()
		case BASE_MODE_X:
			i.register = cpu.r.GetX()
		}
		i.state = READ_LO
	case READ_LO:
		u.addressLo, u.addressHi = absolute(cpu.r.DB, u.highByte, u.lowByte, i.register)

		//abs executes the instruction here if its not a pointer and we are in write mode
		if u.executeInFetch {
			return true
		}

		u.lowByte = cpu.readByte(u.addressLo)

		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.readByte(u.addressHi)
		return true
	case EXTRA_CYCLE_P:
		i.state = READ_LO
	}
	return false
}

func (i *Absolute) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

// the micro instruction for Long, just the normal one not the one for jump
// no point in including jump instructions under the umbrella
type Long struct {
	state InstructionState
	mode  AddressingMode

	register uint16
}

func (i *Long) Setup(u *Umbrella) {
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

		switch i.mode {
		case BASE_MODE:
			i.register = 0
		case BASE_MODE_X:
			i.register = cpu.r.GetX()
		}
		i.state = READ_LO
	case READ_LO:
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, u.bankByte) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)

		//long executes the instruction here if we are in write mode
		if u.executeInFetch {
			return true
		}

		u.lowByte = cpu.readByte(u.addressLo)
		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.readByte(u.addressHi)
		return true
	}
	return false
}

func (i *Long) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}

type Immediate struct {
	state InstructionState
	mode  AddressingMode

	register uint16
}

func (i *Immediate) Setup(u *Umbrella) {
}

func (i *Immediate) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		u.addressLo = mapOffsetToBank(cpu.r.PB, cpu.r.PC)
		if (i.mode == CHECK_PARENT && u.is8Bit(cpu)) || i.mode == LOCKED_8 {
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
	state InstructionState
	mode  AddressingMode

	executeInFetch bool
	isPointer      bool

	register uint16
}

func (i *StackS) Setup(u *Umbrella) {
	i.isPointer = i.mode == INDIRECT_INDEXED
	i.executeInFetch = u.executeInFetch && u.mode == WRITE_RAM && !i.isPointer
}

func (i *StackS) Step(cpu *CPU, u *Umbrella) bool {
	switch i.state {
	case FETCH_OP_1:
		u.lowByte = cpu.fetchByte()
		i.state = REGISTER_READ
	case REGISTER_READ:
		i.register = uint16(u.lowByte) + cpu.r.GetStack() //cpu.r.S
		i.state = READ_LO
	case EXTRA_CYCLE_P:
		//its really just another register read no fancy page trickery but didnt want to create another needless enum for it
		i.register = cpu.r.GetY()
		i.state = RESOLVE_POINTER_LO
	case READ_LO:
		u.addressLo = mapOffsetToBank(0x00, i.register)
		u.addressHi = mapOffsetToBank(0x00, i.register+1)

		//StackS executes the instruction here if its not a pointer and we are in write mode
		if i.executeInFetch {
			return true
		}
		u.lowByte = cpu.readByte(u.addressLo)

		if !i.isPointer && u.is8Bit(cpu) {
			return true
		} else {
			i.state = READ_HI
		}
	case READ_HI:
		u.highByte = cpu.readByte(u.addressHi)
		if !i.isPointer {
			return true
		}
		i.state = EXTRA_CYCLE_P
	case RESOLVE_POINTER_LO:
		u.addressLo = mask24(createAddress(u.lowByte, u.highByte, cpu.r.DB) + uint32(i.register))
		u.addressHi = mask24(u.addressLo + 1)

		if u.mode == WRITE_RAM {
			return true
		}

		u.lowByte = cpu.readByte(u.addressLo)
		if u.is8Bit(cpu) {
			return true
		} else {
			i.state = RESOLVE_POINTER_HI

		}
	case RESOLVE_POINTER_HI:
		u.highByte = cpu.readByte(u.addressHi)
		return true
	}
	return false
}

func (i *StackS) Reset(cpu *CPU) {
	i.state = FETCH_OP_1
}
