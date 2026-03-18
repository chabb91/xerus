package cartridge

func mapLoRom(bank byte, offset uint16, hasSram bool) (int, romRegionType) {
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
	return int(maskedBank>>1)<<16 | int(offset), romAddress
}

func mapHiRom(bank byte, offset uint16, hasSram bool) (int, romRegionType) {
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

func mapExHiRom(bank byte, offset uint16, hasSram bool) (int, romRegionType) {
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

func mapGsu(bank byte, offset uint16, _ bool) (int, romRegionType) {
	if bank == 0x7E || bank == 0x7F {
		return -1, unmappedAddress
	}
	maskedBank := bank & 0x7F
	if maskedBank < 0x40 {
		if offset < 0x8000 {
			if offset >= 0x6000 {
				return int(offset - 0x6000), sramAddress
			}
			return -1, unmappedAddress
		}
		offset = (offset & 0x7FFF) | (uint16(bank&1) << 15)
		return int(maskedBank>>1)<<16 | int(offset), romAddress //lorom
	}
	if bank-0x40 < 0x20 || bank >= 0xC0 { //0x40-0x5F || 0xC0-0xFF
		return int(bank&0x3F)<<16 | int(offset), romAddress //hirom
	}
	if sramBank := bank - 0x70; sramBank < 2 {
		return int(sramBank)<<16 | int(offset), sramAddress
	}
	if sramBank := bank - 0x78; sramBank < 2 { //Additional "Backup" RAM  (128Kbyte max, usually none)
		return int(sramBank)<<16 | int(offset), sramAddress
	}
	return -1, unmappedAddress
}
