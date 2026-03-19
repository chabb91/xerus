package gsu

import (
	"errors"
	"fmt"
)

const (
	FlagZ uint16 = 1 << 1 //Zero			(0=NotZero/NotEqual, 1=Zero/Equal)
	FlagC uint16 = 1 << 2 //Carry			(0=Borrow/Carry, 1=Carry/NoBorrow)
	FlagS uint16 = 1 << 3 //Sign			(0=Positive, 1=Negative)
	FlagV uint16 = 1 << 4 //OverFlow		(0=NoOverFlow, 1=OverFlow)
)

const (
	RAN byte = 1 << 3 //Game Pak RAM bus access (0=SNES, 1=GSU) if cleared while GO=1 the GSU enters WAIT
	RON byte = 1 << 4 //Game Pak ROM bus access (0=SNES, 1=GSU) if cleared while GO=1 the GSU enters WAIT
)

type registers struct {
	cpuRegisterByteLatch byte
	cpuRegisters         [16]uint16

	executionState ExecutionState

	SFR   uint16 //status flag register
	PBR   byte   //program bank register
	ROMBR byte   //game pak ROM bank register
	RAMBR byte   // 1 bit bank 70 or 71 game pak RAM bank register
	CBR   uint16 // cache base register. 12 bit, lower 4 bits unused
	BRAMR byte   //back up RAM register. 1 bit
	VCR   byte   // version code register 1 = MC1 4 = GSU2 the rest unknown??
	CFGR  byte   //config register
	CLSR  byte   //clock select register 0=10mhz, 1=21mhz
	SCBR  byte   //screen base register
	SCMR  byte   //screen mode register
	COLR  byte   //color register
	POR   byte   //plot option register
	//rom buffer prefetch bytes at rombr:r14??
	//sreg/dreg //memorized to/from prefix selections??
}

// set cpu register 0-15 as idx<<1 where the lsb signifies LSB or MSB
// where even addresses LATCH, odd addresses SET.
func (r *registers) setCpuRegister(byteIdx, value byte) {
	if byteIdx&1 == 0 {
		r.cpuRegisterByteLatch = value
	} else {
		wordIdx := byteIdx >> 1
		r.cpuRegisters[wordIdx] = uint16(r.cpuRegisterByteLatch) | uint16(value)<<8
		if wordIdx == 0xF {
			r.executionState = goState
		}
	}
}

func (r *registers) getCpuRegister(byteIdx byte) byte {
	if byteIdx&1 == 0 {
		return byte(r.cpuRegisters[byteIdx>>1])
	} else {
		return byte(r.cpuRegisters[byteIdx>>1] >> 8)
	}
}

func (gsu *GSU) Read(addr uint16) (byte, error) {
	if byteIdx := addr - 0x3000; byteIdx < 0x20 {
		return gsu.r.getCpuRegister(byte(byteIdx)), nil
	}
	if addr == 0x3030 {
		fmt.Println("WAITING FOR GO")
		//TODO no idea what rom[r14] Read is on bit 6
		return byte(gsu.r.SFR) | byte(gsu.r.executionState), nil
	}
	if addr == 0x3039 {
		fmt.Println("CLS: ")
	}
	if addr == 0x3037 {
		fmt.Println("CFGR: ")
	}
	if addr == 0x3038 {
		fmt.Println("SBCR: ")
	}
	if addr == 0x3034 {
		return gsu.r.PBR, nil
	}
	if addr == 0x3036 {
		return gsu.r.ROMBR, nil
	}
	if addr == 0x303C {
		return gsu.r.RAMBR, nil
	}
	if addr == 0x303A {
		//return gsu.r.SCMR & 0x7F, nil
	}
	return 0, errors.New("GSU CONNECTED UHOH")
}

func (gsu *GSU) Write(addr uint16, value byte) error {
	if byteIdx := addr - 0x3000; byteIdx < 0x20 {
		gsu.r.setCpuRegister(byte(byteIdx), value)
		return nil
	}
	if addr == 0x3030 {
		fmt.Println("SETTING GO")
		gsu.r.executionState = ExecutionState(value) & goState
		gsu.r.SFR = (gsu.r.SFR)&0xFF00 | (uint16(value & 0x1E))
	}
	if addr == 0x3039 {
		fmt.Println("CLS: ", value)
	}
	if addr == 0x3037 {
		fmt.Println("CFGR: ", value)
	}
	if addr == 0x3038 {
		fmt.Println("SBCR: ", value)
	}
	if addr == 0x3034 {
		gsu.r.PBR = value
		return nil
	}
	if addr == 0x3036 {
		gsu.r.ROMBR = value
		return nil
	}
	if addr == 0x303C {
		gsu.r.RAMBR = value
		return nil
	}
	if addr == 0x303A {
		//TODO theres some bus contention with this. if ran is 0 and gsu tries to access ram it enters wait state
		gsu.r.SCMR = value & 0x7F
		return nil
	}
	return errors.New("GSU CONNECTED UHOH")
}
