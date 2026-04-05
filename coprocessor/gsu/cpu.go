package gsu

import (
	"SNES_emulator/coprocessor"
	"SNES_emulator/internal/constants"
	"fmt"
)

type immediateInstructionFunc func(gsu *GSU)

const SRAM_BASE_BANK byte = 0x70

var opcodeBuffer [100000]byte
var r12Buffer [100000]uint16
var r13Buffer [100000]uint16
var r15Buffer [100000]uint16
var cpuRegBuffer [100000]string
var instructionBuffer [100000]string
var opcodeIndex int

type GSU struct {
	cartridge coprocessor.CartridgeDataSource

	r registers

	cache      [0x200]byte
	cacheFlags uint32

	immediateBytes       [3]byte
	immediateOpcode      byte
	immediateInstruction immediateInstructionFunc

	prevRamAddr uint32 //the full address expanded with the SRAM_BASE_BANK.

	sReg, dReg byte

	currentOpcode byte
	StopCnt       int
}

func New() coprocessor.Coprocessor {
	gsu := &GSU{StopCnt: 88}
	gsu.r.gsu = gsu
	gsu.r.cpuRegister15Buffer = R15_NOT_BRANCHING

	return gsu
}

func parseOpcode(opcode byte, altnum uint16) string {
	opcodeHn := opcode & 0xF0
	switch {
	case opcode-5 <= 0xA: //BRANCH instructions 0x05-0x0F UNTESTED
		switch opcode {
		case 0x05:
			return "BRA"
		case 0x06:
			return "BGE"
		case 0x07:
			return "BLT"
		case 0x08:
			return "BNE"
		case 0x09:
			return "BEQ"
		case 0x0A:
			return "BPL"
		case 0x0B:
			return "BMI"
		case 0x0C:
			return "BCC"
		case 0x0D:
			return "BCS"
		case 0x0E:
			return "BVC"
		case 0x0F:
			return "BVS"
		}
	case opcodeHn == 0xF0: //IWT instructions
		if altnum == 1 {
			return "LM"
		}
		if altnum == 2 {
			return "SM"
		}
		if altnum == 3 {
			return "IWT"
		}
		return "IWT"
	case opcodeHn == 0xA0: //IBT instructions
		if altnum == 1 {
			return "LMS"
		}
		if altnum == 2 {
			return "SMS"
		}
		if altnum == 3 {
			return "IBT"
		}
		return "IBT"
	case opcode-0x30 <= 0xB: //STW instructions
		return "STW"
	case opcode-0x40 <= 0xB: //LDW instructions
		return "LDW"
	case opcode == 0x90:
		return "SBK"
	case opcode == 0xEF: //GET(load byte from rom)
		if altnum == 1 {
			return "GETBH"
		}
		if altnum == 2 {
			return "GETBL"
		}
		if altnum == 3 {
			return "GETBS"
		}
		return "GETB"
	case opcode == 0xDF: //GETC pretending as RAMB/ROMB
		if altnum == 1 {
			return "GETC"
		}
		if altnum == 2 {
			return "RAMB"
		}
		if altnum == 3 {
			return "ROMB"
		}
		return "GETC"
	case opcode == 0x4E: //COLOR/CMODE
		if altnum == 1 {
			return "CMODE"
		}
		return "COLOR"
	case opcodeHn == 0x50: //ADD/ADC instructions
		if altnum == 1 || altnum == 3 {
			return "ADC"
		}
		return "ADD"
	case opcodeHn == 0x60: //SUB/SBC//CMP instructions
		if altnum == 0 || altnum == 2 {
			return "SUB"
		}
		if altnum == 3 {
			return "CMP"
		}
		return "SBC"
	case opcode == 0x70: //MERGE
		return "MERGE"
	case opcode-0x71 <= 0xE: //AND/BIC
		if altnum == 0 || altnum == 2 {
			return "AND"
		}
		return "BIC"
	case opcode == 0xC0: //HIB
		return "HIB"
	case opcode-0xC1 <= 0xE: //OR/XOR
		if altnum == 0 || altnum == 2 {
			return "OR"
		}
		return "XOR"
	case opcode == 0x4F: //NOT
		return "NOT"
	case opcode-0xD0 <= 0xE: //INC
		return "INC"
	case opcode-0xE0 <= 0xE: //DEC
		return "DEC"
	case opcode == 0x03: //LSR
		return "LSR"
	case opcode == 0x04: //ROL
		return "ROL"
	case opcode == 0x96: //ASR -signed shift
		if altnum == 1 {
			return "DIV2"
		}
		return "ASR"
	case opcode == 0x97: //ROR
		return "ROR"
	case opcode == 0x4D: //SWAP
		return "SWAP"
	case opcode == 0x95: //SEX
		return "SEX"
	case opcode == 0x9E: //LOB
		return "LOB"
	case opcode == 0x9F: //FMULT/LMULT
		if altnum == 1 {
			return "LMULT"
		}
		return "FMULT"
	case opcodeHn == 0x80: //MULT/UMULT
		if altnum == 0 || altnum == 2 {
			return "MULT"
		}
		return "UMULT"
	case opcode-0x98 <= 5: //JMP/LJMP
		if altnum == 1 {
			return "LJMP"
		}
		return "JMP"
	case opcode == 0x3C: //LOOP
		return "LOOP"
	case opcode-0x91 <= 3: //LINK/RETURN TO
		return "LINK"
	case opcode == 0x3D: //ALT1
		return "ALT1"
	case opcode == 0x3E: //ALT2
		return "ALT2"
	case opcode == 0x3F: //ALT3
		return "ALT3"
	case opcodeHn == 0x10: //TO
		return "TO/MOVE"
	case opcodeHn == 0xB0: //FROM
		return "FROM/MOVES"
	case opcodeHn == 0x20: //WITH
		return "WITH"
	case opcode == 0x00: //STOP
		return "STOP"
	case opcode == 0x01: //NOP
		return "NOP"
	case opcode == 0x02: //CACHE
		return "CACHE"
	case opcode == 0x4C: //PLOT??
		if altnum == 1 {
			return "RPIX"
		}
		return "PLOT"
	default:
		panic(fmt.Sprintf("GSU: unknown opcode: $%02x", opcode))
	}
	return ""
}

func (gsu *GSU) Step() uint64 {
	if gsu.r.SFR&FlagGo == 0 {
		return constants.CYCLE_2
	}

	gsu.processByte()
	gsu.preFetchByte()
	return constants.CYCLE_2
}

// the gsu is execute -> fetch.
// TODO prefetch determines cycle cost
func (gsu *GSU) preFetchByte() {
	pc := gsu.r.cpuRegisters[0xF]
	var opcode byte
	var err error

	if idx := pc - gsu.r.CBR; idx < 0x200 {
		cacheMask := uint32((1 << (idx >> 4)))
		if gsu.cacheFlags&cacheMask == 0 {
			rowBaseIdx := idx & 0x1F0
			rowBasePc := pc & 0xFFF0
			for i := range uint16(16) {
				//TODO this read has to add cumulative overhead
				opcode, err = gsu.Read8(gsu.r.PBR, rowBasePc+i)

				if err != nil {
					panic(err.Error())
				}

				gsu.cache[rowBaseIdx+i] = opcode
			}
			gsu.cacheFlags |= cacheMask
		}
		opcode = gsu.cache[idx]
	} else {
		opcode, err = gsu.Read8(gsu.r.PBR, pc)

		if err != nil {
			panic(err.Error())
		}
	}
	gsu.currentOpcode = opcode
	if pcVal := gsu.r.cpuRegister15Buffer; pcVal != R15_NOT_BRANCHING {
		gsu.r.cpuRegisters[0xF] = uint16(pcVal)
		gsu.r.cpuRegister15Buffer = R15_NOT_BRANCHING
	} else {
		gsu.r.cpuRegisters[0xF]++
	}
	//fmt.Printf("%02x\n", opcode)
}

func (gsu *GSU) GetRegisterMap() coprocessor.RegisterMap {
	return coprocessor.RegisterMap{Start: 0x3000, End: 0x347F, Name: "GSU"}
}

func (gsu *GSU) SetCartridge(cartridge coprocessor.CartridgeDataSource) {
	gsu.cartridge = cartridge
}

// every coprocessor carries its own mapper
// which then it can use to get data using the cartridge data source
func (gsu *GSU) Read8(bank byte, offset uint16) (byte, error) {
	if bank < 0x40 {
		offset = (offset & 0x7FFF) | (uint16(bank&1) << 15)
		return gsu.cartridge.ReadRom(int(bank>>1)<<16 | int(offset)), nil //lorom
	}
	if bank-0x40 < 0x20 { //0x40-0x5F
		return gsu.cartridge.ReadRom(int(bank&0x3F)<<16 | int(offset)), nil //hirom
	}
	if bank-0x70 < 2 {
		return gsu.cartridge.ReadRam(int(bank&1)<<16 | int(offset)), nil
	}
	return 0, fmt.Errorf("GSU: Trying to read unmapped memory"+
		" at $%02x%04x", bank, offset)
}

func (gsu *GSU) Write8(bank byte, offset uint16, value byte) error {
	if bank-0x70 < 2 {
		gsu.cartridge.WriteRam(int(bank&1)<<16|int(offset), value)
		return nil
	}
	return fmt.Errorf("GSU: Trying to write unmapped or read only memory"+
		" at $%02x%04x", bank, offset)
}
