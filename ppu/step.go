package ppu

func (ppu *PPU) Step() uint64 {
	draw := currentTimingLUT[ppu.V][ppu.H] //TODO cache &currentTimingLUT[V] between scanlines
	if draw.IsVisible {
		h := draw.H
		v := draw.V<<interlace + (interlaceStep & interlace)
		if h == 0 {
			currentPixelBufferRow = &ppu.Framebuffer.Back[v]
		}
		if ppu.FBlank {
			currentPixelBufferRow[h].SetColor(0, 0, ppu.brightness)
		} else {
			if hires == 1 || pseudoHires == 1 {
				//flipping this causes artifacts because the subscreen is always first in the rendering order
				ss, l2, _ := ppu.renderSubScreen(h, v)
				ms, l1, math := ppu.renderMainScreen(h, v)
				if math {
					//TODO subscreen color math unfortunately depends on the previous ms pixel. this matters in some cases
					//try fixing it later
					cm1 := ppu.WINDOWS.performColorMath(ms, ss, h, l1, l2)
					cm2 := ppu.WINDOWS.performColorMath(ss, ms, h, l2, l1)
					currentPixelBufferRow[h].SetColor(cm2, cm1, ppu.brightness)
				} else {
					currentPixelBufferRow[h].SetColor(ss, ms, ppu.brightness)
				}
			} else {
				ms, l1, math := ppu.renderMainScreen(h, v)
				if math {
					ss, l2, _ := ppu.renderSubScreen(h, v)
					cm1 := ppu.WINDOWS.performColorMath(ms, ss, h, l1, l2)
					currentPixelBufferRow[h].SetColor(cm1, cm1, ppu.brightness)
				} else {
					currentPixelBufferRow[h].SetColor(ms, ms, ppu.brightness)
				}
			}
		}
	}

	if ppu.WINDOWS.rebuildNeeded {
		if ppu.WINDOWS.invalidationCounter > 0 {
			ppu.WINDOWS.invalidationCounter--
		} else {
			ppu.WINDOWS.rebuildDirtyLayerWindowCaches()
		}
	}

	if draw.Action != ActionNone {
		ppu.performAction(draw)
	}

	if ppu.IrqTimeUpTimer > 0 {
		ppu.IrqTimeUpTimer--
		if ppu.IrqTimeUpTimer == 0 {
			ppu.InterruptScheduler.SetTimeUp()
		}
	}

	if irqf := ppu.IrqFunc; irqf != nil && irqf() {
		//if !(ppu.V == 261 && ppu.H == 339) {
		ppu.InterruptScheduler.FireIrq()
		ppu.IrqTimeUpTimer = 3 //2.5 - 3.5 dots after irq fires TIMEUP gets set as i understand
		//}
	}

	cycles := uint64(3)
	if (ppu.H != 323 && ppu.H != 327) ||
		//NTSC Short Line check
		(!ppu.SETINI.Timing.Pal && ppu.V == 240 && (interlace^1)&interlaceStep == 1) {
		cycles = 2
	}

	ppu.H++
	if ppu.H >= ppu.HTotal {
		ppu.H = 0
		ppu.V++
		//PAL Long line check
		if ppu.SETINI.Timing.Pal && ppu.V == 311 && interlaceStep&interlace == 1 {
			ppu.HTotal = H_TOTAL
		} else {
			ppu.HTotal = H_TOTAL - 1
		}
		if ppu.V >= ppu.SETINI.Timing.TotalScanlines-(int(interlace&(interlaceStep^1))^1) {
			ppu.V = 0
		}
	}
	return cycles
}

type InterruptScheduler interface {
	SetRdnmi(bool)
	SetHvbjoyV(bool)
	SetHvbjoyH(bool)
	SetHvbjoyA(bool)
	SetTimeUp()
	FireNmi()
	FireIrq()
}

type HdmaScheduler interface {
	Reload()
	DoTransfer()
}

func (ppu *PPU) performAction(draw VisibilityEntry) {
	switch draw.Action {
	case ActionVBlankStart:
		ppu.VBlank = true
		ppu.InterruptScheduler.FireNmi()
		ppu.InterruptScheduler.SetRdnmi(true)
		if interlace == 0 || interlaceStep == 1 {
			ppu.Framebuffer.Swap()
		}
	case ActionVBlankEnd:
		ppu.VBlank = false
		ppu.InterruptScheduler.SetRdnmi(false)
		if !ppu.FBlank {
			ppu.Obj.resetTimeAndRange()
		}
	case ActionHBlankStart:
		ppu.HBlank = true
	case ActionHBlankEnd:
		if ppu.V == 0 {
			interlaceStep ^= 1
		}
		ppu.HBlank = false
	case ActionSetHvbjoyV:
		ppu.InterruptScheduler.SetHvbjoyV(true)
	case ActionSetHvbjoyH:
		ppu.InterruptScheduler.SetHvbjoyH(true)
	case ActionResetHvbjoyV:
		ppu.InterruptScheduler.SetHvbjoyV(false)
	case ActionResetHvbjoyH:
		ppu.InterruptScheduler.SetHvbjoyH(false)
	case ActionOAMReset:
		ppu.OAM.InvalidateInternalIndex()
	case ActionHDMAStart:
		ppu.HdmaScheduler.DoTransfer()
	case ActionHDMAReload:
		ppu.HdmaScheduler.Reload()
	case ActionSetNmi:
		//TODO
	case ActionJoypadReadStart:
		ppu.InterruptScheduler.SetHvbjoyA(true)
	case ActionJoypadReadEnd:
		ppu.InterruptScheduler.SetHvbjoyA(false)
	case ActionCpuRefresh:
		ppu.Refresh = true
	case ActionPrepareScanline:
		ppu.Obj.prepareScanLine(draw.V<<ppu.SETINI.objInterlace + interlace&interlaceStep)

		shouldReset := true
		if hasMosaic && mosaicSize > 1 {
			if draw.V == 0 {
				mosaicLineCnt = 0
			}
			if draw.V > mosaicStartLine {
				mosaicLineCnt++
				shouldReset = mosaicLineCnt == uint16(mosaicSize)
				if mosaicLineCnt >= uint16(mosaicSize) {
					mosaicLineCnt = 0
				}
			}
		}

		if !ppu.Bg1.mosaic || shouldReset {
			ppu.Bg1.renderCacheEnd = 0
			if ppu.BGMODE == 7 {
				ppu.Mode7.prepareScanLine(draw.V<<interlace + (interlaceStep & interlace))
			}
		}
		if !ppu.Bg2.mosaic || shouldReset {
			ppu.Bg2.renderCacheEnd = 0
		}
		if !ppu.Bg3.mosaic || shouldReset {
			ppu.Bg3.renderCacheEnd = 0
		}
		if !ppu.Bg4.mosaic || shouldReset {
			ppu.Bg4.renderCacheEnd = 0
		}
	}
}
