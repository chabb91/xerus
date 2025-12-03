package debugger

import (
	"encoding/json"
	"fmt"
	"os"
)

type ProcessorState interface {
	CPUState | APUState
}

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

type APUState struct {
	PC  uint16 `json:"pc"`
	A   uint8  `json:"a"`
	X   uint8  `json:"x"`
	Y   uint8  `json:"y"`
	SP  uint8  `json:"sp"`
	PSW uint8  `json:"psw"`
	RAM Memory `json:"ram"`
}

type MemoryBlock struct {
	Address uint32
	Data    byte
}

type InstructionTest[T ProcessorState] struct {
	Name    string  `json:"name"`
	Initial T       `json:"initial"`
	Final   T       `json:"final"`
	Cycles  [][]any `json:"cycles"`
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

func LoadTests[T ProcessorState](path string) ([]InstructionTest[T], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tests []InstructionTest[T]
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
