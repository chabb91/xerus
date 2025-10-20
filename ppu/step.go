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

		//TODO use PPU TIMING FOR THIS
		waitDuration := time.Duration(ppu.SETINI.Timing.TargetFrameMS) - elapsed

		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}
	}

	if !ppu.FBlank {
		draw := ppu.SETINI.TimingLUT[ppu.V*H_TOTAL+ppu.H]
		if draw.IsVisible {
			ppu.Framebuffer.Back[draw.H][draw.V] = ppu.Bg1.GetDotAt(draw.H, draw.V)
		}
	}

	switch ppu.H {
	case 274:
		ppu.HBlank = true
	case 1:
		ppu.HBlank = false
	}

	if ppu.V == ppu.SETINI.getScreenHeight()+1 && ppu.H == 0 {
		ppu.VBlank = true
		ppu.InterruptScheduler.SetRdnmi(true)
		ppu.Framebuffer.Swap()
	} else if ppu.V == 0 && ppu.H == 0 {
		ppu.VBlank = false
		ppu.InterruptScheduler.SetRdnmi(false)
	}
}

type InterruptScheduler interface {
	SetRdnmi(bool)
}
