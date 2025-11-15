package memory

import (
	"fmt"
)

type RegisterHandler interface {
	Read(addr uint16) (byte, error)
	Write(addr uint16, value byte) error
}

// RegisterRange defines a contiguous range of registers
type RegisterRange struct {
	Start   uint16
	End     uint16
	Handler RegisterHandler
	Name    string // For debugging
}

type RegisterSystem struct {
	ranges []RegisterRange
}

func NewRegisterSystem() *RegisterSystem {
	return &RegisterSystem{
		ranges: make([]RegisterRange, 0),
	}
}

func (rs *RegisterSystem) RegisterRange(start, end uint16, handler RegisterHandler, name string) {
	rs.ranges = append(rs.ranges, RegisterRange{
		Start:   start,
		End:     end,
		Handler: handler,
		Name:    name,
	})
}

func (rs *RegisterSystem) FindHandler(addr uint16) (RegisterHandler, string, error) {
	for _, r := range rs.ranges {
		if addr >= r.Start && addr <= r.End {
			return r.Handler, r.Name, nil
		}
	}
	return nil, "", fmt.Errorf("no handler for address $%04X", addr)
}

func (rs *RegisterSystem) IsRegisterAddress(bank byte, addr uint16) bool {
	if addr >= 0x2000 && addr <= 0x5FFF {
		return (bank <= 0x3F) || (bank >= 0x80 && bank <= 0xBF)
	}
	return false
}

func SetupRegisterSystem(bus *RealBus) {
	bus.registers = NewRegisterSystem()

	bus.RegisterRange(0x2180, 0x2183, newWramDataRW(bus.WRAM[:]), "WRAM")

	//bus.registers.RegisterRange(0x2100, 0x213F, ppuHandler, "PPU")
	//bus.registers.RegisterRange(0x2140, 0x2143, apuHandler, "APU")
	//bus.registers.RegisterRange(0x4000, 0x41FF, joypadHandler, "Controllers")
	//bus.registers.RegisterRange(0x4200, 0x44FF, cpuHandler, "CPU Registers")
}

type WramDataRW struct {
	WRAM    []byte
	address uint32
}

func newWramDataRW(WRAM []byte) *WramDataRW {
	return &WramDataRW{WRAM: WRAM}
}

func (wd *WramDataRW) Read(addr uint16) (byte, error) {
	if addr == 0x2180 {
		result := wd.WRAM[wd.address]
		wd.address++
		wd.address &= 0x1FFFF
		return result, nil
	}
	return 0, fmt.Errorf("invalid internal WRAM register read at $%04X", addr)
}

func (wd *WramDataRW) Write(addr uint16, value byte) error {
	switch addr {
	case 0x2180:
		wd.WRAM[wd.address] = value
		wd.address++
		wd.address &= 0x1FFFF
	case 0x2181:
		wd.address = (wd.address & 0x1FF00) | uint32(value)
	case 0x2182:
		wd.address = (wd.address & 0x100FF) | uint32(value)<<8
	case 0x2183:
		wd.address = (wd.address & 0x0FFFF) | uint32(value&1)<<16
	default:
		return fmt.Errorf("invalid internal WRAM register write at $%04X", addr)
	}
	return nil
}
