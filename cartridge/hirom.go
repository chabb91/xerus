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

func (hr hiRom) getHeaderLocation() uint32 {
	return hr.headerLocation
}

func (hr hiRom) getCartridgeType() int {
	return hr.cartridgeType
}

func (_ hiRom) mapToCartridge(bank byte, offset uint16, hasSram bool) (int, int) {
	if bank == 0x7E || bank == 0x7F {
		return -1, unmappedAddress
	}
	if sramBank := bank & 0x7F; sramBank < 0x40 {
		if offset < 0x8000 {
			//trusting fullsnes for the Hirom sram mappings.
			//there is another sram mapping not implemented here
			//10-1f,30-3f,90-9f,b0-bf
			if hasSram && offset >= 0x6000 {
				return int(sramBank-0x20)<<13 | int(offset-0x6000), sramAddress
			}
			return -1, unmappedAddress
		}
	}
	return int(bank&0x3F)<<16 | int(offset), romAddress
}

type exHiRom struct {
	cartridgeType  int
	headerLocation uint32
}

func NewExHiRom() *exHiRom {
	return &exHiRom{
		cartridgeType:  ExHiROM,
		headerLocation: 0x40FFC0}
}

func (hr exHiRom) getHeaderLocation() uint32 {
	return hr.headerLocation
}

func (hr exHiRom) getCartridgeType() int {
	return hr.cartridgeType
}

func (_ exHiRom) mapToCartridge(bank byte, offset uint16, hasSram bool) (int, int) {
	if bank == 0x7E || bank == 0x7F {
		return -1, unmappedAddress
	}
	if bank&0x7F < 0x40 {
		if offset < 0x8000 {
			if hasSram && offset >= 0x6000 {
				return int(bank)<<13 | int(offset-0x6000), sramAddress
			}
			return -1, unmappedAddress
		}
	}
	mask := ((bank ^ 0x80) | 0x7F) >> 1
	return int(bank&mask)<<16 | int(offset), romAddress
}
