package ppu

import (
	"fmt"
	"time"
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
	ActionShortLine
	ActionLongLine
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

	H_TOTAL       = 341
	H_BLANK_START = 22
	H_BLANK_END   = 277

	NTSC_STANDARD_HEIGHT = 224
	NTSC_TOTAL_SCANLINES = 262

	PAL_STANDARD_HEIGHT = 239
	PAL_TOTAL_SCANLINES = 312
)

const LONG_SHORT_SCANLINE_H_TRIGGER = 339

type VisibilityEntry struct {
	H, V      uint16
	IsVisible bool
	Action    PPUAction
}

func (vt *VideoTiming) getScreenHeight(overscan bool) int {
	if overscan {
		return vt.OverscanHeight
	} else {
		return vt.ScreenHeight
	}
}

type VisibilityLUT []VisibilityEntry

type VideoTiming struct {
	ScreenWidth     int
	ScreenHeight    int
	OverscanHeight  int
	InterlaceHeight int

	TotalScanlines int
	TargetFrameMS  time.Duration
	RegionId       byte

	VisibilityLUTs map[bool]VisibilityLUT
}

var NTSC_TIMING = VideoTiming{
	ScreenWidth:     SCREEN_WIDTH,
	ScreenHeight:    NTSC_STANDARD_HEIGHT,
	OverscanHeight:  239,
	InterlaceHeight: NTSC_STANDARD_HEIGHT * 2,
	TotalScanlines:  NTSC_TOTAL_SCANLINES,
	TargetFrameMS:   time.Millisecond * 1000.0 / 60,
	RegionId:        0,
	VisibilityLUTs:  make(map[bool]VisibilityLUT),
}

var PAL_TIMING = VideoTiming{
	ScreenWidth:     SCREEN_WIDTH,
	ScreenHeight:    224,
	OverscanHeight:  PAL_STANDARD_HEIGHT,
	InterlaceHeight: PAL_STANDARD_HEIGHT * 2,
	TotalScanlines:  PAL_TOTAL_SCANLINES,
	TargetFrameMS:   time.Millisecond * 1000.0 / 50.0,
	RegionId:        1,
	VisibilityLUTs:  make(map[bool]VisibilityLUT),
}

func GenerateVisibilityLUT(timing *VideoTiming, isOverscan bool) VisibilityLUT {
	vActive := timing.ScreenHeight
	if isOverscan {
		vActive = timing.OverscanHeight
	}

	//added +1 to cover the extra scanline edge case every other frame
	lut := make(VisibilityLUT, (timing.TotalScanlines+1)*H_TOTAL)

	for v := 0; v < timing.TotalScanlines+1; v++ {
		for h := 0; h < H_TOTAL; h++ {

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
			if (v >= 0 && v < vActive) && h == 278 {
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
			if v == 311 && h == LONG_SHORT_SCANLINE_H_TRIGGER && timing.TotalScanlines == PAL_TOTAL_SCANLINES {
				action = setAction(action, ActionLongLine, v, h)
			}
			if v == 240 && h == LONG_SHORT_SCANLINE_H_TRIGGER && timing.TotalScanlines == NTSC_TOTAL_SCANLINES {
				action = setAction(action, ActionShortLine, v, h)
			}

			if (v == 311 || v == 240) &&
				(h == LONG_SHORT_SCANLINE_H_TRIGGER-1 ||
					h == LONG_SHORT_SCANLINE_H_TRIGGER+1) && action != ActionNone {
				panic(fmt.Sprintf("PPU: Long and Short scanline timings conflict with neighboring actions at V=%d H=%d", v, h))
			}

			isVisible := (h >= H_BLANK_START && h <= H_BLANK_END) && (v >= 1 && v <= vActive)

			key := v*H_TOTAL + h
			entry := VisibilityEntry{
				H:         uint16(h - H_BLANK_START),
				V:         uint16(v - 1),
				IsVisible: isVisible,
				Action:    action,
			}
			lut[key] = entry
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
