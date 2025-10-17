package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const ScreenWidth = 256
const ScreenHeight = 224
const ScalingFactor = 3

type Framebuffer struct {
	front, Back *[ScreenWidth][ScreenHeight]uint16
	swap        chan *[ScreenWidth][ScreenHeight]uint16
	Brightness  byte
}

func NewFramebuffer() *Framebuffer {
	fb := &Framebuffer{
		front: new([ScreenWidth][ScreenHeight]uint16),
		Back:  new([ScreenWidth][ScreenHeight]uint16),
		swap:  make(chan *[ScreenWidth][ScreenHeight]uint16, 1),
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
}

func NewEmulatorDisplay(fb *Framebuffer) *EmulatorDisplay {
	return &EmulatorDisplay{
		fb:                fb,
		img:               ebiten.NewImage(ScreenWidth, ScreenHeight),
		transformedBuffer: make([]byte, 4*ScreenWidth*ScreenHeight),
	}
}

func (ed *EmulatorDisplay) Update() error {
	select {
	case frame := <-ed.fb.swap:
		ed.convertBGR15ToRGBA(frame)
	default:
		// no new frame yet
	}
	return nil
}

func (ed *EmulatorDisplay) Draw(screen *ebiten.Image) {
	ed.img.WritePixels(ed.transformedBuffer)
	screen.DrawImage(ed.img, nil)
}

func (ed *EmulatorDisplay) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (ed *EmulatorDisplay) convertBGR15ToRGBA(buffer *[ScreenWidth][ScreenHeight]uint16) {
	//doesnt work for some reason FIXME
	brightness := (ed.fb.Brightness * 17) & 0xFF

	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			v := buffer[x][y]
			i := (y*ScreenWidth + x) * 4

			ed.transformedBuffer[i] = byte((v & 0x1F) << 3)
			ed.transformedBuffer[i+1] = byte(((v >> 5) & 0x1F) << 3)
			ed.transformedBuffer[i+2] = byte(((v >> 10) & 0x1F) << 3)
			ed.transformedBuffer[i+3] = byte(brightness)
		}
	}
}
