package apu

import (
	"SNES_emulator/apu/spc700"
	"SNES_emulator/memory"
	"encoding/binary"
	"io"

	"github.com/ebitengine/oto/v3"
)

var (
	audioWriter *io.PipeWriter
	otoContext  *oto.Context
	otoPlayer   *oto.Player
)

func InitAudio() {
	pr, pw := io.Pipe()
	audioWriter = pw

	options := &oto.NewContextOptions{
		SampleRate:   32000,
		ChannelCount: 1, // Mono for now
		Format:       oto.FormatSignedInt16LE,
	}

	var ready chan struct{}
	var err error
	otoContext, ready, err = oto.NewContext(options)
	if err != nil {
		panic(err)
	}
	<-ready

	otoPlayer = otoContext.NewPlayer(pr)
	otoPlayer.Play()
}

var SampleChan = make(chan int16, 64000)

func StartAudioThread() {
	pr, pw := io.Pipe()

	options := &oto.NewContextOptions{
		SampleRate:   33558, //33252 for pal, 33558 for ntsc
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	}
	context, ready, _ := oto.NewContext(options)
	<-ready

	player := context.NewPlayer(pr)
	player.Play()

	go func() {
		buf := make([]byte, 2)
		for sample := range SampleChan {
			err := player.Err()

			if err != nil {
				panic(err)
			}
			binary.LittleEndian.PutUint16(buf, uint16(sample))
			pw.Write(buf)
		}
	}()
}

type APU struct {
	psram *SPCMemory
	dsp   *DSP
	cpu   *spc700.CPU
}

func NewApu(bus memory.Bus) *APU {
	psram := NewSPCMemory()
	ret := &APU{
		psram: psram,
		dsp:   NewDsp(psram),
		cpu:   spc700.NewCPU(psram),
	}

	psram.dspRegs = ret.dsp

	//probably the cleanest way
	bus.RegisterRange(0x2140, 0x217F, psram, "APU")
	return ret
}

func (apu *APU) Step() {
	apu.cpu.StepCycle()
	apu.dsp.Step()
	apu.psram.TickTimers()
}
