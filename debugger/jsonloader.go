package debugger

import (
	"encoding/json"
	"fmt"
	"os"
)

type CPUState struct {
	PC  uint16 `json:"pc"`
	S   uint16 `json:"s"`
	P   uint8  `json:"p"`
	A   uint16 `json:"a"`
	X   uint16 `json:"x"`
	Y   uint16 `json:"y"`
	DBR uint8  `json:"dbr"`
	D   uint16 `json:"d"`
	PBR uint8  `json:"pbr"`
	E   uint8  `json:"e"`
	RAM Memory `json:"ram"`
}

type MemoryBlock struct {
	Address uint32
	Data    byte
}

type InstructionTest struct {
	Name    string   `json:"name"`
	Initial CPUState `json:"initial"`
	Final   CPUState `json:"final"`
	//TODO replace empty interface with an actual struct to be able to effectively test cycle accuracy
	Cycles [][]any `json:"cycles"`
}

type Memory []MemoryBlock

func (m *Memory) UnmarshalJSON(data []byte) error {
	var raw [][]uint32
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	blocks := make([]MemoryBlock, len(raw))
	for i, pair := range raw {
		if len(pair) != 2 {
			return fmt.Errorf("invalid memory pair length %d", len(pair))
		}
		blocks[i].Address = pair[0]
		blocks[i].Data = byte(pair[1])
	}
	*m = blocks
	return nil
}

func LoadTests(path string) ([]InstructionTest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tests []InstructionTest
	if err := json.Unmarshal(data, &tests); err != nil {
		return nil, err
	}
	return tests, nil
}

func (c CPUState) IsEmulationMode() bool {
	switch c.E {
	case 1:
		return true
	case 0:
		return false
	}
	return false
}
