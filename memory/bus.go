package memory

import (
	"SNES_emulator/cartridge"
	"log"
)

const WRAM_SIZE = 0x20000 // 128 KB

const FAST_REGION = byte(3)
const SLOW_REGION = byte(4)
const XSLOW_REGION = byte(6)

type Bus interface {
	ReadByte(address uint32) byte
	WriteByte(address uint32, value byte)
	RegisterRange(start, end uint16, handler RegisterHandler, name string)
	//sets the speed of the rom access.
	//serves as a temporary overclock for slow roms.
	//does nothing for fast roms
	SetMEMSEL(value byte)
	GetOpenBus() byte
	GetAccessClass(address uint32) byte
}

type RealBus struct {
	openBus byte

	registers *RegisterSystem

	WRAM      [WRAM_SIZE]byte
	cartridge *cartridge.Cartridge
	memsel    byte
}

func NewBus(cartridge *cartridge.Cartridge) *RealBus {
	rb := &RealBus{
		registers: NewRegisterSystem(),
		cartridge: cartridge,
		memsel:    SLOW_REGION,
	}

	rb.RegisterRange(0x2180, 0x2183, newWramDataRW(rb.WRAM[:]), "WRAM")
	return rb
}

func (b *RealBus) ReadByte(address uint32) byte {
	bank, addr := splitAddress24(address)

	index, ok := b.wramIndex(bank, addr)
	if ok {
		b.openBus = b.WRAM[index]
		return b.openBus
	}

	if b.registers.IsRegisterAddress(bank, addr) {
		handler, name, err := b.registers.FindHandler(addr)
		if err != nil {
			log.Printf("Warning: No handler for register $%04X (%s)", addr, name)
			return b.openBus
		}

		value, err := handler.Read(addr)
		if err != nil {
			log.Printf("Warning: Register read error at $%04X (%s): %v", addr, name, err)
			return b.openBus
		}

		b.openBus = value
		return b.openBus
	}

	value, err := b.cartridge.ReadByte(bank, addr)
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

	if b.registers.IsRegisterAddress(bank, addr) {
		handler, name, err := b.registers.FindHandler(addr)
		if err != nil {
			log.Printf("Warning: No handler for register $%04X (%s)", addr, name)
			return
		}

		err = handler.Write(addr, value)
		if err != nil {
			log.Printf("Warning: Register write error at $%04X (%s): %v", addr, name, err)
		}
		return
	}

	err := b.cartridge.WriteByte(bank, addr, value)
	if err != nil {
		log.Printf("Cartridge: Write to unmapped or invalid address $%06X", address)
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

func (b *RealBus) RegisterRange(start, end uint16, handler RegisterHandler, name string) {
	b.registers.RegisterRange(start, end, handler, name)
}

func (b *RealBus) SetMEMSEL(value byte) {

	if value&1 == 1 {
		b.memsel = FAST_REGION
	} else {
		b.memsel = SLOW_REGION
	}
}

func (b *RealBus) GetOpenBus() byte {
	return b.openBus
}

func (b *RealBus) GetAccessClass(address uint32) byte {
	bank := byte(address >> 16)
	offset := byte(address >> 8)

	if bank >= 0x40 && bank <= 0x7F {
		return SLOW_REGION
	}

	if bank >= 0xC0 || ((bank >= 0x80 && bank <= 0xBF) && offset >= 0x80) {
		return b.memsel
	}

	//[00-3F] or [80-BF]
	if offset >= 0x20 && offset <= 0x5F {
		return FAST_REGION
	}

	if offset == 0x40 || offset == 0x41 {
		return XSLOW_REGION
	}

	return SLOW_REGION
}
