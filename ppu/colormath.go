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

func addColors(main, sub uint16, halve bool) uint16 {
	r := main>>10&31 + (sub >> 10 & 31)
	g := main>>5&31 + (sub >> 5 & 31)
	b := main&31 + (sub & 31)

	if halve {
		r >>= 1
		g >>= 1
		b >>= 1
	}

	return uint16((r & 31 << 10) | (g & 31 << 5) | b&31)
}

func subColors(main, sub uint16, halve bool) uint16 {
	r := main>>10&31 - (sub >> 10 & 31)
	g := main>>5&31 - (sub >> 5 & 31)
	b := main&31 - (sub & 31)

	if halve {
		r >>= 1
		g >>= 1
		b >>= 1
	}

	return uint16((r & 31 << 10) | (g & 31 << 5) | b&31)
}
