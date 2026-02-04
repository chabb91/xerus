package apu

import "fmt"

var Recording []int16

type DSP struct {
	state int

	Voices [8]*Voice
}

func NewDsp(psram *SPCMemory) *DSP {
	ret := &DSP{}
	for i := range len(ret.Voices) {
		ret.Voices[i] = newVoice(i, psram)
	}
	Recording = make([]int16, 0, 1000000)
	return ret
}

func (dsp *DSP) Step() {
	dsp.state++
	if dsp.state <= 31 {
		return
	}
	out := dsp.Voices[0].Tick()
	Recording = append(Recording, out)
	fmt.Println("SAMPLE: ", out)
	dsp.state = 0
}
