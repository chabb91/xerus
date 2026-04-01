package gsu

import (
	"SNES_emulator/coprocessor"
	"SNES_emulator/internal/constants"
	"fmt"
)

type immediateInstructionFunc func(gsu *GSU)

const SRAM_BASE_BANK byte = 0x70

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
}

func New() coprocessor.Coprocessor {
	gsu := &GSU{}
	gsu.r.gsu = gsu
	gsu.r.cpuRegister15Buffer = R15_NOT_BRANCHING

	return gsu
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
		if gsu.cacheFlags&cacheMask != 0 {
			opcode = gsu.cache[idx]
		} else {
			opcode, err = gsu.Read8(gsu.r.PBR, pc)

			if pcVal := gsu.r.cpuRegister15Buffer; pcVal != R15_NOT_BRANCHING {
				for i := range uint16(16) {
					v, _ := gsu.Read8(gsu.r.PBR, (pc&0xFFF0)+i)
					gsu.cache[(idx&0x1F0)+i] = v
				}
				gsu.cacheFlags |= cacheMask
			}

			if idx := uint16(gsu.r.cpuRegister15Buffer) - gsu.r.CBR; idx < 0x200 {
				//panic("gotta fill this in")
			}

			if err != nil {
				panic(err.Error())
			}

			gsu.cache[idx] = opcode
			if idx&0xF == 0xF {
				gsu.cacheFlags |= cacheMask
			}
		}
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
	fmt.Printf("%02x\n", opcode)
}

func (gsu *GSU) fillCacheOnBranch(idx, currentPc uint16) {
	cacheMask := uint32((1 << (idx >> 4)))
	lineFillCnt := min((idx&0xF)^0xF, uint16(gsu.r.cpuRegister15Buffer)-currentPc+1)
	for i := uint16(1); i <= lineFillCnt; i++ {
		val, _ := gsu.Read8(gsu.r.PBR, currentPc+i)
		gsu.cache[idx+i] = val
		if (idx+i)&0xF == 0xF {
			gsu.cacheFlags |= cacheMask
		}
	}
	if idx := uint16(gsu.r.cpuRegister15Buffer) - gsu.r.CBR; idx < 0x200 {
		cacheMask = uint32((1 << (idx >> 4)))
		if gsu.cacheFlags&cacheMask != 0 {
			lineFillCnt = (idx & 0xF) ^ 0xF
			for i := uint16(0); i < lineFillCnt; i++ {
				val, _ := gsu.Read8(gsu.r.PBR, uint16(gsu.r.cpuRegister15Buffer)+i)
				gsu.cache[idx+i] = val
				if (idx+i)&0xF == 0xF {
					gsu.cacheFlags |= cacheMask
				}
			}
		}
	}
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
	//TODO implement code cache
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
	//only the code cache and the game pak ram are writeable
	//TODO implement code cache
	if bank-0x70 < 2 {
		gsu.cartridge.WriteRam(int(bank&1)<<16|int(offset), value)
		return nil
	}
	return fmt.Errorf("GSU: Trying to write unmapped or read only memory"+
		" at $%02x%04x", bank, offset)
}
