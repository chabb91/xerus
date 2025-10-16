package ppu

const (
	H_TOTAL = 341
	V_TOTAL = 262
)

func (ppu *PPU) Step() {
	ppu.H++
	if ppu.H >= H_TOTAL {
		ppu.H = 0
		ppu.V++
		if ppu.V >= V_TOTAL {
			ppu.V = 0
		}
	}

	if !ppu.FBlank {
		draw := visibleLUTNtscNoInterlace[ppu.V][ppu.H]
		if draw.IsVisible {
			ppu.Bg1.GetDotAt(draw.H, draw.V)
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
	} else if ppu.V == 0 && ppu.H == 0 {
		ppu.VBlank = false
	}
}

type InterruptScheduler interface {
	SetRdnmi(bool)
}
