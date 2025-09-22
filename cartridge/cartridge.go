package cartridge

const (
	LoROM   = 0
	HiROM   = 1
	ExHiROM = 5
)

const (
	romAddress = iota
	sramAddress
	unmappedAddress
)

type romMapper interface {
	//	ReadByte(address uint32) (value byte, ok bool)
	//	WriteByte(address uint32) (ok bool)

	getHeaderLocation() uint32
	getCartridgeType() int

	mapToCartridge(bank byte, offset uint16, hasSram bool) (index, addressType int)
}

type Cartridge struct {
	Mapper romMapper

	hasSram bool

	romData  []byte
	sramData []byte
}

func (cart Cartridge) Load() {

}
