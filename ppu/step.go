package ppu

import (
	"fmt"
	"time"
)

const (
	H_TOTAL = 341
	V_TOTAL = 262
)

const (
	NTSC_V      = 224
	PAL_V       = 240
	SCREEN_WITH = 256
)

const TargetFrameDuration = time.Millisecond * 1000

var frameStartTime time.Time

func (ppu *PPU) Step() {
	ppu.H++
	if ppu.H >= H_TOTAL {
		ppu.H = 0
		ppu.V++
		if ppu.V >= V_TOTAL {
			ppu.V = 0
		}
	}

	if ppu.V == 0 && ppu.H == 0 {
		frameStartTime = time.Now()
	}
	if ppu.V == 261 && ppu.H == H_TOTAL-1 {

		elapsed := time.Since(frameStartTime)
		fmt.Println(elapsed)

		waitDuration := TargetFrameDuration - elapsed

		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}
	}

	if !ppu.FBlank {
		draw := visibleLUTNtscNoInterlace[ppu.V][ppu.H]
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

	// NTSC
	// TODO account for pal and overscan and ecvrything else
	if ppu.V == 225 && ppu.H == 0 {
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
