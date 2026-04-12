package gsu

import (
	"fmt"
)

const (
	FlagZ    uint16 = 1 << 1  //Zero			(0=NotZero/NotEqual, 1=Zero/Equal)
	FlagC    uint16 = 1 << 2  //Carry			(0=Borrow/Carry, 1=Carry/NoBorrow)
	FlagS    uint16 = 1 << 3  //Sign			(0=Positive, 1=Negative)
	FlagV    uint16 = 1 << 4  //OverFlow		(0=NoOverFlow, 1=OverFlow)
	FlagGo   uint16 = 1 << 5  //GSU is running on 1, stopped on 0
	FlagR    uint16 = 1 << 6  //ROM[R14] Read (0=No, 1=Reading ROM via R14 address) ??
	FlagAlt1 uint16 = 1 << 8  //ALT1, ALT2, ALT3 prefixes
	FlagAlt2 uint16 = 1 << 9  //ALT1, ALT2, ALT3 prefixes
	FlagIl   uint16 = 1 << 10 //counter for opcodes with immediate operands (low)
	FlagIh   uint16 = 1 << 11 //counter for opcodes with immediate operands (High)
	FlagB    uint16 = 1 << 12 //B prefix (used by MOVE/MOVES)
	FlagIrq  uint16 = 1 << 15 //Interrupt Flag (reset on read, set on STOP) (also set if IRQ masked?)
	FlagAlt3        = FlagAlt1 | FlagAlt2
)

const ( //SCMR
	MD0 byte = 1 << 0
	MD1 byte = 1 << 1 //Color Gradient bits (bpp) (2/4/4/8)
	HT0 byte = 1 << 2 //Screen Height (LSB)
	RAN byte = 1 << 3 //Game Pak RAM bus access (0=SNES, 1=GSU) if cleared while GO=1 the GSU enters WAIT
	RON byte = 1 << 4 //Game Pak ROM bus access (0=SNES, 1=GSU) if cleared while GO=1 the GSU enters WAIT
	HT1 byte = 1 << 5 //Screen Height (MSB)
)

const CFGR_MS0 byte = 1 << 5 //Multiplier Speed Select (0=Standard, 1=High Speed Mode) (CFGR)

const ( //POR
	FlagPlotTransparent byte = 1 << 0 //0= Do Not Plot Color 0, 1= Plot Color 0
	FlagPlotDither      byte = 1 << 1 //0= Normal, 1= Dither (4/16 color mode only)
	FlagColorHighNibble byte = 1 << 2 //0= Normal, 1= Replace incoming LSB by incoming MSB
	FlagColorFreezeHigh byte = 1 << 3 //0= Normal, 1= Write-protect COLOR.MSB
	FlagForceObjMode    byte = 1 << 4 //0= Normal, 1= Force OBJ mode; ignore SCMR.HT0/HT1
)

const R15_NOT_BRANCHING int = -1

type registers struct {
	fetchFunc func()

	cpuRegisterByteLatch byte
	cpuRegisters         [16]uint16
	cpuRegister15Buffer  int //after a branch is taken, it has to be detected and pc not incremented.

	romAddrPtr  byte
	r14Modified bool

	SFR   uint16 //status flag register
	PBR   byte   //program bank register
	ROMBR byte   //game pak ROM bank register
	RAMBR byte   // 1 bit bank 70 or 71 game pak RAM bank register
	CBR   uint16 // cache base register. 12 bit, lower 4 bits unused
	BRAMR byte   //back up RAM register. 1 bit
	VCR   byte   // version code register 1 = MC1 4 = GSU2 the rest unknown??
	CFGR  byte   //config register
	CLSR  byte   //clock select register 0=10mhz, 1=21mhz
	SCBR  uint32 //screen base register
	SCMR  byte   //screen mode register
	COLR  byte   //color register
	POR   byte   //plot option register
	//rom buffer prefetch bytes at rombr:r14??
	//sreg/dreg //memorized to/from prefix selections??
}

func (r *registers) writeCpuRegister(idx byte, val uint16) {
	if idx == 15 {
		r.cpuRegister15Buffer = int(val)
		return
	}
	if idx == 14 {
		r.r14Modified = true
	}
	r.cpuRegisters[idx] = val
}

// returns values that can be directly compared to ALT flags
func (r *registers) getAltNum() uint16 {
	return r.SFR & (FlagAlt1 | FlagAlt2)
}

func (r *registers) getImmediateNum() uint16 {
	return (r.SFR & (FlagIl | FlagIh)) >> 10
}

func (r *registers) setImmediateNum(num uint16) {
	r.SFR &= ^(FlagIl | FlagIh)
	r.SFR |= (num & 3) << 10
}

func (r *registers) hasFlag(flag uint16) bool {
	return r.SFR&flag != 0
}

func (r *registers) setFlag(flag uint16, cond bool) {
	if cond {
		r.SFR |= flag
	} else {
		r.SFR &= ^flag
	}
}
func (r *registers) setColr(value byte) {
	if r.POR&FlagColorHighNibble != 0 {
		r.COLR = r.COLR&0xF0 | value>>4
		return
	}
	if r.POR&FlagColorFreezeHigh != 0 {
		r.COLR = r.COLR&0xF0 | value&0xF
		return
	}
	r.COLR = value
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
			r.SFR |= FlagGo
			r.SFR &= ^(FlagIl | FlagIh) //if it was aborted before we might be stuck in immediate mode
			r.cpuRegister15Buffer = R15_NOT_BRANCHING
			r.fetchFunc()
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
	//fmt.Printf("GSU: READING ADDR $%04x\n", addr)
	if cacheIdx := addr - 0x3100; cacheIdx < 0x200 {
		idx := (gsu.r.CBR + cacheIdx) & 0x1FF
		return gsu.cache[idx], nil
	}
	if byteIdx := addr - 0x3000; byteIdx < 0x20 {
		return gsu.r.getCpuRegister(byte(byteIdx)), nil
	}
	if addr == 0x3030 {
		return byte(gsu.r.SFR), nil
	}
	if addr == 0x3031 {
		tmp := byte(gsu.r.SFR >> 8)

		if gsu.r.CFGR&0x80 == 0 && gsu.r.SFR&FlagIrq != 0 {
			gsu.interruptManager.CartAcknowledgeIrq()
		}

		gsu.r.SFR &= ^FlagIrq //reading clears the irq bit??

		return tmp, nil
	}
	if addr == 0x3034 {
		return gsu.r.PBR, nil
	}
	if addr == 0x3036 {
		return gsu.r.ROMBR, nil
	}
	if addr == 0x3039 {
		return gsu.r.CLSR, nil
	}
	if addr == 0x3037 {
		return gsu.r.CFGR, nil
	}
	if addr == 0x303B {
		return gsu.r.VCR, nil
	}
	if addr == 0x303C {
		return gsu.r.RAMBR, nil
	}
	if addr == 0x303E {
		return byte(gsu.r.CBR), nil
	}
	if addr == 0x303F {
		return byte(gsu.r.CBR >> 8), nil
	}
	return 0, fmt.Errorf("GSU: invalid register read at $%04X", addr)
}

func (gsu *GSU) Write(addr uint16, value byte) error {
	//fmt.Printf("GSU: WRITING ADDR $%04x, %d\n", addr, value)
	if cacheIdx := addr - 0x3100; cacheIdx < 0x200 {
		idx := (gsu.r.CBR + cacheIdx) & 0x1FF
		gsu.cache[idx] = value
		if idx&0xF == 0xF {
			gsu.cacheFlags |= uint32(1 << (idx >> 4))
		}
		return nil
	}
	if byteIdx := addr - 0x3000; byteIdx < 0x20 {
		gsu.r.setCpuRegister(byte(byteIdx), value)
		return nil
	}
	if addr == 0x3030 {
		prevGo := gsu.r.SFR&FlagGo != 0
		gsu.r.SFR = (gsu.r.SFR)&0xFF00 | (uint16(value & 0x1E))
		if gsu.r.SFR&FlagGo == 0 && prevGo {
			gsu.r.CBR = 0
			gsu.cacheFlags = 0
		}
		return nil
	}
	if addr == 0x3031 {
		gsu.r.SFR = (gsu.r.SFR)&0x00FF | (uint16(value) << 8)
		return nil
	}
	if addr == 0x3033 {
		gsu.r.BRAMR = value & 1
		return nil
	}
	if addr == 0x3034 {
		gsu.r.PBR = value & 0x7F
		return nil
	}
	if addr == 0x3037 {
		gsu.r.CFGR = value
		return nil
	}
	if addr == 0x3038 {
		gsu.r.SCBR = 0x70_0000 + uint32(value)<<10
		return nil
	}
	if addr == 0x3039 {
		gsu.r.CLSR = value & 1
		gsu.setAccessTime(value)
		return nil
	}
	if addr == 0x303A {
		gsu.r.SCMR = value & 0x7F
		gsu.updateWait(value)
		return nil
	}
	return fmt.Errorf("GSU: invalid register write at $%04X", addr)
}
