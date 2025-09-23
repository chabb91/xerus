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

func (cart Cartridge) ReadByte(bank byte, offset uint16) (value byte, ok bool) {
	index, addressType := cart.Mapper.mapToCartridge(bank, offset, cart.hasSram)

	switch addressType {
	case romAddress:
		return cart.romData[index%len(cart.romData)], true
	case sramAddress:
		//TODO this can be nil
		return cart.sramData[index%len(cart.romData)], true
	default:
		//unmappedAddress
		return 0, false
	}
}

func (cart Cartridge) HasSram() bool {
	val, ok := cart.ReadByte(0, 0xFFD6)
	if !ok {
		return ok
	} else {
		return (val&0x7 == 0x1) || (val&0x7 == 0x2) || (val&0x7 == 0x4) || (val&0x7 == 0x5)
	}
}
