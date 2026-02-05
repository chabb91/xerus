package spc700

type TestMemory struct {
	ram    [0x10000]byte
	cycles []CycleAccess
}

func newTestMemory() *TestMemory {
	return &TestMemory{cycles: make([]CycleAccess, 16)}
}

type CycleAccess struct {
	Cycle uint
	Addr  uint16
	Value byte
	Type  string // "read", "write", "wait"
}

func (t *TestMemory) Read8(addr uint16) byte {
	val := t.ram[addr]
	t.cycles = append(t.cycles, CycleAccess{
		Cycle: uint(len(t.cycles)),
		Addr:  addr,
		Value: val,
		Type:  "read",
	})
	return val
}

func (t *TestMemory) Write8(addr uint16, val byte) {
	t.ram[addr] = val
	t.cycles = append(t.cycles, CycleAccess{
		Cycle: uint(len(t.cycles)),
		Addr:  addr,
		Value: val,
		Type:  "write",
	})
}

// Test-specific methods (not on interface)
func (t *TestMemory) RecordWait() {
	t.cycles = append(t.cycles, CycleAccess{
		Cycle: uint(len(t.cycles)),
		Type:  "wait",
	})
}

func (t *TestMemory) GetCycles() []CycleAccess {
	return t.cycles
}

func (t *TestMemory) ClearCycles() {
	t.cycles = t.cycles[:0]
}
