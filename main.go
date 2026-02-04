package main

import (
	"SNES_emulator/apu"
	"SNES_emulator/internal/config"
	"SNES_emulator/soc"
	"SNES_emulator/ui"
	"encoding/binary"
	"log"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	config := config.New()
	if config.IsPProfEnabled() {
		f, err := os.Create(config.GetPProfPath())
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	fb := ui.NewFramebuffer()
	display := ui.NewEmulatorDisplay(fb, config)

	soc := soc.NewSoC(config, fb, display.Controller0, display.Controller1)
	defer soc.Cartridge.SaveSramToFile() //save Sram(if exists) on exit
	defer func() {
		DumpToWav("debug_beep.wav", apu.Recording, 32000)
	}()

	ebiten.SetWindowTitle(soc.Cartridge.GetRomName())

	go soc.Run()
	ebiten.RunGame(display)
}

func DumpToWav(filename string, samples []int16, sampleRate int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	numSamples := len(samples)
	bitsPerSample := 16
	numChannels := 1 // Mono for a single voice
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := numSamples * blockAlign

	// --- WRITE WAV HEADER ---
	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVE"))

	// "fmt " subchunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16)) // Subchunk1Size (16 for PCM)
	binary.Write(f, binary.LittleEndian, uint16(1))  // AudioFormat (1 for PCM)
	binary.Write(f, binary.LittleEndian, uint16(numChannels))
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))

	// "data" subchunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	// --- WRITE DATA ---
	return binary.Write(f, binary.LittleEndian, samples)
}
