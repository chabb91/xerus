package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const BufferHeight = 478
const BufferWidthShift = 8
const BufferWidth = 1 << BufferWidthShift

const MaxScreenHeight = BufferHeight
const MaxScreenWidth = BufferWidth * 2

type UiConfig interface {
	GetDisplayScale() float64
	GetInputMapping() []SnesInput
}

type SnesColorData struct {
	Color1, Color2 uint16
	Brightness     byte
}

func (scd *SnesColorData) SetColor(color1, color2 uint16, brightness byte) {
	scd.Color1, scd.Color2, scd.Brightness = color1, color2, brightness
}

type Framebuffer struct {
	front, Back *[BufferHeight][BufferWidth]SnesColorData
	swap        chan *[BufferHeight][BufferWidth]SnesColorData

	CurrentHeight int
	Interlace     byte
}

func NewFramebuffer() *Framebuffer {
	fb := &Framebuffer{
		front: new([BufferHeight][BufferWidth]SnesColorData),
		Back:  new([BufferHeight][BufferWidth]SnesColorData),
		swap:  make(chan *[BufferHeight][BufferWidth]SnesColorData, 1),
		//CurrentHeight is set up on ppu init
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

	ScreenHeight  int
	ScalingFactor float64
	ActiveImage   *ebiten.Image

	Controller0 SnesInput
	Controller1 SnesInput
	Controller2 SnesInput
	Controller3 SnesInput
}

func NewEmulatorDisplay(fb *Framebuffer, config UiConfig) *EmulatorDisplay {
	displayScale := config.GetDisplayScale()
	controllers := config.GetInputMapping()
	return &EmulatorDisplay{
		fb:                fb,
		ActiveImage:       updateActiveImage(MaxScreenHeight, displayScale),
		transformedBuffer: make([]byte, 4*MaxScreenWidth*MaxScreenHeight),
		ScreenHeight:      MaxScreenHeight,
		ScalingFactor:     displayScale,

		Controller0: controllers[0],
		Controller1: controllers[1],
		Controller2: controllers[2],
		Controller3: controllers[3],
	}
}

func updateActiveImage(height int, scalingFactor float64) *ebiten.Image {
	activeImage := ebiten.NewImage(MaxScreenWidth, height)
	ebiten.SetWindowSize(int(float64(MaxScreenWidth)*scalingFactor), int(float64(height)*scalingFactor))

	return activeImage
}

func (ed *EmulatorDisplay) Update() error {
	select {
	case frame := <-ed.fb.swap:

		newHeight := ed.fb.CurrentHeight << 1
		if ed.ScreenHeight != newHeight {
			ed.ScreenHeight = newHeight
			ed.ActiveImage = updateActiveImage(newHeight, ed.ScalingFactor)
		}
		ed.convertBGR15ToRGBA(frame)
		ed.ActiveImage.WritePixels(ed.transformedBuffer[:MaxScreenWidth*ed.ScreenHeight*4])
	default:
		// no new frame yet
	}

	ed.Controller0.UpdateControllerState()
	ed.Controller1.UpdateControllerState()
	ed.Controller2.UpdateControllerState()
	ed.Controller3.UpdateControllerState()
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
	return MaxScreenWidth, ed.ScreenHeight
}

func (ed *EmulatorDisplay) convertBGR15ToRGBA(buffer *[BufferHeight][BufferWidth]SnesColorData) {
	renderedRows := ed.ScreenHeight >> (ed.fb.Interlace ^ 1)
	for y := 0; y < renderedRows; y++ {
		bufferRow := &buffer[y]
		for x := 0; x < BufferWidth; x++ {
			v := bufferRow[x]
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
