package cartridge

import (
	"os"
)

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
	getHeaderLocation() uint32
	getCartridgeType() int

	mapToCartridge(bank byte, offset uint16, hasSram bool) (index, addressType int)
}

type Cartridge struct {
	Mapper romMapper

	romData  []byte
	sramData []byte
}

func NewCartridge(romData []byte, mapper romMapper) *Cartridge {
	cart := &Cartridge{
		romData: romData,
		Mapper:  mapper}

	cart.DetectSram()

	return cart
}

func (cart Cartridge) ReadByte(bank byte, offset uint16) (value byte, ok bool) {
	index, addressType := cart.Mapper.mapToCartridge(bank, offset, cart.HasSram())

	switch addressType {
	case romAddress:
		return cart.romData[index%len(cart.romData)], true
	case sramAddress:
		if cart.HasSram() {
			return cart.sramData[index%len(cart.romData)], true
		} else {
			return 0, false
		}
	default:
		//unmappedAddress
		return 0, false
	}
}

func (cart Cartridge) WriteByte(bank byte, offset uint16, value byte) (ok bool) {
	if !cart.HasSram() {
		return false
	}

	index, addressType := cart.Mapper.mapToCartridge(bank, offset, true)

	if addressType == sramAddress {
		cart.sramData[index%len(cart.romData)] = value
		return true
	}

	return false
}

func (cart Cartridge) HasSram() bool {
	return cart.sramData != nil
}

// TODO this always creates a new sram but if theres a battery and there is a SRAM file already it should be loaded instead
func (cart Cartridge) DetectSram() []byte {
	romType, ok := cart.ReadByte(0, 0xFFD6)
	if !ok {
		return nil
	}

	switch romType & 0x7 {
	case 0x1, 0x2, 0x4, 0x5:
		sizeVal, ok := cart.ReadByte(0, 0xFFD8)
		if sizeVal == 0 || !ok {
			return nil
		}
		//return make([]byte, 1<<(10+sizeVal))
		return make([]byte, (1<<sizeVal)*1024)
	default:
		return nil
	}
}

func Load(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}
