package ppu

type bgModeSetter func(ppu *PPU, mode1Prio, isExtBg bool)

type pipelineTemplate struct {
	layer    ppuLayer
	priority byte
}

var modePriorityOrder = map[byte][]pipelineTemplate{
	0: {
		{obj, 3},
		{bg1, 1},
		{bg2, 1},
		{obj, 2},
		{bg1, 0},
		{bg2, 0},
		{obj, 1},
		{bg3, 1},
		{bg4, 1},
		{obj, 0},
		{bg3, 0},
		{bg4, 0},
	},
	1: {
		{obj, 3},
		{bg1, 1},
		{bg2, 1},
		{obj, 2},
		{bg1, 0},
		{bg2, 0},
		{obj, 1},
		{bg3, 1},
		{obj, 0},
		{bg3, 0},
	},
	2: {
		{obj, 3},
		{bg1, 1},
		{obj, 2},
		{bg2, 1},
		{obj, 1},
		{bg1, 0},
		{obj, 0},
		{bg2, 0},
	},
	3: {
		{obj, 3},
		{bg1, 1},
		{obj, 2},
		{bg2, 1},
		{obj, 1},
		{bg1, 0},
		{obj, 0},
		{bg2, 0},
	},
	4: {
		{obj, 3},
		{bg1, 1},
		{obj, 2},
		{bg2, 1},
		{obj, 1},
		{bg1, 0},
		{obj, 0},
		{bg2, 0},
	},
	5: {
		{obj, 3},
		{bg1, 1},
		{obj, 2},
		{bg2, 1},
		{obj, 1},
		{bg1, 0},
		{obj, 0},
		{bg2, 0},
	},
	6: {
		{obj, 3},
		{bg1, 1},
		{obj, 2},
		{obj, 1},
		{bg1, 0},
		{obj, 0},
	},
	7: {
		{obj, 3},
		{obj, 2},
		{obj, 1},
		{bgMode7, 0},
		{obj, 0},
	},
	8: {
		{obj, 3},
		{obj, 2},
		{bg2, 1}, //TODO i believe bg2 is bg1 and bg1 is mode7
		{obj, 1},
		{bgMode7, 0},
		{obj, 0},
		{bg2, 0},
	},
	9: {
		{bg3, 1},
		{obj, 3},
		{bg1, 1},
		{bg2, 1},
		{obj, 2},
		{bg1, 0},
		{bg2, 0},
		{obj, 1},
		{obj, 0},
		{bg3, 0},
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

func setMode0(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp2
	ppu.Bg2.colorDepth = bpp2
	ppu.Bg3.colorDepth = bpp2
	ppu.Bg4.colorDepth = bpp2

	ppu.Bg1.getPaletteIndex = mode0ColorIndex
	ppu.Bg2.getPaletteIndex = mode0ColorIndex
	ppu.Bg3.getPaletteIndex = mode0ColorIndex
	ppu.Bg4.getPaletteIndex = mode0ColorIndex

	ppu.modePriority = modePriorityOrder[0]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode1(ppu *PPU, mode1Prio, _ bool) {
	ppu.Bg1.colorDepth = bpp4
	ppu.Bg2.colorDepth = bpp4
	ppu.Bg3.colorDepth = bpp2

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg3.getPaletteIndex = modeNormalColorNo8bppIndex

	if mode1Prio {
		ppu.modePriority = modePriorityOrder[9]
	} else {
		ppu.modePriority = modePriorityOrder[1]
	}

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode2(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp4
	ppu.Bg2.colorDepth = bpp4

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[2]

	ppu.Bg1.optFunc = ppu.Bg1.resolveOPTMode26
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = ppu.Bg2.resolveOPTMode26
	ppu.Bg2.OPTMap = ppu.Bg3
}

func setMode3(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp8
	ppu.Bg2.colorDepth = bpp4

	ppu.Bg1.getPaletteIndex = modeNormalColor8BppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[3]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode4(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp8
	ppu.Bg2.colorDepth = bpp2

	ppu.Bg1.getPaletteIndex = modeNormalColor8BppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[4]

	ppu.Bg1.optFunc = ppu.Bg1.resolveOPTMode4
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = ppu.Bg2.resolveOPTMode4
	ppu.Bg2.OPTMap = ppu.Bg3
}

func setMode5(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp4
	ppu.Bg2.colorDepth = bpp2

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[5]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode6(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp4

	ppu.Bg1.getPaletteIndex = modeNormalColorNo8bppIndex

	ppu.modePriority = modePriorityOrder[6]

	ppu.Bg1.optFunc = ppu.Bg1.resolveOPTMode26
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode7(ppu *PPU, _, isExtBg bool) {
	//TODO remember mode 7 this doesnt work yet
	ppu.Bg1.colorDepth = bpp8
	ppu.Bg2.colorDepth = bpp2

	ppu.Bg1.getPaletteIndex = modeNormalColor8BppIndex
	ppu.Bg2.getPaletteIndex = modeNormalColorNo8bppIndex

	if isExtBg {
		ppu.modePriority = modePriorityOrder[7]
	} else {
		ppu.modePriority = modePriorityOrder[8]
	}

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func (ppu *PPU) regeneratePipelines() {
	template := ppu.modePriority

	ppu.mainRenderPipeline = ppu.mainRenderPipeline[:0]
	ppu.subRenderPipeline = ppu.subRenderPipeline[:0]

	for _, step := range template {
		if ppu.isEnabledOnMain(step.layer) {
			ppu.mainRenderPipeline = append(ppu.mainRenderPipeline, step)
		}
		if ppu.isEnabledOnSub(step.layer) {
			ppu.subRenderPipeline = append(ppu.subRenderPipeline, step)
		}
	}
}

func (ppu *PPU) regenerateMainPipeline() {
	template := ppu.modePriority

	ppu.mainRenderPipeline = ppu.mainRenderPipeline[:0]

	for _, step := range template {
		if ppu.isEnabledOnMain(step.layer) {
			ppu.mainRenderPipeline = append(ppu.mainRenderPipeline, step)
		}
	}
}

func (ppu *PPU) regenerateSubPipeline() {
	template := ppu.modePriority

	ppu.subRenderPipeline = ppu.subRenderPipeline[:0]

	for _, step := range template {
		if ppu.isEnabledOnSub(step.layer) {
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

	bgModeLUT[value&7](ppu, (value&8) != 0, ppu.SETINI.m7EXTBG)

	ppu.regeneratePipelines()
}

func (ppu *PPU) setTM(value byte) {
	if value&1 != 0 {
		ppu.Bg1.enabledOnMainScreen = true
	} else {
		ppu.Bg1.enabledOnMainScreen = false
	}
	if value&2 != 0 {
		ppu.Bg2.enabledOnMainScreen = true
	} else {
		ppu.Bg2.enabledOnMainScreen = false
	}

	if value&4 != 0 {
		ppu.Bg3.enabledOnMainScreen = true
	} else {
		ppu.Bg3.enabledOnMainScreen = false
	}

	if value&8 != 0 {
		ppu.Bg4.enabledOnMainScreen = true
	} else {
		ppu.Bg4.enabledOnMainScreen = false
	}

	if value&0x10 != 0 {
		ppu.Obj.enabledOnMainScreen = true
	} else {
		ppu.Obj.enabledOnMainScreen = false
	}
}
func (ppu *PPU) setTS(value byte) {
	if value&1 != 0 {
		ppu.Bg1.enabledOnSubScreen = true
	} else {
		ppu.Bg1.enabledOnSubScreen = false
	}
	if value&2 != 0 {
		ppu.Bg2.enabledOnSubScreen = true
	} else {
		ppu.Bg2.enabledOnSubScreen = false
	}

	if value&4 != 0 {
		ppu.Bg3.enabledOnSubScreen = true
	} else {
		ppu.Bg3.enabledOnSubScreen = false
	}

	if value&8 != 0 {
		ppu.Bg4.enabledOnSubScreen = true
	} else {
		ppu.Bg4.enabledOnSubScreen = false
	}

	if value&0x10 != 0 {
		ppu.Obj.enabledOnSubScreen = true
	} else {
		ppu.Obj.enabledOnSubScreen = false
	}
}

func (ppu *PPU) getLayerDot(layer ppuLayer, H, V uint16) uint16 {
	switch layer {
	case bg1:
		return ppu.Bg1.GetDotAt(H, V)
	case bg2:
		return ppu.Bg2.GetDotAt(H, V)
	case bg3:
		return ppu.Bg3.GetDotAt(H, V)
	case bg4:
		return ppu.Bg4.GetDotAt(H, V)
	case obj:
		return ppu.Obj.resolvedDotsOnScanLine[H].color
	}
	return 0
}

func (ppu *PPU) renderMainScreen(H, V uint16) (uint16, ppuLayer) {
	var val uint16
	for _, v := range ppu.mainRenderPipeline {
		if ppu.WINDOWS.isDotMasked(v.layer, false, H) {
			continue
		}
		val = ppu.getLayerDot(v.layer, H, V)
		if val == BG_BACKDROP_COLOR {
			continue
		}
		return val, v.layer
	}
	return ppu.CGRAM.CGRAM[BG_BACKDROP_COLOR], backdrop
}

func (ppu *PPU) renderSubScreen(H, V uint16) (uint16, ppuLayer) {
	var val uint16
	for _, v := range ppu.subRenderPipeline {
		if ppu.WINDOWS.isDotMasked(v.layer, false, H) {
			continue
		}
		val = ppu.getLayerDot(v.layer, H, V)
		if val == BG_BACKDROP_COLOR {
			continue
		}
		return val, v.layer
	}
	return ppu.CGRAM.CGRAM[BG_BACKDROP_COLOR], backdrop
}
