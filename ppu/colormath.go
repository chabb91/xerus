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
	halfShift := uint16(0)
	if halve {
		halfShift = 1
	}

	b := min((main>>10&31+(sub>>10&31))>>halfShift, 0x1F)
	g := min((main>>5&31+(sub>>5&31))>>halfShift, 0x1F)
	r := min((main&31+(sub&31))>>halfShift, 0x1F)

	return (b << 10) | (g << 5) | r
}

// the result is shifted to the right (after ?) clipping to 0
// the docs are unsure
// tested using bbbradsmith's colormath test rom
func subColors(main, sub uint16, halve bool) uint16 {
	halfShift := int32(0)
	if halve {
		halfShift = 1
	}
	b := max(int32(main>>10&31)-int32((sub>>10&31)), 0) >> halfShift
	g := max(int32(main>>5&31)-int32((sub>>5&31)), 0) >> halfShift
	r := max(int32(main&31)-int32((sub&31)), 0) >> halfShift

	return uint16((b << 10) | (g << 5) | r)
}

func mode0ColorIndex(layer ppuLayer, _ colorDepth, palette byte) byte {
	return palette<<2 + byte(layer<<5)
}

func modeNormalColorNo8bppIndex(_ ppuLayer, colorDepth colorDepth, palette byte) byte {
	return palette << colorDepth
}

func modeNormalColor8BppIndex(_ ppuLayer, _ colorDepth, _ byte) byte {
	return 0
}
