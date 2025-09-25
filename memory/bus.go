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
	openBus byte

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
	bank, addr := splitAddress24(address)

	index, ok := b.wramIndex(bank, addr)
	if ok {
		b.openBus = b.WRAM[index]
		return b.openBus
	}

	value, err := b.cartridge.ReadByte(byte(bank), uint16(addr))
	if err == nil {
		b.openBus = value
		return b.openBus
	}

	log.Printf("Warning: Read from unmapped address $%06X", address)
	return b.openBus
}

func (b *RealBus) WriteByte(address uint32, value byte) {
	bank, addr := splitAddress24(address)

	index, ok := b.wramIndex(bank, addr)
	if ok {
		b.WRAM[index] = value
		return
	}

	err := b.cartridge.WriteByte(byte(bank), uint16(addr), value)
	if err == nil {
		log.Printf("Warning: Write to unmapped or invalid address $%06X", address)
	}

}

func splitAddress24(address uint32) (byte, uint16) {
	return byte((address >> 16) & 0xFF), uint16(address & 0xFFFF)
}

func (b *RealBus) wramIndex(bank byte, offset uint16) (int, bool) {
	if bank == 0x7E || ((bank <= 0x3F || (bank >= 0x80 && bank <= 0xBF)) && offset <= 0x1FFF) {
		return int(offset), true
	}
	if bank == 0x7F {
		return 0x10000 + int(offset), true
	}
	return 0, false
}
