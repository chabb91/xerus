package ppu

type bgModeSetter func(ppu *PPU, mode1Prio, isExtBg bool)
type renderPipelineGeneratorFunc func(ppu *PPU, isSubscreen bool)

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
		{bg1, 0},
		{obj, 0},
	},
	8: {
		{obj, 3},
		{obj, 2},
		{bg2, 1},
		{obj, 1},
		{bg1, 0},
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

var bgModeLUT [8]bgModeSetter

func setMode0(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp2
	ppu.Bg2.colorDepth = bpp2
	ppu.Bg3.colorDepth = bpp2
	ppu.Bg4.colorDepth = bpp2

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

	ppu.modePriority = modePriorityOrder[2]

	ppu.Bg1.optFunc = ppu.Bg1.resolveOPTMode26
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = ppu.Bg2.resolveOPTMode26
	ppu.Bg2.OPTMap = ppu.Bg3
}

func setMode3(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp8
	ppu.Bg2.colorDepth = bpp4

	ppu.modePriority = modePriorityOrder[3]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode4(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp8
	ppu.Bg2.colorDepth = bpp2

	ppu.modePriority = modePriorityOrder[4]

	ppu.Bg1.optFunc = ppu.Bg1.resolveOPTMode4
	ppu.Bg1.OPTMap = ppu.Bg3

	ppu.Bg2.optFunc = ppu.Bg2.resolveOPTMode4
	ppu.Bg2.OPTMap = ppu.Bg3
}

func setMode5(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp4
	ppu.Bg2.colorDepth = bpp2

	ppu.modePriority = modePriorityOrder[5]

	ppu.Bg1.optFunc = nil
	ppu.Bg1.OPTMap = nil

	ppu.Bg2.optFunc = nil
	ppu.Bg2.OPTMap = nil
}

func setMode6(ppu *PPU, _, _ bool) {
	ppu.Bg1.colorDepth = bpp4

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
