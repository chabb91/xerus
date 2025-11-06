package ppu

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

	dirtyMainWindows, dirtySubWindows byte
	invalidationCounter               int

	layers    [7]LayerWindowData
	ColorMath ColorMath
}

func (wc *WindowController) setMaskLogic(layer ppuLayer, logic wMaskLogic) {
	wc.layers[layer].wMaskLogic = logic
	wc.dirtyMainWindows |= 1 << layer
	wc.dirtySubWindows |= 1 << layer
	wc.invalidationCounter = WINDOW_INVALIDATION_COUNTER
}

func (wc *WindowController) maskMainScreen(layer ppuLayer, shouldMask bool) {
	wc.layers[layer].mainScreenMasked = shouldMask
	wc.dirtyMainWindows |= 1 << layer
	wc.invalidationCounter = WINDOW_INVALIDATION_COUNTER
}

func (wc *WindowController) maskSubScreen(layer ppuLayer, shouldMask bool) {
	wc.layers[layer].subScreenMasked = shouldMask
	wc.dirtySubWindows |= 1 << layer
	wc.invalidationCounter = WINDOW_INVALIDATION_COUNTER
}

func (wc *WindowController) markLayerDirty(layer ppuLayer) {
	wc.dirtyMainWindows |= 1 << layer
	wc.dirtySubWindows |= 1 << layer
	wc.invalidationCounter = WINDOW_INVALIDATION_COUNTER
}

func (wc *WindowController) markAllWindowsDirty() {
	wc.dirtyMainWindows = 0xFF
	wc.dirtySubWindows = 0xFF
	wc.invalidationCounter = WINDOW_INVALIDATION_COUNTER
}

func setupLayerMasks(layer *LayerWindowData, value byte) {
	if value&1 != 0 {
		layer.w1Inverted = true
	} else {
		layer.w1Inverted = false
	}

	if value&2 != 0 {
		layer.w1Active = true
	} else {
		layer.w1Active = false
	}

	if value&4 != 0 {
		layer.w2Inverted = true
	} else {
		layer.w2Inverted = false
	}

	if value&8 != 0 {
		layer.w2Active = true
	} else {
		layer.w2Active = false
	}
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
	}

	//should never happen
	return wMaskXNOR
}

func (wc *WindowController) W12SEL(value byte) {
	setupLayerMasks(&wc.layers[bg1], value)
	wc.markLayerDirty(bg1)
	setupLayerMasks(&wc.layers[bg2], value>>4)
	wc.markLayerDirty(bg2)
}

func (wc *WindowController) W34SEL(value byte) {
	setupLayerMasks(&wc.layers[bg3], value)
	wc.markLayerDirty(bg3)
	setupLayerMasks(&wc.layers[bg4], value>>4)
	wc.markLayerDirty(bg4)
}

func (wc *WindowController) WOBJSEL(value byte) {
	setupLayerMasks(&wc.layers[obj], value)
	wc.markLayerDirty(obj)
	setupLayerMasks(&wc.ColorMath.colorWindowData, value>>4)
}

func (wc *WindowController) WBGLOG(value byte) {
	wc.setMaskLogic(bg1, getWMaskLogic(value))
	wc.setMaskLogic(bg2, getWMaskLogic(value>>2))
	wc.setMaskLogic(bg3, getWMaskLogic(value>>4))
	wc.setMaskLogic(bg4, getWMaskLogic(value>>6))
}

func (wc *WindowController) WOBJLOG(value byte) {
	wc.setMaskLogic(obj, getWMaskLogic(value))
	wc.ColorMath.colorWindowData.wMaskLogic = getWMaskLogic(value >> 2)
}

func (wc *WindowController) TMW(value byte) {
	wc.maskMainScreen(bg1, value&1 != 0)
	wc.maskMainScreen(bg2, value&2 != 0)
	wc.maskMainScreen(bg3, value&4 != 0)
	wc.maskMainScreen(bg4, value&8 != 0)
	wc.maskMainScreen(obj, value&0x10 != 0)
}

func (wc *WindowController) TSW(value byte) {
	wc.maskSubScreen(bg1, value&1 != 0)
	wc.maskSubScreen(bg2, value&2 != 0)
	wc.maskSubScreen(bg3, value&4 != 0)
	wc.maskSubScreen(bg4, value&8 != 0)
	wc.maskSubScreen(obj, value&0x10 != 0)
}

func (wc *WindowController) isDotInMask1Range(inverted bool, H uint16) bool {
	ret := false
	if wc.w1LeftPos > wc.w1RightPos {
		ret = false
	} else {
		if uint16(wc.w1LeftPos) < H && uint16(wc.w1RightPos) > H {
			ret = true
		}
	}

	if inverted {
		return !ret
	}

	return ret
}

func (wc *WindowController) isDotInMask2Range(inverted bool, H uint16) bool {
	ret := false
	if wc.w2LeftPos > wc.w2RightPos {
		ret = false
	} else {
		if uint16(wc.w2LeftPos) < H && uint16(wc.w2RightPos) > H {
			ret = true
		}
	}

	if inverted {
		return !ret
	}

	return ret
}

func (lwd *LayerWindowData) isDotMasked(isSubscreen bool, H uint16, wc *WindowController) bool {
	if !isSubscreen && !lwd.mainScreenMasked {
		return false
	}
	if !isSubscreen && lwd.mainScreenMasked {
		if lwd.w1Active && !lwd.w2Active {
			return wc.isDotInMask1Range(lwd.w1Inverted, H)
		}
		if !lwd.w1Active && lwd.w2Active {
			return wc.isDotInMask2Range(lwd.w2Inverted, H)
		}
		if lwd.w1Active && lwd.w2Active {
			w1 := wc.isDotInMask1Range(lwd.w1Inverted, H)
			w2 := wc.isDotInMask2Range(lwd.w2Inverted, H)
			return lwd.wMaskLogic(w1, w2)
		}
		if !lwd.w1Active && !lwd.w2Active {
			return false
		}
	}

	if isSubscreen && !lwd.subScreenMasked {
		return false
	}
	if isSubscreen && lwd.subScreenMasked {
		if lwd.w1Active && !lwd.w2Active {
			return wc.isDotInMask1Range(lwd.w1Inverted, H)
		}
		if !lwd.w1Active && lwd.w2Active {
			return wc.isDotInMask2Range(lwd.w2Inverted, H)
		}
		if lwd.w1Active && lwd.w2Active {
			w1 := wc.isDotInMask1Range(lwd.w1Inverted, H)
			w2 := wc.isDotInMask2Range(lwd.w2Inverted, H)
			return lwd.wMaskLogic(w1, w2)
		}
		if !lwd.w1Active && !lwd.w2Active {
			return false
		}
	}

	return false
}

// TODO find out if a layer is or not and skip rebuilding
func (wc *WindowController) rebuildDirtyLayerWindowCaches() {
	for i := range len(wc.layers) {
		val := byte(1 << i)
		layer := &wc.layers[i]
		if wc.dirtyMainWindows&val != 0 && wc.dirtySubWindows&val == 0 {
			for j := range uint16(SCREEN_WIDTH) {
				layer.mainCache[j] = layer.isDotMasked(false, j, wc)
			}
		} else if wc.dirtyMainWindows&val == 0 && wc.dirtySubWindows&val != 0 {
			for j := range uint16(SCREEN_WIDTH) {
				layer.subCache[j] = layer.isDotMasked(true, j, wc)
			}
		} else if wc.dirtyMainWindows&val != 0 && wc.dirtySubWindows&val != 0 {
			for j := range uint16(SCREEN_WIDTH) {
				layer.mainCache[j] = layer.isDotMasked(false, j, wc)
				layer.subCache[j] = layer.isDotMasked(true, j, wc)
			}
		}
	}
	wc.dirtyMainWindows = 0
	wc.dirtySubWindows = 0
}

func (wc *WindowController) setCGADSUB(value byte) {
	if value&0x80 != 0x80 {
		wc.ColorMath.colorFunction = addColors
	} else {
		wc.ColorMath.colorFunction = subColors
	}
	wc.ColorMath.halfColor = value&0x40 == 0x40
	//TODO remember mode 7
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
	}

	//should never happen
	return colorClipOrPreventMathAlways
}

func (wc *WindowController) isDotInColorMask(H uint16) bool {
	lwd := &wc.ColorMath.colorWindowData
	if lwd.w1Active && !lwd.w2Active {
		return wc.isDotInMask1Range(lwd.w1Inverted, H)
	}
	if !lwd.w1Active && lwd.w2Active {
		return wc.isDotInMask2Range(lwd.w2Inverted, H)
	}
	if lwd.w1Active && lwd.w2Active {
		w1 := wc.isDotInMask1Range(lwd.w1Inverted, H)
		w2 := wc.isDotInMask2Range(lwd.w2Inverted, H)
		return lwd.wMaskLogic(w1, w2)
	}
	if !lwd.w1Active && !lwd.w2Active {
		return false
	}

	return false
}

func (wc *WindowController) performColorMath(mainColor, subColor, H uint16, layer ppuLayer) uint16 {
	colorMath := &wc.ColorMath
	inColorMask := wc.isDotInColorMask(H)
	if colorMath.clipToBlack(inColorMask) {
		mainColor = 0
	}
	if colorMath.preventMath(inColorMask) {
		return mainColor
	}
	if !wc.layers[layer].colorMathActive {
		return mainColor
	}

	var blendColor uint16
	if colorMath.isSubscren {
		blendColor = subColor
	} else {
		blendColor = colorMath.fixedColor
	}

	return colorMath.colorFunction(mainColor, blendColor, colorMath.halfColor)
}
