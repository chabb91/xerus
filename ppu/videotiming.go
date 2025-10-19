package ppu

type PPUAction byte

const (
	ActionNone PPUAction = iota
	ActionVBlankStart
	ActionVBlankEnd
	ActionOAMReset
	ActionHDMAStart
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
	TargetFrameMS  float64

	VisibilityLUTs map[bool]VisibilityLUT
}

var NTSC_TIMING = VideoTiming{
	ScreenWidth:     SCREEN_WIDTH,
	ScreenHeight:    NTSC_STANDARD_HEIGHT,
	OverscanHeight:  239,
	InterlaceHeight: NTSC_STANDARD_HEIGHT * 2,
	TotalScanlines:  NTSC_TOTAL_SCANLINES,
	TargetFrameMS:   1000.0 / 60.0988,
	VisibilityLUTs:  make(map[bool]VisibilityLUT),
}

var PAL_TIMING = VideoTiming{
	ScreenWidth:     SCREEN_WIDTH,
	ScreenHeight:    PAL_STANDARD_HEIGHT,
	OverscanHeight:  239,
	InterlaceHeight: PAL_STANDARD_HEIGHT * 2,
	TotalScanlines:  PAL_TOTAL_SCANLINES,
	TargetFrameMS:   1000.0 / 50.0,
	VisibilityLUTs:  make(map[bool]VisibilityLUT),
}

func GenerateVisibilityLUT(timing *VideoTiming, isOverscan bool) VisibilityLUT {
	vActive := 224
	if isOverscan {
		vActive = 239
	}

	lut := make(VisibilityLUT, timing.TotalScanlines*H_TOTAL)

	for v := 0; v < timing.TotalScanlines; v++ {
		for h := 0; h < H_TOTAL; h++ {

			isVisible := (h >= H_BLANK_START && h <= H_BLANK_END) && (v >= 1 && v < vActive+1)

			key := v*H_TOTAL + h
			entry := VisibilityEntry{
				H:         uint16(h - H_BLANK_START),
				V:         uint16(v - 1),
				IsVisible: isVisible,
				Action:    ActionNone,
			}
			lut[key] = entry
		}
	}

	return lut
}
