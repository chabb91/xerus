package cartridge

import (
	"errors"
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

func (cart Cartridge) ReadByte(bank byte, offset uint16) (byte, error) {
	index, addressType := cart.Mapper.mapToCartridge(bank, offset, cart.HasSram())

	switch addressType {
	case romAddress:
		return cart.romData[index%len(cart.romData)], nil
	case sramAddress:
		if cart.HasSram() {
			return cart.sramData[index%len(cart.romData)], nil
		} else {
			return 0, errors.New("Trying to read SRAM but the cartridge doesnt have one")
		}
	default:
		//unmappedAddress
		return 0, errors.New("Trying to read unmapped address space.")
	}
}

func (cart Cartridge) WriteByte(bank byte, offset uint16, value byte) error {
	if !cart.HasSram() {
		return errors.New("No SRAM present so writes arent allowed")
	}

	index, addressType := cart.Mapper.mapToCartridge(bank, offset, true)

	if addressType == sramAddress {
		cart.sramData[index%len(cart.romData)] = value
		return nil
	}

	return errors.New("Trying to write to read only or unmapped region")
}

func (cart Cartridge) HasSram() bool {
	return cart.sramData != nil
}

// TODO this always creates a new sram but if theres a battery and there is a SRAM file already it should be loaded instead
func (cart Cartridge) DetectSram() []byte {
	romType, err := cart.ReadByte(0, 0xFFD6)
	if err != nil {
		return nil
	}

	switch romType & 0x7 {
	case 0x1, 0x2, 0x4, 0x5:
		sizeVal, err := cart.ReadByte(0, 0xFFD8)
		if sizeVal == 0 || err != nil {
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
