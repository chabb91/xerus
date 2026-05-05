package memory

import (
	"log"

	"github.com/chabb91/xerus/cartridge"
	"github.com/chabb91/xerus/internal/constants"
)

const WRAM_SIZE = 0x20000 // 128 KB

const FAST_REGION = constants.CYCLE_6
const SLOW_REGION = constants.CYCLE_8
const XSLOW_REGION = constants.CYCLE_12

type Bus interface {
	ReadByte(address uint32) byte
	WriteByte(address uint32, value byte)
	RegisterRange(start, end uint16, handler RegisterHandler, name string)
	//sets the speed of the rom access.
	//serves as a temporary overclock for slow roms.
	//does nothing for fast roms
	SetMEMSEL(value byte)
	GetOpenBus() byte
	GetAccessClass(address uint32) uint64
}

type SnesBus struct {
	openBus byte

	registers *RegisterSystem

	WRAM      [WRAM_SIZE]byte
	cartridge *cartridge.Cartridge
	memsel    uint64
}

func NewBus(cartridge *cartridge.Cartridge) *SnesBus {
	rb := &SnesBus{
		registers: NewRegisterSystem(),
		cartridge: cartridge,
		memsel:    SLOW_REGION,
	}
	if cartridge.Coprocessor != nil {
		rm := cartridge.Coprocessor.GetRegisterMap()
		rb.registers.RegisterRange(rm.Start, rm.End, cartridge.Coprocessor, rm.Name)
	}

	rb.RegisterRange(0x2180, 0x2183, newWramDataRW(rb.WRAM[:]), "WRAM")
	return rb
}

func (b *SnesBus) ReadByte(address uint32) byte {
	bank, addr := splitAddress24(address)

	index, ok := b.wramIndex(bank, addr)
	if ok {
		b.openBus = b.WRAM[index]
		return b.openBus
	}

	if b.registers.IsRegisterAddress(bank, addr) {
		handler, name, err := b.registers.FindHandler(addr)
		if err != nil {
			if constants.ShowWarnings {
				log.Printf("Warning: No handler for register $%04X (%s)", addr, name)
			}
			return b.openBus
		}

		value, err := handler.Read(addr)
		if err != nil {
			if constants.ShowWarnings {
				log.Printf("Warning: Register read error at $%04X (%s): %v", addr, name, err)
			}
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

	if constants.ShowWarnings {
		log.Printf("Warning: Read from unmapped address $%06X", address)
	}
	return b.openBus
}

func (b *SnesBus) WriteByte(address uint32, value byte) {
	bank, addr := splitAddress24(address)

	index, ok := b.wramIndex(bank, addr)
	if ok {
		b.WRAM[index] = value
		return
	}

	if b.registers.IsRegisterAddress(bank, addr) {
		handler, name, err := b.registers.FindHandler(addr)
		if err != nil {
			if constants.ShowWarnings {
				log.Printf("Warning: No handler for register $%04X (%s)", addr, name)
			}
			return
		}

		err = handler.Write(addr, value)
		if err != nil {
			if constants.ShowWarnings {
				log.Printf("Warning: Register write error at $%04X (%s): %v", addr, name, err)
			}
		}
		return
	}

	err := b.cartridge.WriteByte(bank, addr, value)
	if err != nil {
		if constants.ShowWarnings {
			log.Printf("Cartridge: Write to unmapped or invalid address $%06X", address)
		}
	}
}

func splitAddress24(address uint32) (byte, uint16) {
	return byte(address >> 16), uint16(address)
}

func (b *SnesBus) wramIndex(bank byte, offset uint16) (int, bool) {
	if bank == 0x7E || ((bank&0x7F <= 0x3F) && offset <= 0x1FFF) {
		return int(offset), true
	}
	if bank == 0x7F {
		return 0x10000 | int(offset), true
	}
	return 0, false
}

func (b *SnesBus) RegisterRange(start, end uint16, handler RegisterHandler, name string) {
	b.registers.RegisterRange(start, end, handler, name)
}

func (b *SnesBus) SetMEMSEL(value byte) {

	if value&1 == 1 {
		b.memsel = FAST_REGION
	} else {
		b.memsel = SLOW_REGION
	}
}

func (b *SnesBus) GetOpenBus() byte {
	return b.openBus
}

func (b *SnesBus) GetAccessClass(address uint32) uint64 {
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
