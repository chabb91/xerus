package cartridge

import (
	"testing"
)

func TestLoROM(t *testing.T) {
	romData, err := Load("/home/chabb/Downloads/CPUADC.sfc")
	if err != nil {
		t.Fatalf("failed to read rom")
	}
	cart := NewCartridge(romData, NewLoRom())
	val1, _ := cart.ReadByte(0x80, 0xFFFC)
	val2, _ := cart.ReadByte(0x80, 0xFFFD)

	if val1 != romData[cart.Mapper.getHeaderLocation()+0x3C] || val2 != romData[cart.Mapper.getHeaderLocation()+0x3D] {
		t.Errorf("RESET VECTOR ISNT MAPPED RIGHT! Got: %v, %v.", val1, val2)
	}
}

func TestHiROM(t *testing.T) {
	romData, err := Load("/home/chabb/Downloads/hvdma_max.sfc")
	if err != nil {
		t.Fatalf("failed to read rom")
	}
	cart := NewCartridge(romData, NewHiRom())
	val1, _ := cart.ReadByte(0x00, 0xFE18)
	val2, _ := cart.ReadByte(0xC0, 0xFE18)
	if val1 != val2 {
		t.Errorf("MIRROR ISNT MAPPED RIGHT! Got: %v, %v.", val1, val2)
	}
	val1, _ = cart.ReadByte(0x80, 0xFFFC)
	val2, _ = cart.ReadByte(0x80, 0xFFFD)

	if val1 != romData[cart.Mapper.getHeaderLocation()+0x3C] || val2 != romData[cart.Mapper.getHeaderLocation()+0x3D] {
		t.Errorf("RESET VECTOR ISNT MAPPED RIGHT! Got: %v, %v.", val1, val2)
	}
}
