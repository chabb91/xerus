package coprocessor

type RegisterMap struct {
	Start, End uint16
	Name       string
}

type Coprocessor interface {
	GetRegisterMap() RegisterMap

	//force memory.RegisterHandler interface for all chips
	Read(addr uint16) (byte, error)
	Write(addr uint16, value byte) error

	SetCartridge(CartridgeDataSource)
}

// passing the cartridge data as interface
type CartridgeDataSource interface {
	ReadRam(index int) byte
	ReadRom(index int) byte
	WriteRam(index int, value byte)
}
