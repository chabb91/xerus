package memory

import (
	"SNES_emulator/cartridge"
	"log"
)

type Bus interface {
	ReadByte(address uint32) byte
	WriteByte(address uint32, value byte)
}

type RealBus struct {
	openBusA, openBusB byte

	WRAM      []byte
	cartridge *cartridge.Cartridge
}

func NewBus(cartridge *cartridge.Cartridge) *RealBus {
	return &RealBus{
		WRAM:      make([]byte, 0x20000), // 128 KB
		cartridge: cartridge,
	}
}

func (b *RealBus) ReadByte(address uint32) byte {
	bank := (address >> 16) & 0xFF
	addr := address & 0xFFFF

	if bank == 0x7E || ((bank <= 0x3F || (bank >= 0x80 && bank <= 0xBF)) && addr <= 0x1FFF) {
		return b.WRAM[addr]
	}
	if bank == 0x7F {
		return b.WRAM[0x10000+addr]
	}

	value, err := b.cartridge.ReadByte(byte(bank), uint16(addr))
	if err == nil {
		return value
	}

	log.Printf("Warning: Read from unmapped address $%06X", address)
	return b.openBusA
}

func (b *RealBus) WriteByte(address uint32, value byte) {
	bank := (address >> 16) & 0xFF
	addr := address & 0xFFFF

	if bank == 0x7E || ((bank <= 0x3F || (bank >= 0x80 && bank <= 0xBF)) && addr <= 0x1FFF) {
		b.WRAM[addr] = value
		return
	}
	if bank == 0x7F {
		b.WRAM[0x10000+addr] = value
		return
	}

	err := b.cartridge.WriteByte(byte(bank), uint16(addr), value)
	if err == nil {
		log.Printf("Warning: Write to unmapped or invalid address $%06X", address)
	}

}
