//go:build !amd64 && !arm64 && !386

package ui

// the idea is to have the two uint16 color data and brightness be uploaded directly
// to the gpu as is (bgr, brightness, 0, bgr, brightness, 0)
// this pretends to be 2xRGBA sample and then the gpu applies a shader in order to convert it
// eliminating this task from the cpu. this also immediately puts everything into the correct
// []byte slice format. Used if endianness is unknown.
func (fb *Framebuffer) WriteDot(color1, color2 uint16, brightness byte) {
	idx := fb.backPointerIdx - uintptr(fb.backPointerBase)
	fb.Back[idx] = byte(color1)
	fb.Back[idx+1] = byte(color1 >> 8)
	fb.Back[idx+2] = brightness
	fb.Back[idx+4] = byte(color2)
	fb.Back[idx+5] = byte(color2 >> 8)
	fb.Back[idx+6] = brightness

	fb.backPointerIdx += 8

	if fb.backPointerIdx&0x7FF == 0 {
		fb.lineCnt++
		if fb.Interlace == 1 {
			if fb.lineCnt == fb.CurrentHeight {
				fb.backPointerIdx = uintptr(fb.backPointerBase) + 0x800
			} else {
				fb.backPointerIdx += 0x800
			}
		}
	}
}
