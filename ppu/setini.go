package ppu

// TODO what is this clown file even. needs rewrite
type SETINI struct {
	externalSync bool
	m7EXTBG      bool
	//hires        bool
	overscan     bool
	objInterlace uint16
	//screenInterlace byte

	Timing       VideoTiming
	TimingLUT    VisibilityLUT
	ScreenHeight int
}

func NewSETINI(region VideoTiming) *SETINI {
	ret := &SETINI{
		Timing:       region,
		TimingLUT:    region.VisibilityLUTs[false],
		ScreenHeight: region.getScreenHeight(false),
	}
	ret.setup(0)
	return ret
}

func (s *SETINI) setup(value byte) {
	s.externalSync = value&0x80 != 0
	s.m7EXTBG = value&0x40 != 0
	//s.hires = value&0x08 != 0
	pseudoHires = value & 8 >> 3
	s.setOverscan(value&0x04 != 0)
	s.objInterlace = uint16(value & 0x02 >> 1)
	//s.screenInterlace = value & 1
	interlace = uint16(value & 1)
	//interlaceStep = 0
}

func (s *SETINI) setOverscan(overscan bool) {
	s.TimingLUT = s.Timing.VisibilityLUTs[overscan]
	s.ScreenHeight = s.Timing.getScreenHeight(overscan)
	s.overscan = overscan
}

func (s *SETINI) getScreenHeight() int {
	return s.ScreenHeight
}

func (s *SETINI) getScreenWidth() int {
	return s.Timing.ScreenWidth
}
