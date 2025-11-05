package soc

import (
	"SNES_emulator/memory"
	"fmt"
)

const (
	JOYPAD_UNLATCHED = iota
	JOYPAD_LATCHED
)

type Joypad interface {
	Latch() uint16
}

type JoypadDataHandler struct {
	joypad        Joypad
	latchedValues uint16
	positionCnt   uint16
}

func (jdh *JoypadDataHandler) ReadNextKey(state int) byte {
	if jdh.positionCnt >= 16 {
		return 0
	}
	var ret byte
	if jdh.joypad == nil {
		ret = 1
	} else {
		ret = byte((jdh.latchedValues >> jdh.positionCnt) & 1)
	}
	//apparently trying to read while latched results in position 0 being read over and over again
	//with no shifting. im not sure if that read is on stale data or on always current tho
	//and im not sure if it even matters.
	if state == JOYPAD_UNLATCHED {
		jdh.positionCnt++
	}
	return ret
}

func (jdh *JoypadDataHandler) Reset(state int) {
	jdh.positionCnt = 0
	if jdh.joypad != nil && state == JOYPAD_LATCHED {
		jdh.latchedValues = jdh.joypad.Latch()
	}
}

type JoypadController struct {
	joypads [2][2]JoypadDataHandler //portnumber/controllernumber

	state int

	bus memory.Bus //for open bus values
}

func NewJoypadController(bus memory.Bus) *JoypadController {
	return &JoypadController{
		bus: bus,
	}
}

// joypad id [0-3]
func (jc *JoypadController) Attach(number int, joypad Joypad) {
	port := number & 1
	data := number & 2
	jc.joypads[port][data].joypad = joypad
}

func (jc *JoypadController) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x4016:
		port := jc.joypads[0]
		return (jc.bus.GetOpenBus() & 0xFC) | (port[1].ReadNextKey(jc.state) << 1) | port[0].ReadNextKey(jc.state), nil
	case 0x4017:
		port := jc.joypads[1]
		return (jc.bus.GetOpenBus() & 0xE0) | 0x1C | (port[1].ReadNextKey(jc.state) << 1) | port[0].ReadNextKey(jc.state), nil
	default:
		return 0, fmt.Errorf("invalid internal Joypad register read at $%04X", addr)
	}
}

func (jc *JoypadController) Write(addr uint16, value byte) error {
	switch addr {
	case 0x4016:
		if value&1 == 1 {
			jc.state = JOYPAD_LATCHED
		} else {
			jc.state = JOYPAD_UNLATCHED
		}
		for i := range jc.joypads {
			for j := range jc.joypads[i] {
				jc.joypads[i][j].Reset(jc.state)
			}
		}
	default:
		return fmt.Errorf("invalid internal Joypad register write at $%04X", addr)
	}
	return nil
}
