package apu

import (
	"SNES_emulator/debugger"
	"strings"
	"testing"
)

var cause string

func TestSingleInstruction(t *testing.T) {
	tests, err := debugger.LoadTests[debugger.APUState]("/home/chabb/Documents/snes_tests/spc700/5f.json")
	if err != nil {
		t.Fatal(err)
	}

	cpu := NewCPU()

	cpu.resetSignal = false

	for _, tc := range tests {
		setState(cpu, tc.Initial)
		i := 0
		for {
			ret := cpu.StepCycle()
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
			if strings.Contains(cause, "Memory Address") {
				t.Errorf("(Memory Address) Expected: %v", tc.Final.RAM)
				for _, v := range tc.Final.RAM {
					if cpu.psram[v.Address] != v.Data {
						t.Error(v.Address, " ", cpu.psram[v.Address], " ", v.Data)
					}
				}
			}
			if strings.Contains(cause, "PC") {
				t.Errorf("(PC) Expected: %v, Got: %v", tc.Final.PC, cpu.r.PC)
			}
		}
		cause = ""
	}
}

func setState(c *CPU, s debugger.APUState) {
	c.r.PC = s.PC
	c.r.A = s.A
	c.r.X = s.X
	c.r.Y = s.Y
	c.r.SP = s.SP
	c.r.PSW = s.PSW

	for _, v := range s.RAM {
		c.psram[v.Address] = v.Data
	}
}

func compareState(c *CPU, s debugger.APUState) bool {
	if c.r.PC != s.PC {
		cause += " PC"
		return false
	}
	if c.r.A != s.A {
		cause += " A"
		return false
	}
	if c.r.X != s.X {
		cause += " X"
		return false
	}
	if c.r.Y != s.Y {
		cause += " Y"
		return false
	}
	if c.r.SP != s.SP {
		cause += " SP"
		return false
	}
	if c.r.PSW != s.PSW {
		cause += " PSW"
		return false
	}
	for _, v := range s.RAM {
		if c.psram[v.Address] != v.Data {
			cause += " Memory Address"
			return false
		}
	}
	return true
}
