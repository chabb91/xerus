package ppu

type bgModeSetter func(ppu *PPU, value byte, mode1Prio, isExtBg bool)
type pipelineStep struct {
	layer    ppuLayer
	priority byte
}

var bgModeLUT uint16
