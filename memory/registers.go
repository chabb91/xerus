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

	//bus.registers.RegisterRange(0x2100, 0x213F, ppuHandler, "PPU")
	//bus.registers.RegisterRange(0x2140, 0x2143, apuHandler, "APU")
	//bus.registers.RegisterRange(0x4000, 0x41FF, joypadHandler, "Controllers")
	//bus.registers.RegisterRange(0x4200, 0x44FF, cpuHandler, "CPU Registers")
}
