package spc700

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chabb91/xerus/debugger"
)

var cause string

func TestAllInetructions(t *testing.T) {
	testDir := "/home/chabb/Documents/SNES-cpu-tests/spc700"

	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		testFile := filepath.Join(testDir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			runInstructionTests(t, testFile)
		})
	}
}

func TestSingleInstruction(t *testing.T) {
	runInstructionTests(t, "/home/chabb/Documents/SNES-cpu-tests/spc700/4f.json")
}

func runInstructionTests(t *testing.T, testFile string) {
	tests, err := debugger.LoadTests[debugger.APUState](testFile)
	if err != nil {
		t.Fatal(err)
	}
	testMem := newTestMemory()
	cpu := NewCPU(testMem)

	cpu.Reset()

	for _, tc := range tests {
		cpu.stopped = false
		setState(cpu, tc.Initial)
		testMem.ClearCycles()
		i := 0
		timeout := 0
		for {
			ioCntPre := len(testMem.cycles)
			ret := cpu.StepCycle()
			ioCntAfter := len(testMem.cycles)
			if ioCntAfter-ioCntPre > 1 {
				t.Fatalf("Fatal: multiple io operations in one cycle on test %d: %s", i, tc.Name)
			}
			timeout++
			if timeout > 1000 {
				t.Fatalf("TIMEOUT on test %d: %s", i, tc.Name)
			}
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
			if strings.Contains(cause, "PSW") {
				t.Errorf("(PSW) Expected: %v, Got: %v", tc.Final.PSW, cpu.r.PSW)
			}
			if strings.Contains(cause, "A") {
				t.Errorf("(A) Expected: %v, Got: %v", tc.Final.A, cpu.r.A)
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

		expValue := expected[i][1]
		isDummyRead := expType == "read" && expValue == nil

		if isDummyRead || expType == "wait" {
			if got[i].Type != "wait" {
				return false
			}
			continue
		}

		if got[i].Type != expType {
			return false
		}

		expAddr, ok := expected[i][0].(float64)
		if !ok {
			return false
		}

		expVal, ok := expValue.(float64)
		if !ok {
			return false
		}

		if got[i].Addr != uint16(expAddr) || got[i].Value != byte(expVal) {
			return false
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
