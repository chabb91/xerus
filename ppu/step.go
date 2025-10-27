package ppu

import (
	"fmt"
	"time"
)

const TargetFrameDuration = time.Millisecond * 1000

var frameStartTime time.Time

func (ppu *PPU) Step() {
	ppu.H++
	if ppu.H >= H_TOTAL {
		ppu.H = 0
		ppu.V++
		if ppu.V >= ppu.SETINI.Timing.TotalScanlines {
			ppu.V = 0
		}
	}

	if ppu.V == 0 && ppu.H == 0 {
		frameStartTime = time.Now()
	}
	if ppu.V == ppu.SETINI.Timing.TotalScanlines-1 && ppu.H == H_TOTAL-1 {

		elapsed := time.Since(frameStartTime)
		fmt.Println(elapsed)

		waitDuration := time.Duration(ppu.SETINI.Timing.TargetFrameMS) - elapsed

		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}
	}

	draw := ppu.SETINI.TimingLUT[ppu.V*H_TOTAL+ppu.H]
	if !ppu.FBlank {
		if draw.IsVisible {
			if ppu.WINDOWS.isDotMasked(bg1, false, draw.H) {
				ppu.Framebuffer.Back[draw.H][draw.V].SetColor(ppu.CGRAM.CGRAM[0], ppu.brightness)
			} else {
				ppu.Framebuffer.Back[draw.H][draw.V].SetColor(ppu.Bg1.GetDotAt(draw.H, draw.V), ppu.brightness)
				//ppu.Framebuffer.Back[draw.H][draw.V] = addColors(ppu.Bg1.GetDotAt(draw.H, draw.V), ppu.Bg2.GetDotAt(draw.H, draw.V), false)
			}
		}
	}

	if draw.Action != ActionNone {
		ppu.performAction(draw.Action)
	}
}

type InterruptScheduler interface {
	SetRdnmi(bool)
	FireNmi()
}

type HdmaScheduler interface {
	Reload()
	DoTransfer()
}

func (ppu *PPU) performAction(action PPUAction) {
	switch action {
	case ActionVBlankStart:
		ppu.VBlank = true
		ppu.InterruptScheduler.FireNmi()
		ppu.Framebuffer.Swap()
	case ActionVBlankEnd:
		ppu.VBlank = false
		ppu.InterruptScheduler.SetRdnmi(false)
	case ActionSetRdnmi:
		ppu.InterruptScheduler.SetRdnmi(true)
	case ActionHBlankStart:
		ppu.HBlank = true
	case ActionHBlankEnd:
		ppu.HBlank = false
	case ActionHBlankEndInterlaceFieldToggle:
		ppu.HBlank = false
	case ActionOAMReset:
		ppu.OAM.InvalidateInternalIndex()
	case ActionHDMAStart:
		ppu.HdmaScheduler.DoTransfer()
	case ActionHDMAReload:
		ppu.HdmaScheduler.Reload()
	case ActionShortLine:
	case ActionLongLine:
	case ActionSetNmi:
	case ActionJoypadReadStart:
	case ActionCpuRefresh:
	}
}
