package ui

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
)

const BufferHeight = 478
const BufferWidthShift = 8

const ScreenWidth = 1 << (BufferWidthShift + 1)

const shaderSource = `
//kage:unit pixels
package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	raw := imageSrc0UnsafeAt(srcPos)

	val := int(raw.r * 255.0) | (int(raw.g * 255.0)<<8)

	r := float((val&0x1F)<<3)/255.0
	g := float(((val>>5)&0x1F)<<3)/255.0
	b := float(((val>>10)&0x1F)<<3)/255.0

	brightness := (raw.b*255)/15

	return vec4(r*brightness, g*brightness, b*brightness, 1.0)
}
`

var bgrShader *ebiten.Shader

func init() {
	var err error
	bgrShader, err = ebiten.NewShader([]byte(shaderSource))
	if err != nil {
		panic("Kage: Shader compilation failed.")
	}
}

type UiConfig interface {
	GetDisplayScale() float64
	GetInputMapping() []SnesInput
}

type Framebuffer struct {
	swap chan *[BufferHeight << (BufferWidthShift + 3)]byte
	f, B *[BufferHeight << (BufferWidthShift + 3)]byte //H*512*4

	backPointer     unsafe.Pointer
	backPointerBase unsafe.Pointer
	lineCnt         int

	CurrentHeight int
	Interlace     byte
}

func NewFramebuffer() *Framebuffer {
	fb := &Framebuffer{
		swap:          make(chan *[BufferHeight << (BufferWidthShift + 3)]byte, 1),
		CurrentHeight: 224,

		f: new([BufferHeight << (BufferWidthShift + 3)]byte),
		B: new([BufferHeight << (BufferWidthShift + 3)]byte),
	}
	fb.backPointer = unsafe.Pointer(&fb.B[0])
	fb.backPointerBase = unsafe.Pointer(&fb.B[0])
	return fb
}

func (fb *Framebuffer) WriteDot(color1, color2 uint16, brightness byte) {
	*(*uint64)(fb.backPointer) = (uint64(color1) | uint64(brightness)<<16 |
		uint64(color2)<<32 | uint64(brightness)<<48)
	step := 8 + uintptr(fb.backPointer)
	if fb.Interlace == 1 && step&0x7FF == 0 {
		fb.lineCnt++
		step += 0x800
		if fb.lineCnt == fb.CurrentHeight {
			step = uintptr(fb.backPointerBase) + 0x800
		}
	}
	fb.backPointer = unsafe.Pointer(uintptr(step))
}

func (fb *Framebuffer) Swap() {
	fb.lineCnt = 0
	fb.backPointer = unsafe.Pointer(&fb.B[0])
	fb.backPointerBase = unsafe.Pointer(&fb.B[0])
	fb.f, fb.B = fb.B, fb.f

	select {
	case fb.swap <- fb.f:
	default:
		//non blocking send
	}
}

type EmulatorDisplay struct {
	fb                *Framebuffer
	transformedBuffer []byte

	ScreenWidth   int
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
		ActiveImage:       updateActiveImage(BufferHeight, displayScale),
		transformedBuffer: make([]byte, 4*ScreenWidth*BufferHeight),
		ScreenWidth:       ScreenWidth,
		ScreenHeight:      BufferHeight,
		ScalingFactor:     displayScale,

		Controller0: controllers[0],
		Controller1: controllers[1],
		Controller2: controllers[2],
		Controller3: controllers[3],
	}
}

func updateActiveImage(height int, scalingFactor float64) *ebiten.Image {
	activeImage := ebiten.NewImage(ScreenWidth, height)
	ebiten.SetWindowSize(int(float64(ScreenWidth)*scalingFactor), int(float64(height)*scalingFactor))

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

		ed.ActiveImage.WritePixels(frame[:(ed.ScreenWidth*ed.ScreenHeight)<<2])
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
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = ed.ActiveImage
	scaleY := float64(int(2 >> ed.fb.Interlace))
	op.GeoM.Scale(1.0, scaleY)
	screen.DrawRectShader(ed.ScreenWidth, ed.ScreenHeight, bgrShader, op)
}

func (ed *EmulatorDisplay) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ed.ScreenWidth, ed.ScreenHeight
}
