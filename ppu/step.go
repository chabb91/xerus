package ppu

import (
	"fmt"
	"time"
)

const TargetFrameDuration = time.Millisecond * 1000

var frameStartTime time.Time

func (ppu *PPU) Step() {
	//TODO create setini struct so overscan can be tracked
	screenHeight := ppu.Timing.getScreenHeight(false)
	ppu.H++
	if ppu.H >= H_TOTAL {
		ppu.H = 0
		ppu.V++
		if ppu.V >= ppu.Timing.TotalScanlines {
			ppu.V = 0
		}
	}

	if ppu.V == 0 && ppu.H == 0 {
		frameStartTime = time.Now()
	}
	if ppu.V == ppu.Timing.TotalScanlines-1 && ppu.H == H_TOTAL-1 {

		elapsed := time.Since(frameStartTime)
		fmt.Println(elapsed)

		//TODO use PPU TIMING FOR THIS
		waitDuration := TargetFrameDuration - elapsed

		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}
	}

	if !ppu.FBlank {
		//TODO create setini struct so overscan can be tracked
		//draw := ppu.Timing.VisibilityLUTs[false][ppu.V*H_TOTAL+ppu.H]
		draw := ppu.ActiveLUT[ppu.V*H_TOTAL+ppu.H]
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

	if ppu.V == screenHeight+1 && ppu.H == 0 {
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
