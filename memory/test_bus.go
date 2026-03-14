package memory

// TestBus implements the Bus interface for automated testing.
type TestBus struct {
	ram map[uint32]byte
}

func NewTestBus() *TestBus {
	return &TestBus{
		ram: make(map[uint32]byte),
	}
}

func (b *TestBus) ReadByte(address uint32) byte {
	if val, ok := b.ram[address]; ok {
		return val
	}
	// For testing, we can assume uninitialized memory is 0
	return 0
}

func (b *TestBus) WriteByte(address uint32, value byte) {
	b.ram[address] = value
}

func (b *TestBus) RegisterRange(start, end uint16, handler RegisterHandler, name string) {
}

func (b *TestBus) SetMEMSEL(value byte) {
}

func (b *TestBus) GetOpenBus() byte {
	return 0
}

func (b *TestBus) GetAccessClass(address uint32) uint64 {
	return FAST_REGION
}
