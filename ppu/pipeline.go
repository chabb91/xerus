package ppu

const (
	renderMode0 byte = iota
	renderMode1
	renderMode1Bg3Prio
	renderMode2
	renderMode3
	renderMode4
	renderMode5
	renderMode6
	renderMode7
	renderMode7Extbg
)

type bgModeSetter func(ppu *PPU, mode1Prio bool)

type pipelineTemplate struct {
	layer    ppuLayer
	priority byte

	renderer rendererFunction

	mainScreenMask *[SCREEN_WIDTH]bool
	subScreenMask  *[SCREEN_WIDTH]bool
}

var modePriorityOrder = map[byte][]pipelineTemplate{
	renderMode0: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg3, 1, nil, nil, nil},
		{bg4, 1, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg3, 0, nil, nil, nil},
		{bg4, 0, nil, nil, nil},
	},
	renderMode1: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg3, 1, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg3, 0, nil, nil, nil},
	},
	renderMode1Bg3Prio: {
		{bg3, 1, nil, nil, nil},
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg3, 0, nil, nil, nil},
	},
	renderMode2: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
	},
	renderMode3: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
	},
	renderMode4: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
	},
	renderMode5: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
	},
	renderMode6: {
		{obj, 3, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 0, nil, nil, nil},
		{obj, 0, nil, nil, nil},
	},
	renderMode7: {
		{obj, 3, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 0, nil, nil, nil},
	},
	renderMode7Extbg: {
		{obj, 3, nil, nil, nil},
		{obj, 2, nil, nil, nil},
		{bg2, 1, nil, nil, nil},
		{obj, 1, nil, nil, nil},
		{bg1, 1, nil, nil, nil},
		{obj, 0, nil, nil, nil},
		{bg2, 0, nil, nil, nil},
	},
}

var bgModeLUT = [8]bgModeSetter{
	setMode0,
	setMode1,
	setMode2,
	setMode3,
	setMode4,
	setMode5,
	setMode6,
	setMode7,
}

func setMode0(ppu *PPU, _ bool) {
	ppu.Bg1.setBgColorDepth(bpp2)
	ppu.Bg2.setBgColorDepth(bpp2)
	ppu.Bg3.setBgColorDepth(bpp2)
	ppu.Bg4.setBgColorDepth(bpp2)

	ppu.Bg1.getPaletteIndex = mode0ColorIndex
	ppu.Bg2.getPaletteIndex = mode0ColorIndex
	ppu.Bg3.getPaletteIndex = mode0ColorIndex
	ppu.Bg4.getPaletteIndex = mode0ColorIndex

	ppu.modePriority = modePriorityOrder[renderMode0]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode1(ppu *PPU, mode1Prio bool) {
	ppu.Bg1.setBgColorDepth(bpp4)
	ppu.Bg2.setBgColorDepth(bpp4)
	ppu.Bg3.setBgColorDepth(bpp2)

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg3.getPaletteIndex = modeNormalColorNo8bppIndex

	if mode1Prio {
		ppu.modePriority = modePriorityOrder[renderMode1Bg3Prio]
	} else {
		ppu.modePriority = modePriorityOrder[renderMode1]
	}

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode2(ppu *PPU, _ bool) {
	ppu.Bg1.setBgColorDepth(bpp4)
	ppu.Bg2.setBgColorDepth(bpp4)

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[renderMode2]

	ppu.Bg1.optFunc = resolveOPTMode26
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = resolveOPTMode26
	ppu.Bg2.OPTMap = ppu.Bg3
}

func setMode3(ppu *PPU, _ bool) {
	ppu.Bg1.setBgColorDepth(bpp8)
	ppu.Bg2.setBgColorDepth(bpp4)

	ppu.Bg1.getPaletteIndex = modeNormalColor8BppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[renderMode3]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode4(ppu *PPU, _ bool) {
	ppu.Bg1.setBgColorDepth(bpp8)
	ppu.Bg2.setBgColorDepth(bpp2)

	ppu.Bg1.getPaletteIndex = modeNormalColor8BppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[renderMode4]

	ppu.Bg1.optFunc = resolveOPTMode4
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = resolveOPTMode4
	ppu.Bg2.OPTMap = ppu.Bg3
}

func setMode5(ppu *PPU, _ bool) {
	ppu.Bg1.setBgColorDepth(bpp4)
	ppu.Bg2.setBgColorDepth(bpp2)

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[renderMode5]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode6(ppu *PPU, _ bool) {
	ppu.Bg1.setBgColorDepth(bpp4)

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[renderMode6]

	ppu.Bg1.optFunc = resolveOPTMode26
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode7(ppu *PPU, _ bool) {
	if ppu.SETINI.m7EXTBG {
		ppu.modePriority = modePriorityOrder[renderMode7Extbg]
	} else {
		ppu.modePriority = modePriorityOrder[renderMode7]
	}
}

func (ppu *PPU) regeneratePipelines() {
	template := ppu.modePriority

	ppu.mainRenderPipeline = ppu.mainRenderPipeline[:0]
	ppu.subRenderPipeline = ppu.subRenderPipeline[:0]

	for _, step := range template {
		if ppu.isEnabledOnMain(step.layer) {
			step.renderer = ppu.getLayerRenderer(step.layer)
			step.mainScreenMask = &ppu.WINDOWS.layers[step.layer].mainCache
			ppu.mainRenderPipeline = append(ppu.mainRenderPipeline, step)
		}
		if ppu.isEnabledOnSub(step.layer) {
			step.renderer = ppu.getLayerRenderer(step.layer)
			step.subScreenMask = &ppu.WINDOWS.layers[step.layer].subCache
			ppu.subRenderPipeline = append(ppu.subRenderPipeline, step)
		}
	}
}

func (ppu *PPU) regenerateMainPipeline() {
	template := ppu.modePriority

	ppu.mainRenderPipeline = ppu.mainRenderPipeline[:0]

	for _, step := range template {
		if ppu.isEnabledOnMain(step.layer) {
			step.renderer = ppu.getLayerRenderer(step.layer)
			step.mainScreenMask = &ppu.WINDOWS.layers[step.layer].mainCache
			ppu.mainRenderPipeline = append(ppu.mainRenderPipeline, step)
		}
	}
}

func (ppu *PPU) regenerateSubPipeline() {
	template := ppu.modePriority

	ppu.subRenderPipeline = ppu.subRenderPipeline[:0]

	for _, step := range template {
		if ppu.isEnabledOnSub(step.layer) {
			step.renderer = ppu.getLayerRenderer(step.layer)
			step.subScreenMask = &ppu.WINDOWS.layers[step.layer].subCache
			ppu.subRenderPipeline = append(ppu.subRenderPipeline, step)
		}
	}
}

func (ppu *PPU) isEnabledOnMain(layer ppuLayer) bool {
	switch layer {
	case bg1:
		return ppu.Bg1.enabledOnMainScreen
	case bg2:
		return ppu.Bg2.enabledOnMainScreen
	case bg3:
		return ppu.Bg3.enabledOnMainScreen
	case bg4:
		return ppu.Bg4.enabledOnMainScreen
	case obj:
		return ppu.Obj.enabledOnMainScreen
	}
	return false
}

func (ppu *PPU) isEnabledOnSub(layer ppuLayer) bool {
	switch layer {
	case bg1:
		return ppu.Bg1.enabledOnSubScreen
	case bg2:
		return ppu.Bg2.enabledOnSubScreen
	case bg3:
		return ppu.Bg3.enabledOnSubScreen
	case bg4:
		return ppu.Bg4.enabledOnSubScreen
	case obj:
		return ppu.Obj.enabledOnSubScreen
	}
	return false
}

func (ppu *PPU) setBGMODE(value byte) {
	ppu.BGMODE = value & 7

	ppu.Bg1.charTileSize = (value >> 4) & 1
	ppu.Bg2.charTileSize = (value >> 5) & 1
	ppu.Bg3.charTileSize = (value >> 6) & 1
	ppu.Bg4.charTileSize = (value >> 7) & 1

	bgModeLUT[value&7](ppu, (value&8) != 0)

	ppu.regeneratePipelines()
	ppu.invalidateAllBackgrounds()
	ppu.markActiveWindowsDirty()
}

func (ppu *PPU) setTM(value byte) {
	ppu.Bg1.enabledOnMainScreen = value&1 != 0
	ppu.Bg2.enabledOnMainScreen = value&2 != 0
	ppu.Bg3.enabledOnMainScreen = value&4 != 0
	ppu.Bg4.enabledOnMainScreen = value&8 != 0
	ppu.Obj.enabledOnMainScreen = value&0x10 != 0
}
func (ppu *PPU) setTS(value byte) {
	ppu.Bg1.enabledOnSubScreen = value&1 != 0
	ppu.Bg2.enabledOnSubScreen = value&2 != 0
	ppu.Bg3.enabledOnSubScreen = value&4 != 0
	ppu.Bg4.enabledOnSubScreen = value&8 != 0
	ppu.Obj.enabledOnSubScreen = value&0x10 != 0
}

func (ppu *PPU) getLayerRenderer(layer ppuLayer) rendererFunction {
	switch layer {
	case bg1:
		if ppu.BGMODE != 7 {
			return ppu.Bg1.GetDotAt
		} else {
			return ppu.Mode7.GetDotAt
		}
	case bg2:
		if ppu.BGMODE != 7 {
			return ppu.Bg2.GetDotAt
		} else {
			return ppu.Mode7.GetDotAtEXTBG
		}
	case bg3:
		return ppu.Bg3.GetDotAt
	case bg4:
		return ppu.Bg4.GetDotAt
	case obj:
		return ppu.Obj.GetDotAt
	}

	return nil
}

func (ppu *PPU) renderMainScreen(H, V uint16) (uint16, ppuLayer, bool) {
	var val int
	var prio byte
	var math bool
	for _, v := range ppu.mainRenderPipeline {
		if v.mainScreenMask[H] {
			continue
		}

		if v.layer == obj {
			if v.priority == 3 {
				val, prio, math = v.renderer(H, V)
				colorCache[v.layer], spritePrio, spriteMath = val, prio, math
			} else {
				val, prio, math = colorCache[v.layer], spritePrio, spriteMath
			}
		} else {
			if v.priority == 1 {
				val, prio, math = v.renderer(H<<uint16(hires)+uint16(hires), V)
				colorCache[v.layer] = val
			} else {
				val, prio, math = colorCache[v.layer], 0, true
			}
		}

		if val == BG_BACKDROP_COLOR || prio != v.priority {
			continue
		}
		return uint16(val), v.layer, math
	}
	return ppu.CGRAM.CGRAM[0], backdrop, true
}

func (ppu *PPU) renderSubScreen(H, V uint16) (uint16, ppuLayer, bool) {
	var val int
	var prio byte
	var math bool
	for _, v := range ppu.subRenderPipeline {
		if v.subScreenMask[H] {
			continue
		}

		if v.layer == obj {
			if v.priority == 3 {
				val, prio, math = v.renderer(H, V)
				colorCache[v.layer], spritePrio, spriteMath = val, prio, math
			} else {
				val, prio, math = colorCache[v.layer], spritePrio, spriteMath
			}
		} else {
			if v.priority == 1 {
				val, prio, math = v.renderer(H<<uint16(hires), V)
				colorCache[v.layer] = val
			} else {
				val, prio, math = colorCache[v.layer], 0, true
			}
		}

		if val == BG_BACKDROP_COLOR || prio != v.priority {
			continue
		}
		return uint16(val), v.layer, math
	}
	return ppu.CGRAM.CGRAM[0], backdrop, true
}
