package soc

import (
	"fmt"

	"github.com/chabb91/xerus/memory"
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

	dataLineId byte
}

func (jdh *JoypadDataHandler) ReadNextKey(state int) byte {
	if jdh.positionCnt >= 16 {
		//hardware returns 1 here for the data 0 line and 0 for data 1 line on each port
		//not doing this can break input handling for some games
		return jdh.dataLineId ^ 1
	}
	var ret byte
	if jdh.joypad == nil {
		ret = 0
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

func NewJoypadController(bus memory.Bus, joypads []Joypad) *JoypadController {
	jc := &JoypadController{
		bus: bus,
	}

	//setting up the return value after 16 reads.
	jc.joypads[0][1].dataLineId = 1
	jc.joypads[1][1].dataLineId = 1

	jc.AttachMultiple(joypads)

	bus.RegisterRange(0x4016, 0x4017, jc, "Joypad")
	return jc
}

// joypad id [0-3],
// null safe,
// port 0, 1, 0, 1
// (p0:l0, p1:l0, p0:l1, p1:l1)
func (jc *JoypadController) Attach(number int, joypad Joypad) {
	port := number & 1
	data := (number & 2) >> 1
	jc.joypads[port][data].joypad = joypad
}

func (jc *JoypadController) AttachMultiple(joypads []Joypad) {
	amount := min(len(joypads), 4)
	for i := range amount {
		jc.Attach(i, joypads[i])
	}
}

func (jc *JoypadController) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x4016:
		port := &jc.joypads[0]
		return (jc.bus.GetOpenBus() & 0xFC) | (port[1].ReadNextKey(jc.state) << 1) | port[0].ReadNextKey(jc.state), nil
	case 0x4017:
		port := &jc.joypads[1]
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
