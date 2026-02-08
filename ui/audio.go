package ui

import (
	"io"

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
			ChannelCount: 1, //mono for now
			Format:       oto.FormatSignedInt16LE,
		}
		context, ready, _ := oto.NewContext(options)
		<-ready

		audio = &emulatorAudio{otoContext: context}
	}

	return audio
}

func (ea *emulatorAudio) Play(buffer io.Reader) {
	ea.otoPlayer = ea.otoContext.NewPlayer(buffer)
	ea.otoPlayer.Play()
}
