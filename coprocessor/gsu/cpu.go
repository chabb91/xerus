package gsu

import (
	"SNES_emulator/coprocessor"
	"SNES_emulator/internal/constants"
	"fmt"
)

type ExecutionState uint16

const (
	goState   ExecutionState = 1 << 5
	stopState ExecutionState = 0
)

type GSU struct {
	cartridge coprocessor.CartridgeDataSource

	r registers

	cache [0x200]byte
}

/*
in a byte sequence in gsuadc.sfc: 21 3d 50:
21 means WITH register 1 aka select r1 as rn and rs and set B=1
3d means select ALT1
50 is ADD but with 3d it becomes:  3D 5n         2 000vscz ADC Rn       adc Rd,Rs,Rn ;Rd=Rs+Rn+Cy
read it together: r1 = r1 + r0 + carry
my brain hurts
*/

func (gsu *GSU) Step() uint64 {
	if gsu.r.executionState == stopState {
		return constants.CYCLE_2
	}
	val, _ := gsu.Read8(gsu.r.PBR, gsu.r.cpuRegisters[0xF])
	gsu.r.cpuRegisters[0xF]++
	if val == 0x00 {
		fmt.Println("STOPPING")
		gsu.r.executionState = stopState
	}
	fmt.Printf("%02x\n", val)
	return constants.CYCLE_2
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
