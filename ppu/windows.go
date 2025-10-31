package ppu

type wMaskLogic func(bool, bool) bool

type LayerWindowData struct {
	w1Active, w2Active                bool
	w1Inverted, w2Inverted            bool
	mainScreenMasked, subScreenMasked bool
	colorMathActive                   bool

	wMaskLogic wMaskLogic
}

type WindowController struct {
	w1LeftPos, w1RightPos byte
	w2LeftPos, w2RightPos byte

	//TODO color math shouldnt be bunched together with the rest.
	//it needs to be a separate struct.
	layers [6]LayerWindowData
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
	setupLayerMasks(&wc.layers[bg2], value>>4)
}

func (wc *WindowController) W34SEL(value byte) {
	setupLayerMasks(&wc.layers[bg3], value)
	setupLayerMasks(&wc.layers[bg4], value>>4)
}

func (wc *WindowController) WOBJSEL(value byte) {
	setupLayerMasks(&wc.layers[obj], value)
	setupLayerMasks(&wc.layers[colorWindow], value>>4)
}

func (wc *WindowController) WBGLOG(value byte) {
	wc.layers[bg1].wMaskLogic = getWMaskLogic(value)
	wc.layers[bg2].wMaskLogic = getWMaskLogic(value >> 2)
	wc.layers[bg3].wMaskLogic = getWMaskLogic(value >> 4)
	wc.layers[bg4].wMaskLogic = getWMaskLogic(value >> 6)
}

func (wc *WindowController) WOBJLOG(value byte) {
	wc.layers[obj].wMaskLogic = getWMaskLogic(value)
	wc.layers[colorWindow].wMaskLogic = getWMaskLogic(value >> 2)
}

func (wc *WindowController) TMW(value byte) {
	if value&1 != 0 {
		wc.layers[bg1].mainScreenMasked = true
	} else {
		wc.layers[bg1].mainScreenMasked = false
	}
	if value&2 != 0 {
		wc.layers[bg2].mainScreenMasked = true
	} else {
		wc.layers[bg2].mainScreenMasked = false
	}
	if value&4 != 0 {
		wc.layers[bg3].mainScreenMasked = true
	} else {
		wc.layers[bg3].mainScreenMasked = false
	}
	if value&8 != 0 {
		wc.layers[bg4].mainScreenMasked = true
	} else {
		wc.layers[bg4].mainScreenMasked = false
	}
	if value&0x10 != 0 {
		wc.layers[obj].mainScreenMasked = true
	} else {
		wc.layers[obj].mainScreenMasked = false
	}
}

func (wc *WindowController) TSW(value byte) {
	if value&1 != 0 {
		wc.layers[bg1].subScreenMasked = true
	} else {
		wc.layers[bg1].subScreenMasked = false
	}
	if value&2 != 0 {
		wc.layers[bg2].subScreenMasked = true
	} else {
		wc.layers[bg2].subScreenMasked = false
	}
	if value&4 != 0 {
		wc.layers[bg3].subScreenMasked = true
	} else {
		wc.layers[bg3].subScreenMasked = false
	}
	if value&8 != 0 {
		wc.layers[bg4].subScreenMasked = true
	} else {
		wc.layers[bg4].subScreenMasked = false
	}
	if value&0x10 != 0 {
		wc.layers[obj].subScreenMasked = true
	} else {
		wc.layers[obj].subScreenMasked = false
	}
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

func (wc *WindowController) isDotMasked(layer ppuLayer, isSubscreen bool, H uint16) bool {
	lwd := wc.layers[layer]
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

type ColorMath struct {
	fixedColor  uint16
	addColors   bool
	halfColor   bool
	directColor bool
	isSubscren  bool

	preventMath byte
	clipToBlack byte
}

func (cm *ColorMath) setColData(value byte) {
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
