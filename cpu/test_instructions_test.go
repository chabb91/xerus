package cpu

import (
	"SNES_emulator/debugger"
	"SNES_emulator/memory"
	"strings"
	"testing"
)

var cause string

func Test4C(t *testing.T) {
	tests, err := debugger.LoadTests("../testdata/40.n.json")
	if err != nil {
		t.Fatal(err)
	}

	ram := memory.NewTestBus()
	cpu := NewCPU(ram)

	for _, tc := range tests {
		cpu.Reset()
		setState(cpu, tc.Initial)
		i := 0
		for {
			ret := cpu.stepCycle()
			if i < len(tc.Cycles) {
				if !compareCycle(cpu, tc.Cycles[i]) {
					//t.Errorf("INACCURATE CYCLE: %v, %s[%v]", tc.Name, "cycle", i)
				}
			}
			i++
			if ret {
				if len(tc.Cycles) != i {
					t.Errorf("CYCLE COUNT MISMATCH: %v, %v(expected), %v(emulated)", tc.Name, len(tc.Cycles), i)
				}
				break
			}
		}

		if !compareState(cpu, tc.Final) {
			t.Errorf("FAIL: %v, %s", tc.Name, cause)
			t.Errorf("Expected: %v", tc.Final.RAM)
			for _, v := range tc.Final.RAM {
				if cpu.bus.ReadByte(v.Address) != v.Data {
					t.Error(v.Address, " ", cpu.bus.ReadByte(v.Address), " ", v.Data)
				}
			}

		}
	}
}

func compareCycle(c *CPU, ref []any) bool {
	if len(ref) < 3 {
		return false
	}

	addr, ok := ref[0].(float64)
	if !ok {
		return false
	}

	var val byte
	if ref[1] != nil {
		floatVal, ok := ref[1].(float64)
		if !ok {
			return false
		}
		val = byte(floatVal)

		if c.bus.ReadByte(uint32(addr)) != val {
			return false
		}
	}

	out, ok := ref[2].(string)
	if !ok {
		return false
	}

	ok1 := strings.ContainsRune(out, 'm') == c.r.hasFlag(FlagM)
	ok2 := strings.ContainsRune(out, 'x') == c.r.hasFlag(FlagX)
	ok3 := strings.ContainsRune(out, 'e') == c.r.E

	return ok1 && ok2 && ok3
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
	if c.r.GetX() != s.X {
		cause = "X"
		return false
	}
	if c.r.GetY() != s.Y {
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
