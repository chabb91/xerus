package gsu

import (
	"SNES_emulator/coprocessor"
	"errors"
	"fmt"
)

func (gsu *GSU) GetRegisterMap() coprocessor.RegisterMap {
	return coprocessor.RegisterMap{Start: 0x3000, End: 0x347F, Name: "GSU"}
}

func (gsu *GSU) Read(addr uint16) (byte, error) {
	if addr == 0x3030 {
		fmt.Println("WAITING FOR GO")
		//return 1 << 5, nil
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
		fmt.Println("PBR: ")
	}
	if addr == 0x3036 {
		fmt.Println("ROMBR: ")
	}
	if addr == 0x303C {
		fmt.Println("RAMBR: ")
	}
	if addr == 0x303A {
		fmt.Println("SCMR: ")
	}
	if addr == 0x301E {
		fmt.Println("R15L: ")
	}
	if addr == 0x301F {
		fmt.Println("R15H: ")
	}
	return 0, errors.New("GSU CONNECTED UHOH")
}

func (gsu *GSU) Write(addr uint16, value byte) error {
	if addr == 0x3030 {
		fmt.Println("SETTING GO")
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
		fmt.Println("PBR: ", value)
	}
	if addr == 0x3036 {
		fmt.Println("ROMBR: ", value)
	}
	if addr == 0x303C {
		fmt.Println("RAMBR: ", value)
	}
	if addr == 0x303A {
		fmt.Println("SCMR: ", value)
	}
	if addr == 0x301E {
		fmt.Println("R15L: ", value)
	}
	if addr == 0x301F {
		fmt.Println("R15H: ", value)
	}
	return errors.New("GSU CONNECTED UHOH")
}

func (gsu *GSU) SetCartridge(cartridge coprocessor.CartridgeDataSource) {
	gsu.cartridge = cartridge
}
