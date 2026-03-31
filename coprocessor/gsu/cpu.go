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

	cache [0x200]byte

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
	val, err := gsu.Read8(gsu.r.PBR, gsu.r.cpuRegisters[0xF])

	if err != nil {
		panic(err.Error())
	}
	gsu.currentOpcode = val
	if pcVal := gsu.r.cpuRegister15Buffer; pcVal != R15_NOT_BRANCHING {
		gsu.r.cpuRegisters[0xF] = uint16(pcVal)
		gsu.r.cpuRegister15Buffer = R15_NOT_BRANCHING
	} else {
		gsu.r.cpuRegisters[0xF]++
	}
	fmt.Printf("%02x\n", val)
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
