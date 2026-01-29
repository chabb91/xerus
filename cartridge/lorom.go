package cartridge

type loRom struct {
	cartridgeType  int
	headerLocation uint32
}

func NewLoRom() *loRom {
	return &loRom{
		cartridgeType:  LoROM,
		headerLocation: 0x7FC0}
}

func (lr loRom) getHeaderLocation() uint32 {
	return lr.headerLocation
}

func (lr loRom) getCartridgeType() int {
	return lr.cartridgeType
}

func (_ loRom) mapToCartridge(bank byte, offset uint16, hasSram bool) (int, int) {
	maskedBank := bank & 0x7F
	if bank == 0x7E || bank == 0x7F || (maskedBank < 0x40 && offset < 0x8000) {
		return -1, unmappedAddress
	}
	if hasSram {
		if maskedBank <= 0x7D && maskedBank >= 0x70 {
			return int(maskedBank-0x70)<<15 | int(offset), sramAddress //<<15 == *0x8000
		}
	}
	offset = (offset & 0x7FFF) | (uint16(bank&1) << 15)
	bank = (bank & 0x7F) >> 1
	return int(bank)<<16 | int(offset), romAddress
}
