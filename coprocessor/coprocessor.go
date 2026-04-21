package coprocessor

import "github.com/chabb91/xerus/internal/types"

type RegisterMap struct {
	Start, End uint16
	Name       string
}

type Coprocessor interface {
	GetRegisterMap() RegisterMap

	//force memory.RegisterHandler interface for all chips
	Read(addr uint16) (byte, error)
	Write(addr uint16, value byte) error

	//every coprocessor carries its own mapper
	//which then it can use to get data using the cartridge data source
	Read8(bank byte, offset uint16) (byte, error)
	Write8(bank byte, offset uint16, value byte) error

	//override the mapper detected by the cartridge if needed
	//return nil otherwise
	OverrideCartridgeMapper() types.RomMapper

	SetCartridge(CartridgeDataSource)
	SetInterruptManager(InterruptManager)

	Step() uint64
}

// passing the cartridge data as interface
type CartridgeDataSource interface {
	ReadRam(index int) byte
	ReadRom(index int) byte
	WriteRam(index int, value byte)
}

// wiring up cartridge irq signal
type InterruptManager interface {
	CartFireIrq()
	CartAcknowledgeIrq()
}
