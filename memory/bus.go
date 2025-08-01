package memory

import "log"

// Bus represents the system bus, connecting the CPU to memory and peripherals.
// This is a minimal LoROM implementation with 128KB WRAM.
type Bus struct {
	WRAM []byte // 128 KB of Work RAM
	ROM  []byte // Cartridge ROM data
}

// NewBus creates and initializes a new Bus instance.
// It requires the cartridge's ROM data to be provided.
func NewBus(romData []byte) *Bus {
	return &Bus{
		WRAM: make([]byte, 0x20000), // 128 KB
		ROM:  romData,
	}
}

// ReadByte reads a single byte from the 24-bit address space.
// It handles a minimal address map for WRAM and ROM.
func (b *Bus) ReadByte(address uint32) byte {
	// A simple address decoding for a minimal setup.
	// This only handles WRAM mirrors and the LoROM bank.
	bank := (address >> 16) & 0xFF
	addr := address & 0xFFFF

	// ----------------------
	// WRAM ($00-3F:0000-1FFF mirrors, $7E-7F:0000-FFFF)
	// ----------------------
	if bank >= 0x7E && bank <= 0x7F {
		// 128KB WRAM is mirrored here
		return b.WRAM[addr]
	}
	if bank >= 0x00 && bank <= 0x3F && addr >= 0x0000 && addr <= 0x1FFF {
		// WRAM mirror
		return b.WRAM[addr]
	}

	// ----------------------
	// LoROM ROM ($80-FF:8000-FFFF)
	// ----------------------
	if bank >= 0x80 && addr >= 0x8000 {
		// Calculate the offset into the ROM data.
		// Banks 80-FF map to the ROM.
		romOffset := (uint32(bank-0x80) * 0x8000) + (addr - 0x8000)

		// Check if the offset is within the bounds of the ROM.
		if int(romOffset) < len(b.ROM) {
			return b.ROM[romOffset]
		}
	}

	// Default to returning 0 for unmapped or invalid addresses.
	log.Printf("Warning: Read from unmapped address $%06X", address)
	return 0
}

// WriteByte writes a single byte to the 24-bit address space.
// This minimal implementation only allows writes to WRAM.
func (b *Bus) WriteByte(address uint32, value byte) {
	bank := (address >> 16) & 0xFF
	addr := address & 0xFFFF

	// ----------------------
	// WRAM ($00-3F:0000-1FFF mirrors, $7E-7F:0000-FFFF)
	// ----------------------
	if bank >= 0x7E && bank <= 0x7F {
		if int(addr) < len(b.WRAM) {
			b.WRAM[addr] = value
		}
		return
	}
	if bank >= 0x00 && bank <= 0x3F && addr >= 0x0000 && addr <= 0x1FFF {
		if int(addr) < len(b.WRAM) {
			b.WRAM[addr] = value
		}
		return
	}

	// For a bare-minimum implementation, we'll ignore writes to unmapped addresses or ROM.
	log.Printf("Warning: Write to unmapped or invalid address $%06X", address)
}
