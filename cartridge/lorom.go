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

func (lr loRom) mapToCartridge(bank byte, offset uint16, hasSram bool) (int, int) {
	if bank == 0x7E || bank == 0x7F {
		return -1, unmappedAddress
	}
	if offset >= 0x8000 {
		if bank <= 0x7D {
			return int(bank)<<15 + int(offset-0x8000), romAddress //<<15 == *0x8000
		}
		if bank >= 0x80 {
			return int(bank-0x80)<<15 + int(offset-0x8000), romAddress
		}
	} else {
		if hasSram {
			if bank <= 0x7D && bank >= 0x70 {
				return int(bank-0x70)<<15 + int(offset), sramAddress
			}
		} else {
			if bank <= 0x7D && bank >= 0x40 {
				return int(bank)<<15 + int(offset), romAddress
			}
			if bank >= 0xC0 {
				return int(bank-0x80)<<15 + int(offset), romAddress
			}
		}
	}
	return -1, unmappedAddress
}
