package cpu

import (
	"SNES_emulator/debugger"
	"SNES_emulator/memory"
	"testing"
)

var cause string

func Test4C(t *testing.T) {
	tests, err := debugger.LoadTests("../testdata/6c.n.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		ram := memory.NewTestBus()
		cpu := NewCPU(ram)
		cpu.Reset()
		setState(cpu, tc.Initial)
		for {
			ret := cpu.stepCycle()
			if ret {
				break
			}
		}

		if !compareState(cpu, tc.Final) {
			t.Errorf("FAIL: %v, %s", tc.Name, cause)
		}
	}
}

func setState(c *CPU, s debugger.CPUState) {
	if s.E == 0 {
		c.r.E = false
	} else {
		c.r.EmulationON()
	}

	c.r.PC = s.PC
	c.r.SetStack(s.S)
	c.r.P = s.P
	c.r.A = s.A
	c.r.X = s.X
	c.r.Y = s.Y
	c.r.DB = s.DBR
	c.r.D = s.D
	c.r.PB = s.PBR

	for _, v := range s.RAM {
		c.bus.WriteByte(v.Address, v.Data)
	}
}

func compareState(c *CPU, s debugger.CPUState) bool {
	if c.r.A != s.A {
		cause = "A"
		return false
	}
	if c.r.PC != s.PC {
		cause = "PC"
		return false
	}
	if c.r.GetStack() != s.S {
		cause = "S"
		return false
	}
	if c.r.P != s.P {
		cause = "P"
		return false
	}
	if c.r.X != s.X {
		cause = "X"
		return false
	}
	if c.r.Y != s.Y {
		cause = "Y"
		return false
	}
	if c.r.D != s.D {
		cause = "D"
		return false
	}
	if c.r.DB != s.DBR {
		cause = "DB"
		return false
	}
	if c.r.PB != s.PBR {
		cause = "PB"
		return false
	}
	if s.IsEmulationMode() != c.r.E {
		cause = "E"
		return false
	}
	for _, v := range s.RAM {
		if c.bus.ReadByte(v.Address) != v.Data {
			cause = "Memory Address"
			return false
		}
	}
	return true
}
