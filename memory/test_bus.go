package memory

// TestBus implements the Bus interface for automated testing.
type TestBus struct {
	ram map[uint32]byte
}

// NewTestBus creates a new Bus for testing.
func NewTestBus() *TestBus {
	return &TestBus{
		ram: make(map[uint32]byte),
	}
}

// mapWRAMAddress resolves a WRAM address, handling mirrors.
func (b *TestBus) mapWRAMAddress(address uint32) uint32 {
	// Return original address if not WRAM
	return address
}

func (b *TestBus) ReadByte(address uint32) byte {
	// First, map the address to its canonical location
	canonicalAddress := b.mapWRAMAddress(address)

	if val, ok := b.ram[canonicalAddress]; ok {
		return val
	}
	// For testing, we can assume uninitialized memory is 0
	return 0
}

func (b *TestBus) WriteByte(address uint32, value byte) {
	// Map the address to its canonical location before writing
	canonicalAddress := b.mapWRAMAddress(address)
	b.ram[canonicalAddress] = value
}
