package ui

import (
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
)

var audio *emulatorAudio

type emulatorAudio struct {
	otoContext *oto.Context
	otoPlayer  *oto.Player
}

// Creates a package level context and player to avoid it getting garbage collected
// essentially singleton
func GetEmulatorAudio() *emulatorAudio {
	if audio == nil {
		options := &oto.NewContextOptions{
			SampleRate:   32000,
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   time.Millisecond * 16,
		}
		context, ready, _ := oto.NewContext(options)
		<-ready

		audio = &emulatorAudio{otoContext: context}
	}

	return audio
}

func (ea *emulatorAudio) Play(buffer io.Reader) {
	ea.otoPlayer = ea.otoContext.NewPlayer(buffer)
	ea.otoPlayer.SetBufferSize(2000)
	ea.otoPlayer.Play()
}
