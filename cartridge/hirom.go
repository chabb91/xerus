package cartridge

type hiRom struct {
	cartridgeType  int
	headerLocation uint32
}

func NewHiRom() *hiRom {
	return &hiRom{
		cartridgeType:  HiROM,
		headerLocation: 0xFFC0}
}

func (lr hiRom) getHeaderLocation() uint32 {
	return lr.headerLocation
}

func (lr hiRom) getCartridgeType() int {
	return lr.cartridgeType
}

func (lr hiRom) mapToCartridge(bank byte, offset uint16, hasSram bool) (int, int) {
	if bank == 0x7E || bank == 0x7F {
		return -1, unmappedAddress
	}
	if bank >= 0xC0 {
		return int(bank-0xC0)<<16 + int(offset), romAddress
	}
	if bank >= 0x80 {
		//trusting fullsnes for the Hirom sram mappings. there is another sram mapping not implemented here
		//10-1f,30-3f,90-9f,b0-bf
		if hasSram && bank >= 0xA0 && offset >= 0x6000 && offset < 0x8000 {
			return int(bank-0xA0)<<13 + int(offset-0x6000), sramAddress
		}
		if offset >= 0x8000 {
			return int(bank-0x80)<<16 + int(offset), romAddress
		} else {
			return -1, unmappedAddress
		}
	}
	if bank >= 0x40 {
		return int(bank-0x40)<<16 + int(offset), romAddress
	}
	//bank >=0
	if hasSram && bank >= 0x20 && offset >= 0x6000 && offset < 0x8000 {
		return int(bank-0x20)<<13 + int(offset-0x6000), sramAddress
	}
	if offset >= 0x8000 {
		return int(bank)<<16 + int(offset), romAddress
	}
	return -1, unmappedAddress
}
