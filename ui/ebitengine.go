package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const BufferHeight = 478
const BufferWidth = 256
const BufferWidthShift = 8

const MaxScreenHeight = BufferHeight
const MaxScreenWidth = BufferWidth * 2
const ScalingFactor = 1.5

type SnesColorData struct {
	Color1, Color2 uint16
	Brightness     byte
}

func (scd *SnesColorData) SetColor(color1, color2 uint16, brightness byte) {
	scd.Color1, scd.Color2, scd.Brightness = color1, color2, brightness
}

type Framebuffer struct {
	front, Back *[BufferWidth][BufferHeight]SnesColorData
	swap        chan *[BufferWidth][BufferHeight]SnesColorData

	CurrentHeight int
	Interlace     byte
}

func NewFramebuffer() *Framebuffer {
	fb := &Framebuffer{
		front:         new([BufferWidth][BufferHeight]SnesColorData),
		Back:          new([BufferWidth][BufferHeight]SnesColorData),
		swap:          make(chan *[BufferWidth][BufferHeight]SnesColorData, 1),
		CurrentHeight: 224,
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
	transformedBuffer []byte

	ScreenWidth  int
	ScreenHeight int
	ActiveImage  *ebiten.Image

	Controller1 SnesInput
}

func NewEmulatorDisplay(fb *Framebuffer) *EmulatorDisplay {
	return &EmulatorDisplay{
		fb:                fb,
		ActiveImage:       updateActiveImage(MaxScreenHeight),
		transformedBuffer: make([]byte, 4*MaxScreenWidth*MaxScreenHeight),
		ScreenWidth:       MaxScreenWidth,
		ScreenHeight:      MaxScreenHeight,

		Controller1: NewSnesControllerInput(0),
	}
}

func updateActiveImage(height int) *ebiten.Image {
	activeImage := ebiten.NewImage(MaxScreenWidth, height)
	ebiten.SetWindowSize(int(float64(MaxScreenWidth)*ScalingFactor), int(float64(height)*ScalingFactor))

	return activeImage
}

func (ed *EmulatorDisplay) Update() error {
	select {
	case frame := <-ed.fb.swap:

		newHeight := ed.fb.CurrentHeight << 1
		if ed.ScreenHeight != newHeight {
			ed.ScreenHeight = newHeight
			ed.ActiveImage = updateActiveImage(newHeight)
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
	op := &ebiten.DrawImageOptions{}
	scaleX := float64(1)
	scaleY := float64(int(2 >> ed.fb.Interlace))
	op.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(ed.ActiveImage, op)
}

func (ed *EmulatorDisplay) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ed.ScreenWidth, ed.ScreenHeight
}

func (ed *EmulatorDisplay) convertBGR15ToRGBA(buffer *[BufferWidth][BufferHeight]SnesColorData) {
	for y := 0; y < ed.ScreenHeight>>(int((ed.fb.Interlace+1)&1)); y++ {
		for x := 0; x < BufferWidth; x++ {
			v := buffer[x][y]
			i := (y<<BufferWidthShift + x) << 3

			r := float32(v.Color1 & 0x1F << 3)
			g := float32((v.Color1 >> 5) & 0x1F << 3)
			b := float32((v.Color1 >> 10) & 0x1F << 3)

			scale := float32(v.Brightness) / 15

			ed.transformedBuffer[i+0] = byte(r * scale)
			ed.transformedBuffer[i+1] = byte(g * scale)
			ed.transformedBuffer[i+2] = byte(b * scale)
			ed.transformedBuffer[i+3] = 0xFF // alpha always fully opaque

			r = float32(v.Color2 & 0x1F << 3)
			g = float32((v.Color2 >> 5) & 0x1F << 3)
			b = float32((v.Color2 >> 10) & 0x1F << 3)

			ed.transformedBuffer[i+4] = byte(r * scale)
			ed.transformedBuffer[i+5] = byte(g * scale)
			ed.transformedBuffer[i+6] = byte(b * scale)
			ed.transformedBuffer[i+7] = 0xFF
		}
	}
}
