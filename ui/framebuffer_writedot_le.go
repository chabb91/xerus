//go:build amd64 || arm64 || 386

package ui

import "unsafe"

// the idea is to have the two uint16 color data and brightness be uploaded directly
// to the gpu as is (bgr, brightness, 0, bgr, brightness, 0)
// this pretends to be 2xRGBA sample and then the gpu applies a shader in order to convert it
// eliminating this task from the cpu. this also immediately puts everything into the correct
// []byte slice format. Used with known little endian architectures.
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
