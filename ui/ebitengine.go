package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const DefaultWidth = 256
const DefaultHeight = 224
const MaxWidth = 512
const MaxHeight = 478
const ScalingFactor = 3

type SnesColorData struct {
	Color      uint16
	Brightness byte
}

func (scd *SnesColorData) SetColor(color uint16, brightness byte) {
	scd.Color, scd.Brightness = color, brightness
}

type Framebuffer struct {
	front, Back *[MaxWidth][MaxHeight]SnesColorData
	swap        chan *[MaxWidth][MaxHeight]SnesColorData

	CurrentWidth  int
	CurrentHeight int
}

func NewFramebuffer() *Framebuffer {
	fb := &Framebuffer{
		front:         new([MaxWidth][MaxHeight]SnesColorData),
		Back:          new([MaxWidth][MaxHeight]SnesColorData),
		swap:          make(chan *[MaxWidth][MaxHeight]SnesColorData, 1),
		CurrentWidth:  DefaultWidth,
		CurrentHeight: DefaultHeight,
	}
	return fb
}

func (fb *Framebuffer) Swap() {
	fb.front, fb.Back = fb.Back, fb.front

	select {
	case fb.swap <- fb.front:
	default:
		//non blocking send
	}
}

type EmulatorDisplay struct {
	fb                *Framebuffer
	img               *ebiten.Image
	transformedBuffer []byte

	ScreenWidth  int
	ScreenHeight int
	ActiveImage  *ebiten.Image

	Controller1 SnesInput
}

func NewEmulatorDisplay(fb *Framebuffer) *EmulatorDisplay {
	return &EmulatorDisplay{
		fb:                fb,
		img:               ebiten.NewImage(MaxWidth, MaxHeight),
		transformedBuffer: make([]byte, 4*MaxWidth*MaxHeight),
		ScreenWidth:       MaxWidth,
		ScreenHeight:      MaxHeight,

		Controller1: NewSnesControllerInput(0),
	}
}

func (ed *EmulatorDisplay) Update() error {
	select {
	case frame := <-ed.fb.swap:

		newWidth := ed.fb.CurrentWidth
		newHeight := ed.fb.CurrentHeight
		if ed.ScreenWidth != newWidth || ed.ScreenHeight != newHeight {
			ed.ScreenWidth = newWidth
			ed.ScreenHeight = newHeight
			ed.ActiveImage = ebiten.NewImage(newWidth, newHeight)
			ebiten.SetWindowSize(newWidth*ScalingFactor, newHeight*ScalingFactor)
		}
		ed.convertBGR15ToRGBA(frame)

		activePixelsSlice := ed.transformedBuffer[:ed.ScreenWidth*ed.ScreenHeight*4]
		ed.ActiveImage.WritePixels(activePixelsSlice)
	default:
		// no new frame yet
	}

	ed.Controller1.UpdateControllerState()
	return nil
}

func (ed *EmulatorDisplay) Draw(screen *ebiten.Image) {
	if ed.ActiveImage == nil {
		return
	}
	screen.DrawImage(ed.ActiveImage, nil)
}

func (ed *EmulatorDisplay) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ed.ScreenWidth, ed.ScreenHeight
}

func (ed *EmulatorDisplay) convertBGR15ToRGBA(buffer *[MaxWidth][MaxHeight]SnesColorData) {
	for y := 0; y < ed.ScreenHeight; y++ {
		for x := 0; x < ed.ScreenWidth; x++ {
			v := buffer[x][y]
			i := (y*ed.ScreenWidth + x) << 2

			r := float32(v.Color&0x1F) * 8
			g := float32((v.Color>>5)&0x1F) * 8
			b := float32((v.Color>>10)&0x1F) * 8

			scale := float32(v.Brightness) / 15

			ed.transformedBuffer[i+0] = byte(r * scale)
			ed.transformedBuffer[i+1] = byte(g * scale)
			ed.transformedBuffer[i+2] = byte(b * scale)
			ed.transformedBuffer[i+3] = 0xFF // alpha always fully opaque
		}
	}
}
