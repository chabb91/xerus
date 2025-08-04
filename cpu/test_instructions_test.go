package cpu

import (
	"SNES_emulator/debugger"
	"SNES_emulator/memory"
	"strings"
	"testing"
)

var cause string

func Test4C(t *testing.T) {
	tests, err := debugger.LoadTests("../testdata/4c.e.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		ram := memory.NewTestBus()
		cpu := NewCPU(ram)
		cpu.Reset()
		setState(cpu, tc.Initial)
		i := 0
		for {
			if i < len(tc.Cycles) {
				if !compareCycle(cpu, tc.Cycles[i]) {
					t.Errorf("INACCURATE CYCLE: %v, %s[%v]", tc.Name, "cycle", i)
				}
			}
			ret := cpu.stepCycle()
			i++
			if ret {
				if len(tc.Cycles) != i {
					t.Errorf("CYCLE COUNT MISMATCH: %v, %s[%v]", tc.Name, "cycle", i)
				}
				break
			}
		}

		if !compareState(cpu, tc.Final) {
			t.Errorf("FAIL: %v, %s", tc.Name, cause)
		}
	}
}

func compareCycle(c *CPU, ref []any) bool {
	addr := uint32(ref[0].(float64))
	val := byte(ref[1].(float64))
	out := ref[2].(string)
	if c.bus.ReadByte(addr) == val {
		ok1 := strings.ContainsRune(out, 'm') == c.r.hasFlag(FlagM)
		ok2 := strings.ContainsRune(out, 'x') == c.r.hasFlag(FlagX)
		ok3 := strings.ContainsRune(out, 'e') == c.r.E
		return ok1 && ok2 && ok3

	}
	return false
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
