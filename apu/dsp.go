package apu

import (
	"fmt"
)

var Recording = make([]int16, 0, 1000000)
var localBuf []byte

const DSP_REG_SIZE = 0x80

type DSPInterface interface {
	ReadRegister(reg byte) byte
	WriteRegister(reg byte, val byte)
}

type DSP struct {
	state     int
	registers [DSP_REG_SIZE]byte

	Voices [8]*Voice
}

func NewDsp(psram *SPCMemory) *DSP {
	dsp := &DSP{}
	for i := range len(dsp.Voices) {
		dsp.Voices[i] = newVoice(i, &dsp.registers, &psram.ram)
	}

	return dsp
}

/*
func (dsp *DSP) Step() {
	dsp.state++
	if dsp.state <= 31 {
		return
	}
	out := dsp.Voices[0].Tick()
	//Recording = append(Recording, out)
	//fmt.Println("SAMPLE: ", out)
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(out))
	localBuf = append(localBuf, b...)

	// Flush to the system every ~500 samples
	if len(localBuf) >= 1000 {
		_, err := audioWriter.Write(localBuf)
		if err != nil {
			fmt.Println("Audio Write Error:", err)
		}
		localBuf = localBuf[:0]
	}
	dsp.state = 0
}
*/

func (dsp *DSP) Step() {
	dsp.state++
	if dsp.state <= 31 {
		return
	}

	out := dsp.Voices[0].Tick()

	select {
	case SampleChan <- out:
	default:
		// Audio thread is falling behind!
	}

	dsp.state = 0
}

func (d *DSP) ReadRegister(reg byte) byte {
	return d.registers[reg&0x7F]
}

func (d *DSP) WriteRegister(reg byte, val byte) {
	if reg <= 0x7F {
		d.registers[reg] = val

		v := d.Voices[0]
		if reg == 0x4C && val&1 == 1 {
			fmt.Println("KEYON: ", val)
			v.keyOn()
		}
		if reg == 0x5C && val&1 == 1 {
			fmt.Println("KEYOFF: ", val)
			v.keyOff()
		}
	}

}
