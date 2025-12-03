package apu

const PSRAM_SIZE = 0x10000

type Memory interface {
	Read8(addr uint16) byte
	Write8(addr uint16, val byte)
}

type SPCMemory struct {
	ram [PSRAM_SIZE]byte
}

// TODO implement register handling
// just a placeholder for now
func (s *SPCMemory) Read8(addr uint16) byte {
	switch {
	case addr < 0xF0:
	case addr >= 0xF0 && addr <= 0xF1:
	case addr == 0xF3:
	case addr >= 0xF4 && addr <= 0xF7:
	case addr >= 0xF8 && addr <= 0xF9:
	case addr >= 0xFA && addr <= 0xFF:
	case addr >= 0xFFC0:
	default:
	}
	return s.ram[addr]
}

func (s *SPCMemory) Write8(addr uint16, val byte) {
	switch {
	case addr < 0xF0:
	case addr == 0xF1:
	case addr == 0xF2:
	case addr == 0xF3:
	case addr >= 0xF4 && addr <= 0xF7:
	case addr >= 0xFA && addr <= 0xFF:
	case addr >= 0xFFC0:
	default:
		s.ram[addr] = val
	}
}
