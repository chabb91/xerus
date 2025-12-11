package apu

const PSRAM_SIZE = 0x10000

var iplRom = [64]byte{0xCD, 0xEF, 0xBD, 0xE8, 0x00, 0xC6, 0x1D, 0xD0, 0xFC, 0x8F, 0xAA, 0xF4, 0x8F, 0xBB, 0xF5, 0x78,
	0xCC, 0xF4, 0xD0, 0xFB, 0x2F, 0x19, 0xEB, 0xF4, 0xD0, 0xFC, 0x7E, 0xF4, 0xD0, 0x0B, 0xE4, 0xF5,
	0xCB, 0xF4, 0xD7, 0x00, 0xFC, 0xD0, 0xF3, 0xAB, 0x01, 0x10, 0xEF, 0x7E, 0xF4, 0x10, 0xEB, 0xBA,
	0xF6, 0xDA, 0x00, 0xBA, 0xF4, 0xC4, 0xF4, 0xDD, 0x5D, 0xD0, 0xDB, 0x1F, 0x00, 0x00, 0xC0, 0xFF}

type Memory interface {
	Read8(addr uint16) byte
	Write8(addr uint16, val byte)
}

type SPCMemory struct {
	ram [PSRAM_SIZE]byte

	test    byte
	control byte
	dspAddr byte //fake it till you make it
	dspRegs [128]byte
	ports   [4]IOPort
	Timers  *[3]*Timer
}

func NewSPCMemory() *SPCMemory {
	ret := &SPCMemory{}
	ret.test = 0xA
	ret.control = 0xB0

	return ret
}

type IOPort struct {
	fromCPU   byte
	towardCPU byte
}

/*
type Timer struct {
	counter byte //s1
	divider byte //s2
	output  byte //s3
	target  byte
	enabled bool
}
*/

func (s *SPCMemory) Read8(addr uint16) byte {
	switch {
	case addr >= 0xF0 && addr <= 0xF1:
		return 0
	case addr == 0xF2:
		return s.dspAddr
	case addr == 0xF3:
		return s.dspRegs[s.dspAddr&0x7F]
	case addr >= 0xF4 && addr <= 0xF7:
		return s.ports[addr-0xF4].fromCPU
	case addr >= 0xF8 && addr <= 0xF9:
		return s.ram[addr]
	case addr >= 0xFA && addr <= 0xFC:
		return 0
	case addr >= 0xFD && addr <= 0xFF:
		idx := addr - 0xFD
		ret := s.Timers[idx].output & 0xF
		s.Timers[idx].output = 0
		return ret
	case addr >= 0xFFC0:
		if s.control >= 0x80 {
			return iplRom[addr-0xFFC0]
		} else {
			return s.ram[addr]
		}
	default:
		return s.ram[addr]
	}
}

func (s *SPCMemory) Write8(addr uint16, val byte) {
	s.ram[addr] = val
	switch {
	case addr == 0xF0:
		s.test = val
	case addr == 0xF1:
		if val&0x10 != 0 {
			s.ports[0].fromCPU = 0
			s.ports[1].fromCPU = 0
		}
		if val&0x20 != 0 {
			s.ports[2].fromCPU = 0
			s.ports[3].fromCPU = 0
		}
		s.Timers[0].enabled = val&1 != 0
		s.Timers[1].enabled = val&2 != 0
		s.Timers[2].enabled = val&4 != 0

		if s.control&1 == 0 && val&1 != 0 {
			s.Timers[0].stage2Counter = 0
			s.Timers[0].output = 0
		}
		if s.control&2 == 0 && val&2 != 0 {
			s.Timers[1].stage2Counter = 0
			s.Timers[1].output = 0
		}
		if s.control&4 == 0 && val&4 != 0 {
			s.Timers[2].stage2Counter = 0
			s.Timers[2].output = 0
		}

		s.control = val
	case addr == 0xF2:
		s.dspAddr = val
	case addr == 0xF3:
		if s.dspAddr <= 0x7F {
			s.dspRegs[s.dspAddr] = val
		}
	case addr >= 0xF4 && addr <= 0xF7:
		s.ports[addr-0xF4].towardCPU = val
	case addr >= 0xFA && addr <= 0xFC:
		s.Timers[addr-0xFA].target = val
	}
}

// cpu side
func (s *SPCMemory) Read(addr uint16) (byte, error) {
	ioNum := (addr - 0x2140) & 3
	return s.ports[ioNum].towardCPU, nil
}
func (s *SPCMemory) Write(addr uint16, value byte) error {
	ioNum := (addr - 0x2140) & 3
	s.ports[ioNum].fromCPU = value
	return nil
}
