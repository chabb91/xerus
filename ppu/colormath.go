package ppu

import "image/color"

func SNESColorToARGB(snesColor uint16) color.NRGBA {
	red := byte((snesColor & 0x1F) << 3)
	green := byte(((snesColor >> 5) & 0x1F) << 3)
	blue := byte(((snesColor >> 10) & 0x1F) << 3)
	return color.NRGBA{
		R: red,
		G: green,
		B: blue,
		A: 0xFF,
	}
}
