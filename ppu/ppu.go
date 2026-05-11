package ppu

import (
	"fmt"

	"github.com/chabb91/xerus/memory"
	"github.com/chabb91/xerus/ui"
)

const CHIP_5C77_VERSION = byte(1)
const CHIP_5C78_VERSION = byte(3)

type tileDataSource interface {
	getOAMLow() []byte
	getOAMHigh() []byte
	getVRAM() []uint16
	getCGRAM() []uint16
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
	externalSync bool //not used
	m7EXTBG      bool

	Timing VideoTiming

	ScreenHeight int //overscan or not

	OAM   *OAMController
	VRAM  *VRAMController
	CGRAM *CGRAMController

	WINDOWS WindowController

	Bg1     *Background
	Bg2     *Background
	Bg3     *Background
	Bg4     *Background
	BGxnOFS BGxnOFS

	Mode7 *Mode7
	M7x   M7Registers

	Obj *Objects

	BGMODE                 byte
	registerPreviousValues [64]uint16

	FBlank, VBlank, HBlank bool
	brightness             byte

	H, V int

	HLatch, VLatch int
	HHigh, VHigh   bool
	Wrio           *byte //maintained elsewhere but the ppu needs access to this
	LatchFlag      byte  //1 == latched, 0 == unlatched

	modePriority       []pipelineTemplate
	mainRenderPipeline []pipelineTemplate
	subRenderPipeline  []pipelineTemplate

	bgEpochs [5]*uint64 //1 2 3 4 and obj

	InterruptScheduler InterruptScheduler
	HdmaScheduler      HdmaScheduler

	IrqFunc        func() bool
	IrqTimeUpTimer byte

	Refresh bool

	ppu1OB, ppu2OB byte
	//required for like 1 thing only. unlucky
	bus memory.Bus

	HTotal int

	Framebuffer *ui.Framebuffer
}

func NewPPU(bus memory.Bus, isPal bool) *PPU {
	ppu := &PPU{
		bus:    bus,
		HTotal: H_TOTAL,
	}

	if isPal {
		ppu.Timing = PAL_TIMING
	} else {
		ppu.Timing = NTSC_TIMING
	}

	//these 3 need to be initialized first so the DI works later
	ppu.VRAM = NewVRAM(ppu)
	ppu.OAM = NewOAM(ppu)
	ppu.CGRAM = NewCGRAM(&ppu.ppu2OB)

	ppu.mainRenderPipeline = make([]pipelineTemplate, 0, 12)
	ppu.subRenderPipeline = make([]pipelineTemplate, 0, 12)

	ppu.Bg1 = NewBackground(ppu, bg1)
	ppu.Bg2 = NewBackground(ppu, bg2)
	ppu.Bg3 = NewBackground(ppu, bg3)
	ppu.Bg4 = NewBackground(ppu, bg4)
	ppu.Obj = newObjects(ppu, ppu.OAM.GetSpritePriority, obj)

	ppu.Mode7 = newMode7(ppu, ppu.Bg1, ppu.Bg2)

	ppu.bgEpochs[bg1] = &ppu.Bg1.currentEpoch
	ppu.bgEpochs[bg2] = &ppu.Bg2.currentEpoch
	ppu.bgEpochs[bg3] = &ppu.Bg3.currentEpoch
	ppu.bgEpochs[bg4] = &ppu.Bg4.currentEpoch
	ppu.bgEpochs[obj] = &ppu.Obj.currentEpoch

	bus.RegisterRange(0x2100, 0x213F, ppu, "PPU")
	return ppu
}

// initializing the ppu to a known state at start
func (ppu *PPU) Init() {
	for i := range uint16(64) {
		ppu.registerPreviousValues[i] = 0xFFFF
		ppu.Write(0x2100|i, 0)
	}
}

// Some of these registers can only be read and written to at specific times defined by the blanking periods
// TODO
func (ppu *PPU) Read(addr uint16) (byte, error) {
	switch addr {
	//TODO these 3 can only be read in fblank/vblank
	//no idea if this is correct or not tbh
	case 0x2134:
		result := int32(ppu.Mode7.m7A) * int32(int8(ppu.Mode7.m7B>>8))
		//fmt.Println("READING MUL LO ", byte(result))
		return ppu.returnAndSetPpu1OB(byte(result)), nil
	case 0x2135:
		result := int32(ppu.Mode7.m7A) * int32(int8(ppu.Mode7.m7B>>8))
		//fmt.Println("READING MUL MID ", byte(result>>8))
		return ppu.returnAndSetPpu1OB(byte(result >> 8)), nil
	case 0x2136:
		result := int32(ppu.Mode7.m7A) * int32(int8(ppu.Mode7.m7B>>8))
		//fmt.Println("READING MUL HI ", byte(result>>16))
		return ppu.returnAndSetPpu1OB(byte(result >> 16)), nil
	case 0x2137:
		ppu.LatchHV()
		return ppu.bus.GetOpenBus(), nil
	case 0x2138:
		return ppu.returnAndSetPpu1OB(ppu.OAM.ReadOAMData()), nil
	case 0x2139:
		return ppu.returnAndSetPpu1OB(ppu.VRAM.ReadDataLow()), nil
	case 0x213A:
		return ppu.returnAndSetPpu1OB(ppu.VRAM.ReadDataHigh()), nil
	case 0x213B:
		return ppu.returnAndSetPpu2OB(ppu.CGRAM.ReadData()), nil
	case 0x213C:
		var ret byte
		if ppu.HHigh {
			ret = byte(ppu.HLatch>>8)&1 | ppu.ppu2OB&0xFE
		} else {
			ret = byte(ppu.HLatch)
		}
		ppu.HHigh = !ppu.HHigh
		return ppu.returnAndSetPpu2OB(ret), nil
	case 0x213D:
		var ret byte
		if ppu.VHigh {
			ret = byte(ppu.VLatch>>8)&1 | ppu.ppu2OB&0xFE
		} else {
			ret = byte(ppu.VLatch)
		}
		ppu.VHigh = !ppu.VHigh
		return ppu.returnAndSetPpu2OB(ret), nil
	case 0x213E:
		//bit 5 is some master/slave mode thing but seems to always be 0
		return ppu.returnAndSetPpu1OB(ppu.Obj.timeOver | ppu.Obj.rangeOver | ppu.ppu1OB&0x10 | CHIP_5C77_VERSION), nil
	case 0x213F:
		ppu.VHigh, ppu.HHigh = false, false
		ret := byte(interlaceStep&1)<<7 | ppu.ppu2OB&0x20 | ppu.Timing.RegionId<<4 | CHIP_5C78_VERSION
		if *ppu.Wrio >= 0x80 {
			ret |= ppu.LatchFlag << 6
			ppu.LatchFlag = 0
		} else {
			ret |= 1 << 6
		}
		return ppu.returnAndSetPpu2OB(ret), nil
	default:
		if ppu.isPpu1WriteRegisterRead(addr) {
			return ppu.ppu1OB, nil
		} else {
			return 0, fmt.Errorf("invalid PPU register read at $%04X", addr)
		}
	}
}

// TODO some of these heavy register operations should be deferred to the next scanline for accuracy
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
		if ppu.registerPreviousValues[5] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[5] = uint16(value)
		ppu.setBGMODE(value)
		ppu.setHiresFlag()
	case 0x2106:
		//fmt.Println("MOSAIC: ", value)
		ppu.Bg1.mosaic = value&1 == 1
		ppu.Bg2.mosaic = value&2 == 2
		ppu.Bg3.mosaic = value&4 == 4
		ppu.Bg4.mosaic = value&8 == 8

		ms := value>>4 + 1
		if ms != mosaicSize {
			if ppu.V > 0 && ppu.ScreenHeight >= ppu.V {
				mosaicStartLine = uint16(ppu.V)
			} else {
				mosaicStartLine = 0
			}
		}
		mosaicSize = ms
		hasMosaic = value&0xF > 0
	case 0x2107:
		//fmt.Println("BG1SC: ", value)
		if ppu.registerPreviousValues[0x07] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x07] = uint16(value)
		ppu.Bg1.tileMapSize = uint16(value & 0x3)
		ppu.Bg1.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg1)
	case 0x2108:
		//fmt.Println("BG2SC: ", value)
		if ppu.registerPreviousValues[0x08] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x08] = uint16(value)
		ppu.Bg2.tileMapSize = uint16(value & 0x3)
		ppu.Bg2.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg2)
	case 0x2109:
		//fmt.Println("BG3SC: ", value)
		if ppu.registerPreviousValues[0x09] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x09] = uint16(value)
		ppu.Bg3.tileMapSize = uint16(value & 0x3)
		ppu.Bg3.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg3)
	case 0x210A:
		//fmt.Println("BG4SC: ", value)
		if ppu.registerPreviousValues[0x0A] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x0A] = uint16(value)
		ppu.Bg4.tileMapSize = uint16(value & 0x3)
		ppu.Bg4.tileMapAddress = (uint16((value>>2)&0x3F) << 10) & 0x7FFF
		ppu.invalidateLayer(bg4)
	case 0x210B:
		//fmt.Println("BG12NBA: ", value)
		if ppu.registerPreviousValues[0x0B] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x0B] = uint16(value)
		ppu.Bg1.charTileAddressBase = (uint16(value&0xF) << 12) & 0x7FFF
		ppu.Bg2.charTileAddressBase = (uint16((value>>4)&0xF) << 12) & 0x7FFF
		ppu.invalidateLayer(bg1)
		ppu.invalidateLayer(bg2)
	case 0x210C:
		//fmt.Println("BG34NBA: ", value)
		if ppu.registerPreviousValues[0x0C] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x0C] = uint16(value)
		ppu.Bg3.charTileAddressBase = (uint16(value&0xF) << 12) & 0x7FFF
		ppu.Bg4.charTileAddressBase = (uint16((value>>4)&0xF) << 12) & 0x7FFF
		ppu.invalidateLayer(bg3)
		ppu.invalidateLayer(bg4)
	case 0x210D:
		ppu.Bg1.hScroll = ppu.BGxnOFS.hFormula(value)
		ppu.Mode7.hScroll = signExtend13(ppu.M7x.setRegister(value))
	case 0x210E:
		ppu.Bg1.vScroll = ppu.BGxnOFS.vFormula(value)
		ppu.Mode7.vScroll = signExtend13(ppu.M7x.setRegister(value))
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
		ppu.VRAM.setupVMAIN(value)
		//fmt.Println("VMAIN: ", value)
	case 0x2116:
		ppu.VRAM.UpdateAddressLow(value)
		//fmt.Println("VMADDLOW: ", value)
	case 0x2117:
		ppu.VRAM.UpdateAddressHigh(value)
		//fmt.Println("VMADDHIGH: ", value)
	case 0x2118:
		ppu.VRAM.WriteDataLow(value)
	case 0x2119:
		ppu.VRAM.WriteDataHigh(value)
	case 0x211A:
		//fmt.Println("M7SEL: ", value)
		ppu.Mode7.setM7Sel(value)
	case 0x211B:
		//fmt.Println("WRITING A ", value)
		ppu.Mode7.m7A = int16(ppu.M7x.setRegister(value))
	case 0x211C:
		//fmt.Println("WRITING B ", value)
		ppu.Mode7.m7B = int16(ppu.M7x.setRegister(value))
	case 0x211D:
		//fmt.Println("WRITING C ", value)
		ppu.Mode7.m7C = int16(ppu.M7x.setRegister(value))
	case 0x211E:
		//fmt.Println("WRITING D ", value)
		ppu.Mode7.m7D = int16(ppu.M7x.setRegister(value))
	case 0x211F:
		//fmt.Println("WRITING X ", value)
		ppu.Mode7.m7X = signExtend13(ppu.M7x.setRegister(value))
	case 0x2120:
		//fmt.Println("WRITING Y ", value)
		ppu.Mode7.m7Y = signExtend13(ppu.M7x.setRegister(value))
	case 0x2121:
		ppu.CGRAM.SetAddWord(value)
	case 0x2122:
		ppu.CGRAM.WriteData(value)
	case 0x2123:
		if ppu.registerPreviousValues[0x23] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x23] = uint16(value)
		ppu.WINDOWS.W12SEL(value)
	case 0x2124:
		if ppu.registerPreviousValues[0x24] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x24] = uint16(value)
		ppu.WINDOWS.W34SEL(value)
	case 0x2125:
		if ppu.registerPreviousValues[0x25] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x25] = uint16(value)
		ppu.WINDOWS.WOBJSEL(value)
	case 0x2126:
		if ppu.registerPreviousValues[0x26] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x26] = uint16(value)
		ppu.WINDOWS.w1LeftPos = value
		ppu.markActiveWindowsDirty()
	case 0x2127:
		if ppu.registerPreviousValues[0x27] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x27] = uint16(value)
		ppu.WINDOWS.w1RightPos = value
		ppu.markActiveWindowsDirty()
	case 0x2128:
		if ppu.registerPreviousValues[0x28] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x28] = uint16(value)
		ppu.WINDOWS.w2LeftPos = value
		ppu.markActiveWindowsDirty()
	case 0x2129:
		if ppu.registerPreviousValues[0x29] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x29] = uint16(value)
		ppu.WINDOWS.w2RightPos = value
		ppu.markActiveWindowsDirty()
	case 0x212A:
		if ppu.registerPreviousValues[0x2A] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x2A] = uint16(value)
		ppu.WINDOWS.WBGLOG(value)
	case 0x212B:
		if ppu.registerPreviousValues[0x2B] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x2B] = uint16(value)
		ppu.WINDOWS.WOBJLOG(value)
	case 0x212C:
		//fmt.Println("TM: ", value)
		if ppu.registerPreviousValues[0x2C] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x2C] = uint16(value)
		ppu.setTM(value)
		ppu.regenerateMainPipeline()
		ppu.invalidateAllLayers()
		ppu.markActiveWindowsDirty()
	case 0x212D:
		//fmt.Println("TS: ", value)
		if ppu.registerPreviousValues[0x2D] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x2D] = uint16(value)
		ppu.setTS(value)
		ppu.regenerateSubPipeline()
		ppu.invalidateAllLayers()
		ppu.markActiveWindowsDirty()
	case 0x212E:
		ppu.WINDOWS.TMW(value)
	case 0x212F:
		ppu.WINDOWS.TSW(value)
	case 0x2130:
		//fmt.Println("CGWSEL", value)
		ppu.WINDOWS.ColorMath.setCGWSEL(value, &ppu.Bg1.isDirectColor)
	case 0x2131:
		//fmt.Println("CGADSUB", value)
		ppu.WINDOWS.setCGADSUB(value)
	case 0x2132:
		//fmt.Println("COLDATA", value)
		ppu.WINDOWS.ColorMath.setCOLDATA(value)
	case 0x2133:
		//fmt.Println("SETINI", value)
		if ppu.registerPreviousValues[0x33] == uint16(value) {
			break
		}
		ppu.registerPreviousValues[0x33] = uint16(value)

		prevEXTBG := ppu.m7EXTBG
		ppu.setupSETINI(value)
		ppu.Framebuffer.CurrentHeight = ppu.ScreenHeight // - (1 << interlace)
		ppu.Framebuffer.Interlace = byte(interlace)

		if ppu.BGMODE == 7 && prevEXTBG != ppu.m7EXTBG {
			setMode7(ppu, false)
			ppu.regeneratePipelines()
			ppu.invalidateAllBackgrounds()
			ppu.markActiveWindowsDirty()
		}
	default:
		return fmt.Errorf("invalid PPU register write at $%04X", addr)
	}
	return nil
}

func (ppu *PPU) setupSETINI(value byte) {
	ppu.externalSync = value&0x80 != 0
	ppu.m7EXTBG = value&0x40 != 0
	pseudoHires = value & 8 >> 3
	interlace = uint16(value & 1)
	ppu.Obj.interlace = uint16(value & 0x02 >> 1)

	overscan := value&0x04 != 0
	ppu.Timing.activeVisibilityLUT = ppu.Timing.VisibilityLUTs[overscan]
	currentTimingRow = &(*ppu.Timing.activeVisibilityLUT)[ppu.V]
	ppu.ScreenHeight = getScreenHeight(overscan)
}

func (ppu *PPU) getOAMLow() []byte {
	return ppu.OAM.LowTable[:]
}

func (ppu *PPU) getOAMHigh() []byte {
	return ppu.OAM.HighTable[:]
}

func (ppu *PPU) getVRAM() []uint16 {
	return ppu.VRAM.VRAM[:]
}

func (ppu *PPU) getCGRAM() []uint16 {
	return ppu.CGRAM.CGRAM[:]
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

	ppu.WINDOWS.prepareToRebuild()
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

func (ppu *PPU) setHiresFlag() {
	if ppu.BGMODE == 5 || ppu.BGMODE == 6 {
		hires = 1
	} else {
		hires = 0
	}
}

func (ppu *PPU) LatchHV() {
	if *ppu.Wrio >= 0x80 {
		ppu.HLatch = ppu.H
		ppu.VLatch = ppu.V
		ppu.LatchFlag = 1
	}
}

func (ppu *PPU) returnAndSetPpu1OB(value byte) byte {
	ppu.ppu1OB = value
	return value
}

func (ppu *PPU) returnAndSetPpu2OB(value byte) byte {
	ppu.ppu2OB = value
	return value
}

func (ppu *PPU) isPpu1WriteRegisterRead(addr uint16) bool {
	if addr&0xFF00 != 0x2100 || (addr&0x00F0)>>4 > 2 {
		return false
	}
	lowNibble := addr & 0xF
	if (lowNibble >= 4 && lowNibble <= 6) || (lowNibble >= 8 && lowNibble <= 0xA) {
		return true
	}
	return false
}
