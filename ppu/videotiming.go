package ppu

import (
	"fmt"
)

type PPUAction byte

const (
	ActionNone PPUAction = iota
	ActionVBlankStart
	ActionVBlankEnd
	ActionHBlankStart
	ActionHBlankEnd
	ActionOAMReset
	ActionHDMAStart
	ActionHDMAReload
	ActionSetNmi
	//ActionSetRdnmi
	ActionJoypadReadStart
	ActionJoypadReadEnd
	ActionCpuRefresh
	ActionSetHvbjoyV
	ActionResetHvbjoyV
	ActionSetHvbjoyH
	ActionResetHvbjoyH
	ActionPrepareScanline //arbitrary action to pre calculate stuff for every scanline
)

const (
	SCREEN_WIDTH = 256

	H_TOTAL       = 340
	H_BLANK_START = 22
	H_BLANK_END   = 277

	SCREEN_HEIGHT          = 224
	SCREEN_HEIGHT_OVERSCAN = 239

	NTSC_TOTAL_SCANLINES = 263 //accounting for the extra scanline at interlace field 0
	PAL_TOTAL_SCANLINES  = 313
)

type VisibilityEntry struct {
	H, V      uint16
	IsVisible bool
	Action    PPUAction
}

func getScreenHeight(overscan bool) int {
	if overscan {
		return SCREEN_HEIGHT_OVERSCAN
	} else {
		return SCREEN_HEIGHT
	}
}

type VisibilityLUT [][H_TOTAL + 1]VisibilityEntry

type VideoTiming struct {
	TotalScanlines int
	RegionId       byte
	Pal            bool

	VisibilityLUTs map[bool]VisibilityLUT
}

var NTSC_TIMING = VideoTiming{
	TotalScanlines: NTSC_TOTAL_SCANLINES,
	RegionId:       0,
	Pal:            false,
	VisibilityLUTs: make(map[bool]VisibilityLUT),
}

var PAL_TIMING = VideoTiming{
	TotalScanlines: PAL_TOTAL_SCANLINES,
	RegionId:       1,
	Pal:            true,
	VisibilityLUTs: make(map[bool]VisibilityLUT),
}

func GenerateVisibilityLUT(timing *VideoTiming, isOverscan bool) VisibilityLUT {
	vActive := getScreenHeight(isOverscan)

	//added +1 to cover the extra scanline edge case every other frame
	lut := make(VisibilityLUT, timing.TotalScanlines)

	for v := 0; v < timing.TotalScanlines; v++ {
		for h := 0; h < H_TOTAL+1; h++ {

			action := ActionNone
			if v == 0 && h == 0 {
				action = setAction(action, ActionVBlankEnd, v, h)
				//this also clears nmi if it hasnt been consumed yet
			}
			if v == vActive+1 && h == 0 {
				action = setAction(action, ActionVBlankStart, v, h)
				//TODO this also sets NMI 2(0.5dots) cycles later.
			}
			if v == 0 && h == 30 {
				action = setAction(action, ActionResetHvbjoyV, v, h)
			}
			if v == vActive+1 && h == 22 {
				action = setAction(action, ActionSetHvbjoyV, v, h)
			}
			if v == vActive+1 && h == 10 {
				action = setAction(action, ActionOAMReset, v, h)
			}
			if v == vActive+1 && h == 33 {
				action = setAction(action, ActionJoypadReadStart, v, h)
			}
			if v == vActive+4 && h == 66 {
				action = setAction(action, ActionJoypadReadEnd, v, h)
			}
			if v == 0 && h == 6 {
				action = setAction(action, ActionHDMAReload, v, h)
			}
			if (v >= 0 && v <= vActive) && h == 278 {
				action = setAction(action, ActionHDMAStart, v, h)
			}
			if (v >= 1 && v <= vActive) && h == H_BLANK_START-1 {
				action = setAction(action, ActionPrepareScanline, v, h)
			}
			if h == 1 {
				action = setAction(action, ActionHBlankEnd, v, h)
			}
			if h == 274 {
				action = setAction(action, ActionHBlankStart, v, h)
			}
			if h == 23 {
				action = setAction(action, ActionResetHvbjoyH, v, h)
			}
			if h == 289 {
				action = setAction(action, ActionSetHvbjoyH, v, h)
			}
			if h == 134 {
				action = setAction(action, ActionCpuRefresh, v, h)
			}

			isVisible := (h >= H_BLANK_START && h <= H_BLANK_END) && (v >= 1 && v <= vActive)

			entry := VisibilityEntry{
				H:         uint16(h - H_BLANK_START),
				V:         uint16(v - 1),
				IsVisible: isVisible,
				Action:    action,
			}
			lut[v][h] = entry
		}
	}

	return lut
}

func setAction(current PPUAction, newVal PPUAction, v, h int) PPUAction {
	if current != ActionNone {
		panic(fmt.Sprintf("PPU: Action collision at V=%d H=%d: tried to set %v but %v already exists, cannot create the LUT.",
			v, h, newVal, current))
	}
	return newVal
}
