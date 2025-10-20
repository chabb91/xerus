package ppu

import "time"

type PPUAction byte

const (
	ActionNone PPUAction = iota
	ActionVBlankStart
	ActionVBlankEnd
	ActionHBlankStart
	ActionHBlankEnd
	ActionHBlankEndInterlaceFieldToggle
	ActionOAMReset
	ActionHDMAStart
	ActionHDMAReload
	ActionShortLine
	ActionLongLine
	ActionSetNmi
	ActionJoypadReadStart
	ActionJoypadReadEnd
	ActionCpuRefresh
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

type VisibilityEntry struct {
	H, V      uint16
	IsVisible bool
	Action    PPUAction
}

func (vt VideoTiming) getScreenHeight(overscan bool) int {
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

	VisibilityLUTs map[bool]VisibilityLUT
}

var NTSC_TIMING = VideoTiming{
	ScreenWidth:     SCREEN_WIDTH,
	ScreenHeight:    NTSC_STANDARD_HEIGHT,
	OverscanHeight:  239,
	InterlaceHeight: NTSC_STANDARD_HEIGHT * 2,
	TotalScanlines:  NTSC_TOTAL_SCANLINES,
	TargetFrameMS:   time.Millisecond * 1000.0 / 60,
	VisibilityLUTs:  make(map[bool]VisibilityLUT),
}

var PAL_TIMING = VideoTiming{
	ScreenWidth:     SCREEN_WIDTH,
	ScreenHeight:    224,
	OverscanHeight:  PAL_STANDARD_HEIGHT,
	InterlaceHeight: PAL_STANDARD_HEIGHT * 2,
	TotalScanlines:  PAL_TOTAL_SCANLINES,
	TargetFrameMS:   time.Millisecond * 1000.0 / 50.0,
	VisibilityLUTs:  make(map[bool]VisibilityLUT),
}

func GenerateVisibilityLUT(timing *VideoTiming, isOverscan bool) VisibilityLUT {
	vActive := timing.ScreenHeight
	if isOverscan {
		vActive = timing.OverscanHeight
	}

	lut := make(VisibilityLUT, timing.TotalScanlines*H_TOTAL)

	for v := 0; v < timing.TotalScanlines; v++ {
		for h := 0; h < H_TOTAL; h++ {

			action := ActionNone
			if v == 0 && h == 0 {
				action = ActionVBlankEnd
			}
			if v == vActive+1 && h == 0 {
				action = ActionVBlankStart
				//TODO this also sets NMI 2(0.5dots) cycles later.
			}
			if v == vActive+1 && h == 10 {
				action = ActionOAMReset
			}
			if v == vActive+1 && h == 33 {
				action = ActionJoypadReadStart
			}
			if v == 0 && h == 6 {
				action = ActionHDMAReload
			}
			if (v >= 0 && v <= vActive) && h == 278 {
				action = ActionHDMAStart
			}
			if h == 1 {
				action = ActionHBlankEnd
				if v == 0 {
					action = ActionHBlankEndInterlaceFieldToggle
				}
			}
			if h == 274 {
				action = ActionHBlankStart
			}
			if h == 134 {
				action = ActionCpuRefresh
			}
			if v == 311 && h == 23 && timing.TotalScanlines == PAL_TOTAL_SCANLINES {
				//TODO find the best way to handle the infinite loop this causes with just h--
				//luckily this is interlace only
				action = ActionLongLine
			}
			if v == 240 && h == 23 && timing.TotalScanlines == NTSC_TOTAL_SCANLINES {
				action = ActionShortLine
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
