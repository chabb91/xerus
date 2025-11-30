package ppu

type SETINI struct {
	externalSync bool
	m7EXTBG      bool
	objInterlace uint16

	Timing VideoTiming

	ScreenHeight int
}

func NewSETINI(region VideoTiming) *SETINI {
	ret := &SETINI{
		Timing: region,
	}
	ret.setOverscan(false)
	return ret
}

func (s *SETINI) setup(value byte) {
	s.externalSync = value&0x80 != 0
	s.m7EXTBG = value&0x40 != 0
	pseudoHires = value & 8 >> 3
	s.setOverscan(value&0x04 != 0)
	s.objInterlace = uint16(value & 0x02 >> 1)
	interlace = uint16(value & 1)
}

func (s *SETINI) setOverscan(overscan bool) {
	currentTimingLUT = s.Timing.VisibilityLUTs[overscan]
	s.ScreenHeight = s.Timing.getScreenHeight(overscan)
}

func (s *SETINI) getScreenHeight() int {
	return s.ScreenHeight
}
