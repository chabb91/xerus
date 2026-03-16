package ppu

import (
	"fmt"
	"image/color"
)

const WINDOW_INVALIDATION_COUNTER = 10

type wMaskLogic func(bool, bool) bool

type LayerWindowData struct {
	w1Active, w2Active                bool
	w1Inverted, w2Inverted            bool
	mainScreenMasked, subScreenMasked bool
	colorMathActive                   bool

	wMaskLogic wMaskLogic

	mainCache [SCREEN_WIDTH]bool
	subCache  [SCREEN_WIDTH]bool
}

type WindowController struct {
	w1LeftPos, w1RightPos byte
	w2LeftPos, w2RightPos byte

	rebuildNeeded                     bool
	dirtyMainWindows, dirtySubWindows byte
	invalidationCounter               int

	layers    [6]LayerWindowData
	ColorMath ColorMath
}

func (wc *WindowController) prepareToRebuild() {
	wc.rebuildNeeded = !wc.ColorMath.windowValid ||
		wc.dirtyMainWindows != 0 ||
		wc.dirtySubWindows != 0
	if wc.rebuildNeeded {
		wc.invalidationCounter = WINDOW_INVALIDATION_COUNTER
	}
}

func (wc *WindowController) setMaskLogic(layer ppuLayer, logic wMaskLogic) {
	//TODO
	//func cant be directly compared so cant invalidate in a smart way here.
	//taking the small performance hit for now ig
	wc.layers[layer].wMaskLogic = logic
	wc.dirtyMainWindows |= 1 << layer
	wc.dirtySubWindows |= 1 << layer
}

func (wc *WindowController) maskMainScreen(layer ppuLayer, shouldMask bool) {
	lwc := &wc.layers[layer]
	if lwc.mainScreenMasked != shouldMask {
		lwc.mainScreenMasked = shouldMask
		wc.dirtyMainWindows |= 1 << layer
	}
}

func (wc *WindowController) maskSubScreen(layer ppuLayer, shouldMask bool) {
	lwc := &wc.layers[layer]
	if lwc.subScreenMasked != shouldMask {
		lwc.subScreenMasked = shouldMask
		wc.dirtySubWindows |= 1 << layer
	}
}

func (wc *WindowController) markLayerDirty(layer ppuLayer) {
	wc.dirtyMainWindows |= 1 << layer
	wc.dirtySubWindows |= 1 << layer
}

func setupLayerMasks(layer *LayerWindowData, value byte) {
	layer.w1Inverted = value&1 != 0
	layer.w1Active = value&2 != 0
	layer.w2Inverted = value&4 != 0
	layer.w2Active = value&8 != 0
}

func wMaskOR(w1, w2 bool) bool {
	return w1 || w2
}
func wMaskAND(w1, w2 bool) bool {
	return w1 && w2
}
func wMaskXOR(w1, w2 bool) bool {
	return w1 != w2
}
func wMaskXNOR(w1, w2 bool) bool {
	return w1 == w2
}

func getWMaskLogic(value byte) wMaskLogic {
	switch value & 0x3 {
	case 0:
		return wMaskOR
	case 1:
		return wMaskAND
	case 2:
		return wMaskXOR
	case 3:
		return wMaskXNOR
	default:
		panic(fmt.Errorf("PPU: Window Mask Logic is in an unexpected state."))
	}
}

func (wc *WindowController) W12SEL(value byte) {
	setupLayerMasks(&wc.layers[bg1], value)
	wc.markLayerDirty(bg1)
	setupLayerMasks(&wc.layers[bg2], value>>4)
	wc.markLayerDirty(bg2)
	wc.prepareToRebuild()
}

func (wc *WindowController) W34SEL(value byte) {
	setupLayerMasks(&wc.layers[bg3], value)
	wc.markLayerDirty(bg3)
	setupLayerMasks(&wc.layers[bg4], value>>4)
	wc.markLayerDirty(bg4)
	wc.prepareToRebuild()
}

func (wc *WindowController) WOBJSEL(value byte) {
	setupLayerMasks(&wc.layers[obj], value)
	wc.markLayerDirty(obj)
	setupLayerMasks(&wc.ColorMath.colorWindowData, value>>4)
	wc.ColorMath.windowValid = false
	wc.prepareToRebuild()
}

func (wc *WindowController) WBGLOG(value byte) {
	wc.setMaskLogic(bg1, getWMaskLogic(value))
	wc.setMaskLogic(bg2, getWMaskLogic(value>>2))
	wc.setMaskLogic(bg3, getWMaskLogic(value>>4))
	wc.setMaskLogic(bg4, getWMaskLogic(value>>6))
	wc.prepareToRebuild()
}

func (wc *WindowController) WOBJLOG(value byte) {
	wc.setMaskLogic(obj, getWMaskLogic(value))
	wc.ColorMath.colorWindowData.wMaskLogic = getWMaskLogic(value >> 2)
	wc.ColorMath.windowValid = false
	wc.prepareToRebuild()
}

func (wc *WindowController) TMW(value byte) {
	wc.maskMainScreen(bg1, value&1 != 0)
	wc.maskMainScreen(bg2, value&2 != 0)
	wc.maskMainScreen(bg3, value&4 != 0)
	wc.maskMainScreen(bg4, value&8 != 0)
	wc.maskMainScreen(obj, value&0x10 != 0)
	wc.prepareToRebuild()
}

func (wc *WindowController) TSW(value byte) {
	wc.maskSubScreen(bg1, value&1 != 0)
	wc.maskSubScreen(bg2, value&2 != 0)
	wc.maskSubScreen(bg3, value&4 != 0)
	wc.maskSubScreen(bg4, value&8 != 0)
	wc.maskSubScreen(obj, value&0x10 != 0)
	wc.prepareToRebuild()
}

func (wc *WindowController) markAllWindowsDirty() {
	wc.dirtyMainWindows = 0xFF
	wc.dirtySubWindows = 0xFF
	wc.ColorMath.windowValid = false
	wc.prepareToRebuild()
}

func isDotInMaskRange(leftPos, rightPos byte, inverted bool, H uint16) bool {
	ret := false
	if leftPos > rightPos {
		ret = false
	} else {
		if uint16(leftPos) < H && uint16(rightPos) > H {
			ret = true
		}
	}

	return ret != inverted //bool xor i.e toggle
}

func (lwd *LayerWindowData) isDotMasked(isScreenMasked bool, H uint16, wc *WindowController) bool {
	if !isScreenMasked {
		return false
	} else {
		if lwd.w1Active && !lwd.w2Active {
			return isDotInMaskRange(wc.w1LeftPos, wc.w1RightPos, lwd.w1Inverted, H)
		}
		if !lwd.w1Active && lwd.w2Active {
			return isDotInMaskRange(wc.w2LeftPos, wc.w2RightPos, lwd.w2Inverted, H)
		}
		if lwd.w1Active && lwd.w2Active {
			w1 := isDotInMaskRange(wc.w1LeftPos, wc.w1RightPos, lwd.w1Inverted, H)
			w2 := isDotInMaskRange(wc.w2LeftPos, wc.w2RightPos, lwd.w2Inverted, H)
			return lwd.wMaskLogic(w1, w2)
		}

		return false //both inactive
	}

}

func (wc *WindowController) rebuildDirtyLayerWindowCaches() {
	for i := range len(wc.layers) {
		val := byte(1 << i)
		layer := &wc.layers[i]
		if wc.dirtyMainWindows&val != 0 && wc.dirtySubWindows&val == 0 {
			for j := range uint16(SCREEN_WIDTH) {
				layer.mainCache[j] = layer.isDotMasked(layer.mainScreenMasked, j, wc)
			}
		} else if wc.dirtyMainWindows&val == 0 && wc.dirtySubWindows&val != 0 {
			for j := range uint16(SCREEN_WIDTH) {
				layer.subCache[j] = layer.isDotMasked(layer.subScreenMasked, j, wc)
			}
		} else if wc.dirtyMainWindows&val != 0 && wc.dirtySubWindows&val != 0 {
			for j := range uint16(SCREEN_WIDTH) {
				layer.mainCache[j] = layer.isDotMasked(layer.mainScreenMasked, j, wc)
				layer.subCache[j] = layer.isDotMasked(layer.subScreenMasked, j, wc)
			}
		}
	}
	if !wc.ColorMath.windowValid {
		cwd := &wc.ColorMath.colorWindowData
		for i := range uint16(SCREEN_WIDTH) {
			cwd.mainCache[i] = cwd.isDotMasked(true, i, wc)
		}
	}

	wc.dirtyMainWindows = 0
	wc.dirtySubWindows = 0
	wc.ColorMath.windowValid = true

	wc.rebuildNeeded = false
}

func (wc *WindowController) setCGADSUB(value byte) {
	if value&0x80 != 0x80 {
		wc.ColorMath.colorFunction = addColors
	} else {
		wc.ColorMath.colorFunction = subColors
	}
	wc.ColorMath.halfColor = value&0x40 == 0x40
	wc.layers[bg1].colorMathActive = value&1 != 0
	wc.layers[bg2].colorMathActive = value&2 != 0
	wc.layers[bg3].colorMathActive = value&4 != 0
	wc.layers[bg4].colorMathActive = value&8 != 0
	wc.layers[obj].colorMathActive = value&0x10 != 0
	wc.layers[backdrop].colorMathActive = value&0x20 != 0
}

type ColorMath struct {
	fixedColor    uint16
	colorFunction colorMathFunction
	halfColor     bool
	isSubscren    bool

	preventMath clipOrPreventMathFunction
	clipToBlack clipOrPreventMathFunction

	//has unused params. was meant for layers, makes setup easy
	windowValid     bool
	colorWindowData LayerWindowData
}

func (cm *ColorMath) setCOLDATA(value byte) {
	if value&0x80 != 0 { //blue
		cm.fixedColor = (cm.fixedColor & 0x3FF) | (uint16(value&0x1F) << 10)
	}
	if value&0x40 != 0 { //green
		cm.fixedColor = (cm.fixedColor & 0xFC1F) | (uint16(value&0x1F) << 5)
	}
	if value&0x20 != 0 { //red
		cm.fixedColor = (cm.fixedColor & 0xFFE0) | (uint16(value & 0x1F))
	}
	cm.fixedColor &= 0x7FFF
}

func (cm *ColorMath) setCGWSEL(value byte, directColor *bool) {
	cm.clipToBlack = getColorClipOrPreventMathMode((value >> 6) & 3)
	cm.preventMath = getColorClipOrPreventMathMode((value >> 4) & 3)
	cm.isSubscren = value&2 != 0
	//assign the value directly to bg1 so there is no lookup delay later
	if directColor != nil {
		*directColor = value&1 != 0
	}
}

type clipOrPreventMathFunction func(bool) bool
type colorMathFunction func(main, sub uint16, halve bool) uint16

func colorClipOrPreventMathNever(_ bool) bool {
	return false
}

func colorClipOrPreventMathOutsideWindow(inColorWindow bool) bool {
	return !inColorWindow
}

func colorClipOrPreventMathInsideWindow(inColorWindow bool) bool {
	return inColorWindow
}

func colorClipOrPreventMathAlways(_ bool) bool {
	return true
}

func getColorClipOrPreventMathMode(value byte) clipOrPreventMathFunction {
	switch value & 0x3 {
	case 0:
		return colorClipOrPreventMathNever
	case 1:
		return colorClipOrPreventMathOutsideWindow
	case 2:
		return colorClipOrPreventMathInsideWindow
	case 3:
		return colorClipOrPreventMathAlways
	default:
		panic(fmt.Errorf("PPU: Color Clip Logic is in an unexpected state."))
	}
}

func (wc *WindowController) performColorMath(mainColor, subColor, H uint16, mainLayer, subLayer ppuLayer) uint16 {
	colorMath := &wc.ColorMath
	inColorMask := colorMath.colorWindowData.mainCache[H]
	if colorMath.clipToBlack(inColorMask) {
		mainColor = 0
	}
	if colorMath.preventMath(inColorMask) || !wc.layers[mainLayer].colorMathActive {
		return mainColor
	}

	//TODO halfcolor might work for fixed color even if ss == backdrop.
	//if thats the case this if has to call colorfunc differently based on subscreen or fixed blend
	var blendColor uint16
	if colorMath.isSubscren && subLayer != backdrop {
		blendColor = subColor
	} else {
		blendColor = colorMath.fixedColor
	}

	return colorMath.colorFunction(
		mainColor,
		blendColor,
		colorMath.halfColor && mainLayer != backdrop && subLayer != backdrop,
	)
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
