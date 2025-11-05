package ppu

import (
	"fmt"
	"time"
)

var frameStartTime time.Time

func (ppu *PPU) Step() {
	timing := ppu.SETINI.Timing

	draw := ppu.SETINI.TimingLUT[ppu.V*H_TOTAL+ppu.H]
	if !ppu.FBlank {
		if draw.IsVisible {
			ms, l1, math := ppu.renderMainScreen(draw.H, draw.V)
			if math {
				ss, _, _ := ppu.renderSubScreen(draw.H, draw.V)
				ppu.Framebuffer.Back[draw.H][draw.V].SetColor(ppu.WINDOWS.performColorMath(ms, ss, draw.H, l1), ppu.brightness)
			} else {
				ppu.Framebuffer.Back[draw.H][draw.V].SetColor(ms, ppu.brightness)
			}
		}
	}

	if draw.Action != ActionNone {
		ppu.performAction(draw)
	}

	ppu.H++
	if ppu.H >= H_TOTAL {
		ppu.H = 0
		ppu.V++
		if ppu.V >= timing.TotalScanlines {
			ppu.V = 0

			elapsed := time.Since(frameStartTime)
			fmt.Println(elapsed)

			waitDuration := time.Duration(timing.TargetFrameMS) - elapsed

			if waitDuration > 0 {
				time.Sleep(waitDuration)
			}
			frameStartTime = time.Now()
		}
	}
}

type InterruptScheduler interface {
	SetRdnmi(bool)
	SetHvbjoyV(bool)
	SetHvbjoyH(bool)
	FireNmi()
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
		ppu.InterruptScheduler.SetHvbjoyV(true)
		ppu.Framebuffer.Swap()
	case ActionVBlankEnd:
		ppu.VBlank = false
		ppu.InterruptScheduler.SetRdnmi(false)
		ppu.InterruptScheduler.SetHvbjoyV(false)
	case ActionSetRdnmi:
		ppu.InterruptScheduler.SetRdnmi(true)
	case ActionHBlankStart:
		ppu.HBlank = true
		ppu.InterruptScheduler.SetHvbjoyH(true)
	case ActionHBlankEnd:
		ppu.HBlank = false
		//TODO HVBJoY troggers on a slightly different timer
		ppu.InterruptScheduler.SetHvbjoyH(false)
	case ActionHBlankEndInterlaceFieldToggle:
		ppu.HBlank = false
		ppu.InterruptScheduler.SetHvbjoyH(false)
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
	case ActionPrepareScanline:
		if ppu.Obj.isActive() {
			ppu.Obj.prepareScanLine(draw.V)
		}
	}
}
