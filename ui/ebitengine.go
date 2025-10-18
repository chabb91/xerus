package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const DefaultWidth = 256
const DefaultHeight = 224
const MaxWidth = 512
const MaxHeight = 478
const ScalingFactor = 3

type Framebuffer struct {
	front, Back *[MaxWidth][MaxHeight]uint16
	swap        chan *[MaxWidth][MaxHeight]uint16
	Brightness  byte

	CurrentWidth  int
	CurrentHeight int
}

func NewFramebuffer() *Framebuffer {
	fb := &Framebuffer{
		front:         new([MaxWidth][MaxHeight]uint16),
		Back:          new([MaxWidth][MaxHeight]uint16),
		swap:          make(chan *[MaxWidth][MaxHeight]uint16, 1),
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
}

func NewEmulatorDisplay(fb *Framebuffer) *EmulatorDisplay {
	return &EmulatorDisplay{
		fb:                fb,
		img:               ebiten.NewImage(MaxWidth, MaxHeight),
		transformedBuffer: make([]byte, 4*MaxWidth*MaxHeight),
		ScreenWidth:       MaxWidth,
		ScreenHeight:      MaxHeight,
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

func (ed *EmulatorDisplay) convertBGR15ToRGBA(buffer *[MaxWidth][MaxHeight]uint16) {
	//doesnt work for some reason FIXME
	brightness := (ed.fb.Brightness * 17) & 0xFF

	for y := 0; y < ed.ScreenHeight; y++ {
		for x := 0; x < ed.ScreenWidth; x++ {
			v := buffer[x][y]
			i := (y*ed.ScreenWidth + x) * 4

			ed.transformedBuffer[i] = byte((v & 0x1F) << 3)
			ed.transformedBuffer[i+1] = byte(((v >> 5) & 0x1F) << 3)
			ed.transformedBuffer[i+2] = byte(((v >> 10) & 0x1F) << 3)
			ed.transformedBuffer[i+3] = byte(brightness)
		}
	}
}
