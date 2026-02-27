package ui

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"

	_ "embed"
)

const BufferHeight = 478
const BufferWidthShift = 8

const ScreenWidth = 1 << (BufferWidthShift + 1)

//go:embed shaders/bgr15.kage
var shaderSource []byte

var bgrShader *ebiten.Shader

func init() {
	var err error
	bgrShader, err = ebiten.NewShader(shaderSource)
	if err != nil {
		panic(string("Kage: Shader compilation failed. " + err.Error()))
	}
}

type UiConfig interface {
	GetDisplayScale() float64
	GetInputMapping() []SnesInput
}

type Framebuffer struct {
	swap chan *[BufferHeight << (BufferWidthShift + 3)]byte
	f, B *[BufferHeight << (BufferWidthShift + 3)]byte //H*512*4

	backPointerBase unsafe.Pointer
	backPointerIdx  uintptr
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
	fb.backPointerBase = unsafe.Pointer(&fb.B[0])
	fb.backPointerIdx = uintptr(fb.backPointerBase)
	return fb
}

// the idea is to have the two uint16 color data and brightness be uploaded directly
// to the gpu as is (bgr, brightness, 0, bgr, brightness, 0)
// this pretends to be 2xRGBA sample and then the gpu applies a shader in order to convert it
// eliminating this task from the cpu. this also immediately puts everything into the correct
// []byte slice format. Doesnt work on big endian systems
func (fb *Framebuffer) WriteDot(color1, color2 uint16, brightness byte) {
	currentPointer := unsafe.Pointer(uintptr(fb.backPointerIdx))
	*(*uint64)(currentPointer) = (uint64(color1) | uint64(brightness)<<16 |
		uint64(color2)<<32 | uint64(brightness)<<48)

	fb.backPointerIdx += 8

	if fb.Interlace == 1 && fb.backPointerIdx&0x7FF == 0 {
		fb.lineCnt++
		if fb.lineCnt == fb.CurrentHeight {
			fb.backPointerIdx = uintptr(fb.backPointerBase) + 0x800
		} else {
			fb.backPointerIdx += 0x800
		}
	}
}

func (fb *Framebuffer) Swap() {
	fb.f, fb.B = fb.B, fb.f

	fb.backPointerBase = unsafe.Pointer(&fb.B[0])
	fb.backPointerIdx = uintptr(fb.backPointerBase)
	fb.lineCnt = 0

	select {
	case fb.swap <- fb.f:
	default:
		//non blocking send
	}
}

type EmulatorDisplay struct {
	fb *Framebuffer

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
		fb:            fb,
		ActiveImage:   updateActiveImage(BufferHeight, displayScale),
		ScreenWidth:   ScreenWidth,
		ScreenHeight:  BufferHeight,
		ScalingFactor: displayScale,

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
