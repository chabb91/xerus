package gsu

import (
	"fmt"
	"os"
	"strings"
)

type tracer struct {
	stopCnt     int
	maxLines    int
	currentLine int
	prevPc      uint16

	file *os.File
}

func newTracer(maxLines, stopCnt int) *tracer {
	f, err := os.Create("gsu_loop_dump.txt")
	if err != nil {
		panic(err)
	}
	return &tracer{
		file:     f,
		maxLines: maxLines,
		stopCnt:  stopCnt,
	}
}

func (t *tracer) trace(gsu *GSU) {
	if t.stopCnt <= 0 {
		var instructionCode string
		if gsu.r.getImmediateNum() == 0 {
			instructionCode = parseOpcode(gsu.currentOpcode, gsu.r.getAltNum()>>8)
		} else {
			instructionCode = "imm"
		}
		fmt.Fprintf(t.file, "%s (%02X) R: %v SFR: %s dr: %d sr: %d", instructionCode, gsu.currentOpcode,
			gsu.r.cpuRegisters, sfrToString(gsu.r.SFR), gsu.dReg, gsu.sReg)
		if gsu.r.cpuRegisters[0xF]-t.prevPc != 1 {
			fmt.Fprintf(t.file, " JUMP \n")
		} else {
			fmt.Fprintf(t.file, "\n")
		}
		t.prevPc = gsu.r.cpuRegisters[0xF]
		t.currentLine++
		if t.currentLine == t.maxLines {
			defer t.file.Close()

			t.file.Sync()
			panic("GSU: trace collected. exiting.")
		}
	} else {
		if gsu.currentOpcode == 0x00 && gsu.r.getImmediateNum() == 0 {
			t.stopCnt--
		}
	}
}

func sfrToString(sfr uint16) string {
	var sb strings.Builder

	format := func(bit uint16, char string) {
		if sfr&bit != 0 {
			sb.WriteString(strings.ToUpper(char))
		} else {
			sb.WriteString(strings.ToLower(char))
		}
	}

	format(FlagIrq, "i")
	sb.WriteString("-")
	sb.WriteString("-")
	format(FlagB, "b")
	format(FlagIh, "i2")
	format(FlagIl, "i1")
	format(FlagAlt2, "a2")
	format(FlagAlt1, "a1")
	sb.WriteString("-")
	format(FlagR, "r")
	format(FlagGo, "g")
	format(FlagV, "v")
	format(FlagS, "s")
	format(FlagC, "c")
	format(FlagZ, "z")
	sb.WriteString("-")

	return sb.String()
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
