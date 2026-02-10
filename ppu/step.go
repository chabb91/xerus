package ppu

func (ppu *PPU) Step() {
	draw := currentTimingLUT[ppu.V*H_TOTAL+ppu.H]
	if draw.IsVisible {
		h := draw.H
		v := draw.V<<interlace + (interlaceStep & interlace)
		if ppu.FBlank {
			ppu.Framebuffer.Back[h][v].SetColor(0, 0, ppu.brightness)
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
					ppu.Framebuffer.Back[h][v].SetColor(cm2, cm1, ppu.brightness)
				} else {
					ppu.Framebuffer.Back[h][v].SetColor(ss, ms, ppu.brightness)
				}
			} else {
				ms, l1, math := ppu.renderMainScreen(h, v)
				if math {
					ss, l2, _ := ppu.renderSubScreen(h, v)
					cm1 := ppu.WINDOWS.performColorMath(ms, ss, h, l1, l2)
					ppu.Framebuffer.Back[h][v].SetColor(cm1, cm1, ppu.brightness)
				} else {
					ppu.Framebuffer.Back[h][v].SetColor(ms, ms, ppu.brightness)
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

	ppu.H++
	if ppu.H >= H_TOTAL {
		ppu.H = 0
		ppu.V++
		timing := &ppu.SETINI.Timing
		if ppu.V >= timing.TotalScanlines+int(interlace&(1-interlaceStep)) {
			ppu.V = 0
		}
	}
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
		if interlace == 0 || interlace&interlaceStep == 1 {
			ppu.Framebuffer.Swap()
		}
		interlaceStep ^= 1
	case ActionVBlankEnd:
		ppu.VBlank = false
		ppu.InterruptScheduler.SetRdnmi(false)
		if !ppu.FBlank {
			ppu.Obj.resetTimeAndRange()
		}
	case ActionHBlankStart:
		ppu.HBlank = true
	case ActionHBlankEnd:
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
	case ActionShortLine:
		if interlace == 0 && interlaceStep == 1 {
			ppu.H++
		}
	case ActionLongLine:
		if interlace == 1 {
			if interlaceStep == 1 {
				if interlaceLongLine {
					ppu.H--
					interlaceLongLine = false
				}
			} else {
				interlaceLongLine = true
			}
		}
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
