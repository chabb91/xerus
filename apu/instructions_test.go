package apu

import (
	"SNES_emulator/debugger"
	"strings"
	"testing"
)

var cause string

func TestSingleInstruction(t *testing.T) {
	tests, err := debugger.LoadTests[debugger.APUState]("/home/chabb/Documents/snes_tests/spc700/f3.json")
	if err != nil {
		t.Fatal(err)
	}
	testMem := newTestMemory()
	cpu := NewCPU(testMem)

	cpu.resetSignal = false

	for _, tc := range tests {
		setState(cpu, tc.Initial)
		testMem.ClearCycles()
		i := 0
		for {
			ret := cpu.StepCycle()
			i++
			if len(testMem.cycles) != i {
				testMem.RecordWait()
			}
			if ret {
				if len(tc.Cycles) != i {
					t.Errorf("CYCLE COUNT MISMATCH: %v, %v(expected), %v(emulated)", tc.Name, len(tc.Cycles), i)
				}
				break
			}
		}
		if !compareCycles(testMem.cycles, tc.Cycles) {
			t.Errorf("INACCURATE CYCLE: Expected: %v, Got: %v", tc.Cycles, testMem.cycles)
		}
		if !compareState(cpu, tc.Final) {
			t.Errorf("FAIL: %v, %s", tc.Name, cause)
			if strings.Contains(cause, "Memory Address") {
				t.Errorf("(Memory Address) Expected: %v", tc.Final.RAM)
				for _, v := range tc.Final.RAM {
					if addr := testMem.ram[v.Address]; addr != v.Data {
						t.Error(v.Address, " ", addr, " ", v.Data)
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

func compareCycles(got []CycleAccess, expected [][]any) bool {
	if len(got) != len(expected) {
		return false
	}

	for i := range len(got) {
		expType, ok := expected[i][2].(string)
		if !ok {
			return false
		}

		if got[i].Type != expType {
			return false
		}

		if expType != "wait" {
			expAddr, ok := expected[i][0].(float64)
			if !ok {
				return false
			}

			expValue, ok := expected[i][1].(float64)
			if !ok {
				return false
			}

			if got[i].Addr != uint16(expAddr) || got[i].Value != byte(expValue) {
				return false
			}
		}
	}

	return true
}

func setState(c *CPU, s debugger.APUState) {
	c.r.PC = s.PC
	c.r.A = s.A
	c.r.X = s.X
	c.r.Y = s.Y
	c.r.SP = s.SP
	c.r.PSW = s.PSW

	testMem := c.psram.(*TestMemory)
	for _, v := range s.RAM {
		testMem.ram[v.Address] = v.Data
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

	testMem := c.psram.(*TestMemory)
	for _, v := range s.RAM {
		if testMem.ram[v.Address] != v.Data {
			cause += " Memory Address"
			return false
		}
	}
	return true
}
