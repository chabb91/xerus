package ppu

import (
	"SNES_emulator/ui"
	"fmt"
)

type tileDataSource interface {
	getOAMLow() []byte
	getOAMHigh() []byte
	getVRAM() []uint16
	getCGRAM() []uint16
	getPriorityRotation() byte
}

type tileValidator interface {
	tryInvalidate(addr uint16)
	invalidateLayer(layerIndex ppuLayer)
	invalidateAllLayers()
}

type spriteValidator interface {
	invalidateSpriteLo(id uint16)
	invalidateSpriteHi(id uint16)
}

type PPU struct {
	SETINI *SETINI

	OAM   *OAMController
	VRAM  *VRAMController
	CGRAM *CGRAMController

	WINDOWS WindowController

	BGMODE byte

	Bg1     *Background
	Bg2     *Background
	Bg3     *Background
	Bg4     *Background
	BGxnOFS *BGxnOFS

	Obj *Objects

	FBlank, VBlank, HBlank bool
	brightness             byte

	H, V int

	modePriority       []pipelineTemplate
	mainRenderPipeline []pipelineTemplate
	subRenderPipeline  []pipelineTemplate

	bgEpochs [6]*uint64 //1 2 3 4 mode7 and obj

	InterruptScheduler InterruptScheduler
	HdmaScheduler      HdmaScheduler

	Framebuffer *ui.Framebuffer
}

func NewPPU() *PPU {
	ppu := &PPU{
		CGRAM:   NewCGRAM(),
		BGxnOFS: &BGxnOFS{},
		SETINI:  NewSETINI(PAL_TIMING),
	}
	ppu.mainRenderPipeline = make([]pipelineTemplate, 0, 12)
	ppu.subRenderPipeline = make([]pipelineTemplate, 0, 12)

	ppu.Bg1 = NewBackground(ppu, bg1)
	ppu.Bg2 = NewBackground(ppu, bg2)
	ppu.Bg3 = NewBackground(ppu, bg3)
	ppu.Bg4 = NewBackground(ppu, bg4)
	ppu.Obj = newObjects(ppu, obj)

	ppu.bgEpochs[bg1] = &ppu.Bg1.currentEpoch
	ppu.bgEpochs[bg2] = &ppu.Bg2.currentEpoch
	ppu.bgEpochs[bg3] = &ppu.Bg3.currentEpoch
	ppu.bgEpochs[bg4] = &ppu.Bg4.currentEpoch
	ppu.bgEpochs[obj] = &ppu.Obj.currentEpoch
	//TODO placeholder to avoid nil
	ppu.bgEpochs[bgMode7] = new(uint64)

	ppu.VRAM = NewVRAM(ppu)
	ppu.OAM = NewOAM(ppu)

	return ppu
}

// Some of these registers can only be read and written to at specific times defined by the blanking periods
// TODO
func (ppu *PPU) Read(addr uint16) (byte, error) {
	switch addr {
	case 0x2138:
		return ppu.OAM.ReadOAMData(), nil
	case 0x2139:
		return ppu.VRAM.ReadDataLow(), nil
	case 0x213A:
		return ppu.VRAM.ReadDataHigh(), nil
	case 0x213B:
		return ppu.CGRAM.ReadData(), nil
	default:
		return 0, fmt.Errorf("invalid PPU register read at $%04X", addr)
	}
}

// TODO some of these heavy register operations should be deferred to the next scanline for accuracy
// its called mode latch delay
// bgmode and mosaic for sure belong in this category
func (ppu *PPU) Write(addr uint16, value byte) error {
	switch addr {
	case 0x2100:
		tempFBlank := ppu.FBlank

		ppu.FBlank = (value>>7)&1 == 1
		ppu.brightness = value & 0xF

		if !tempFBlank && ppu.FBlank {
			ppu.OAM.InvalidateInternalIndex()
		}
	case 0x2101:
		ppu.Obj.setupOBSEL(value)
		ppu.invalidateLayer(obj)
	case 0x2102:
		ppu.OAM.SetAddWordLow(value)
	case 0x2103:
		ppu.OAM.SetAddWordHigh(value)
	case 0x2104:
		ppu.OAM.WriteOAMData(value)
	case 0x2105:
		//fmt.Println("BGMODE: ", value)
		ppu.setBGMODE(value)
	case 0x2106:
		fmt.Println("MOSAIC: ", value)
		ppu.Bg1.mosaic = value&1 == 1
		ppu.Bg2.mosaic = value&2 == 2
		ppu.Bg3.mosaic = value&4 == 4
		ppu.Bg4.mosaic = value&8 == 8

		ms := value>>4 + 1
		if ms != mosaicSize {
			if ppu.V > 0 && ppu.SETINI.Timing.TotalScanlines >= ppu.V {
				mosaicStartLine = uint16(ppu.V)
			} else {
				mosaicStartLine = 0
			}
		}
		mosaicSize = ms
		hasMosaic = value&0xF > 0
	case 0x2107:
		fmt.Println("BG1SC: ", value)
		ppu.Bg1.tileMapSize = uint16(value & 0x3)
		ppu.Bg1.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg1)
	case 0x2108:
		fmt.Println("BG2SC: ", value)
		ppu.Bg2.tileMapSize = uint16(value & 0x3)
		ppu.Bg2.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg2)
	case 0x2109:
		fmt.Println("BG3SC: ", value)
		ppu.Bg3.tileMapSize = uint16(value & 0x3)
		ppu.Bg3.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg3)
	case 0x210A:
		fmt.Println("BG4SC: ", value)
		ppu.Bg4.tileMapSize = uint16(value & 0x3)
		ppu.Bg4.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg4)
	case 0x210B:
		fmt.Println("BG12NBA: ", value)
		ppu.Bg1.charTileAddressBase = (uint16(value&0xF) << 12) & 0x7FFF
		ppu.Bg2.charTileAddressBase = (uint16((value>>4)&0xF) << 12) & 0x7FFF
		ppu.invalidateLayer(bg1)
		ppu.invalidateLayer(bg2)
	case 0x210C:
		fmt.Println("BG34NBA: ", value)
		ppu.Bg3.charTileAddressBase = (uint16(value&0xF) << 12) & 0x7FFF
		ppu.Bg4.charTileAddressBase = (uint16((value>>4)&0xF) << 12) & 0x7FFF
		ppu.invalidateLayer(bg3)
		ppu.invalidateLayer(bg4)
	//TODO add mode 7 scrolling
	case 0x210D:
		ppu.Bg1.hScroll = ppu.BGxnOFS.hFormula(value)
	case 0x210E:
		ppu.Bg1.vScroll = ppu.BGxnOFS.vFormula(value)
	case 0x210F:
		ppu.Bg2.hScroll = ppu.BGxnOFS.hFormula(value)
	case 0x2110:
		ppu.Bg2.vScroll = ppu.BGxnOFS.vFormula(value)
	case 0x2111:
		ppu.Bg3.hScroll = ppu.BGxnOFS.hFormula(value)
	case 0x2112:
		ppu.Bg3.vScroll = ppu.BGxnOFS.vFormula(value)
	case 0x2113:
		ppu.Bg4.hScroll = ppu.BGxnOFS.hFormula(value)
	case 0x2114:
		ppu.Bg4.vScroll = ppu.BGxnOFS.vFormula(value)
	case 0x2115:
		ppu.VRAM.vmain.Setup(value)
		fmt.Println("VMAIN: ", value)
	case 0x2116:
		ppu.VRAM.UpdateAddressLow(value)
		fmt.Println("VMADDLOW: ", value)
	case 0x2117:
		ppu.VRAM.UpdateAddressHigh(value)
		fmt.Println("VMADDHIGH: ", value)
	case 0x2118:
		ppu.VRAM.WriteDataLow(value)
	case 0x2119:
		ppu.VRAM.WriteDataHigh(value)
	case 0x2121:
		ppu.CGRAM.SetAddWord(value)
	case 0x2122:
		ppu.CGRAM.WriteData(value)
	case 0x2123:
		ppu.WINDOWS.W12SEL(value)
	case 0x2124:
		ppu.WINDOWS.W34SEL(value)
	case 0x2125:
		ppu.WINDOWS.WOBJSEL(value)
	case 0x2126:
		ppu.WINDOWS.w1LeftPos = value
		ppu.markActiveWindowsDirty()
	case 0x2127:
		ppu.WINDOWS.w1RightPos = value
		ppu.markActiveWindowsDirty()
	case 0x2128:
		ppu.WINDOWS.w2LeftPos = value
		ppu.markActiveWindowsDirty()
	case 0x2129:
		ppu.WINDOWS.w2RightPos = value
		ppu.markActiveWindowsDirty()
	case 0x212A:
		ppu.WINDOWS.WBGLOG(value)
	case 0x212B:
		ppu.WINDOWS.WOBJLOG(value)
	case 0x212C:
		fmt.Println("TM: ", value)
		ppu.setTM(value)
		ppu.regenerateMainPipeline()
		ppu.invalidateAllLayers()
		ppu.markActiveWindowsDirty()
	case 0x212D:
		fmt.Println("TS: ", value)
		ppu.setTS(value)
		ppu.regenerateSubPipeline()
		ppu.invalidateAllLayers()
		ppu.markActiveWindowsDirty()
	case 0x212E:
		ppu.WINDOWS.TMW(value)
	case 0x212F:
		ppu.WINDOWS.TSW(value)
	case 0x2130:
		fmt.Println("CGWSEL", value)
		//TODO remember mode7
		ppu.WINDOWS.ColorMath.setCGWSEL(value, &ppu.Bg1.isDirectColor)
	case 0x2131:
		fmt.Println("CGADSUB", value)
		ppu.WINDOWS.setCGADSUB(value)
	case 0x2132:
		fmt.Println("COLDATA", value)
		ppu.WINDOWS.ColorMath.setCOLDATA(value)
	case 0x2133:
		fmt.Println("SETINI", value)
		ppu.SETINI.setup(value)
		ppu.Framebuffer.CurrentHeight, ppu.Framebuffer.CurrentWidth = ppu.SETINI.getScreenHeight(), ppu.SETINI.getScreenWidth()
	default:
		return fmt.Errorf("invalid PPU register write at $%04X", addr)
	}
	return nil
}

func (ppu *PPU) getOAMLow() []byte {
	return ppu.OAM.LowTable
}

func (ppu *PPU) getOAMHigh() []byte {
	return ppu.OAM.HighTable
}

func (ppu *PPU) getVRAM() []uint16 {
	return ppu.VRAM.VRAM
}

func (ppu *PPU) getCGRAM() []uint16 {
	return ppu.CGRAM.CGRAM
}

func (ppu *PPU) getPriorityRotation() byte {
	return ppu.OAM.GetSpritePriority()
}

func (ppu *PPU) tryInvalidate(addr uint16) {
	//only check locally if the layer is enabled
	if ppu.Bg1.isActive() {
		ppu.Bg1.Invalidate(addr)
	}
	if ppu.Bg2.isActive() {
		ppu.Bg2.Invalidate(addr)
	}
	if ppu.Bg3.isActive() {
		ppu.Bg3.Invalidate(addr)
	}
	if ppu.Bg4.isActive() {
		ppu.Bg4.Invalidate(addr)
	}
	if ppu.Obj.isActive() {
		ppu.Obj.Invalidate(addr)
	}
}

func (ppu *PPU) markActiveWindowsDirty() {
	if ppu.Bg1.isActive() {
		ppu.WINDOWS.markLayerDirty(bg1)
	}
	if ppu.Bg2.isActive() {
		ppu.WINDOWS.markLayerDirty(bg2)
	}
	if ppu.Bg3.isActive() {
		ppu.WINDOWS.markLayerDirty(bg3)
	}
	if ppu.Bg4.isActive() {
		ppu.WINDOWS.markLayerDirty(bg4)
	}
	if ppu.Obj.isActive() {
		ppu.WINDOWS.markLayerDirty(obj)
	}

	ppu.WINDOWS.ColorMath.windowValid = false
	ppu.WINDOWS.invalidationCounter = WINDOW_INVALIDATION_COUNTER
}

func (ppu *PPU) invalidateLayer(layerIndex ppuLayer) {
	if layerIndex >= 0 && layerIndex < ppuLayer(len(ppu.bgEpochs)) {
		*ppu.bgEpochs[layerIndex]++
	}
}

func (ppu *PPU) invalidateAllLayers() {
	for i := range ppu.bgEpochs {
		*ppu.bgEpochs[i]++
	}
}

func (ppu *PPU) invalidateAllBackgrounds() {
	*ppu.bgEpochs[bg1]++
	*ppu.bgEpochs[bg2]++
	*ppu.bgEpochs[bg3]++
	*ppu.bgEpochs[bg4]++
}

// sprites are only being invalidated locally because if a rom doesnt enable them oam is not interacted with
func (ppu *PPU) invalidateSpriteLo(id uint16) {
	ppu.Obj.Sprites[(id>>2)&127].isValid = false
}

func (ppu *PPU) invalidateSpriteHi(id uint16) {
	id = (id & 31) << 2
	for i := range uint16(4) {
		ppu.Obj.Sprites[id+i].isValid = false
	}
}
