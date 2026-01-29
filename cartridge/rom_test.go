package cartridge

import (
	"testing"
)

func TestLoROM(t *testing.T) {
	loRomHeader := 0x7FC0
	cart := NewCartridge("/home/chabb/Downloads/CPUADC.sfc")
	val1, _ := cart.ReadByte(0x80, 0xFFFC)
	val2, _ := cart.ReadByte(0x80, 0xFFFD)

	if val1 != cart.romData[loRomHeader+0x3C] || val2 != cart.romData[loRomHeader+0x3D] {
		t.Errorf("RESET VECTOR ISNT MAPPED RIGHT! Got: %v, %v.", val1, val2)
	}
}

func TestHiROM(t *testing.T) {
	hiRomHeader := 0xFFC0
	cart := NewCartridge("/home/chabb/Downloads/hvdma_max.sfc")
	val1, _ := cart.ReadByte(0x00, 0xFE18)
	val2, _ := cart.ReadByte(0xC0, 0xFE18)
	if val1 != val2 {
		t.Errorf("MIRROR ISNT MAPPED RIGHT! Got: %v, %v.", val1, val2)
	}
	val1, _ = cart.ReadByte(0x80, 0xFFFC)
	val2, _ = cart.ReadByte(0x80, 0xFFFD)

	if val1 != cart.romData[hiRomHeader+0x3C] || val2 != cart.romData[hiRomHeader+0x3D] {
		t.Errorf("RESET VECTOR ISNT MAPPED RIGHT! Got: %v, %v.", val1, val2)
	}
}
