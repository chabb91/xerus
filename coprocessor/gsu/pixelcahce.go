package gsu

// heavily based on BSNES.
// let it be known:
// i had my own working PLOT implementation without the use of the double cache
// but the explanation in FULLSNES had me confused about caching
// and i wanted to increase accuracy.

type pixelCache struct {
	colorIdx   [8]byte //the color index as indexed by CGRAM for each pixel
	flags      byte
	xOffset, y byte //X&0xF8, Y&0xFF
}

func (pc *pixelCache) isNewRow(x, y byte) bool {
	return x&0xF8 != pc.xOffset || y != pc.y
}

func (gsu *GSU) getTileRowAddress(x, y uint16) (tra, bitplanes uint32) {
	screenHeight := (((gsu.r.SCMR & HT1) >> 4) | ((gsu.r.SCMR & HT0) >> 2)) |
		(byte(int8((gsu.r.POR&FlagForceObjMode)<<3))>>7)&3
	bitplanes = 2 << uint32((gsu.r.SCMR&MD0)+((gsu.r.SCMR&MD1)>>1))

	var tn uint16
	switch screenHeight {
	case 0:
		tn = ((x & 0xF8) << 1) + ((y & 0xF8) >> 3)
	case 1:
		tn = ((x & 0xF8) << 1) + ((x & 0xF8) >> 1) + ((y & 0xF8) >> 3)
	case 2:
		tn = ((x & 0xF8) << 1) + ((x & 0xF8) >> 0) + ((y & 0xF8) >> 3)
	case 3:
		tn = ((y & 0x80) << 2) + ((x & 0x80) << 1) + ((y & 0x78) << 1) + ((x & 0x78) >> 3)
	default:
		panic("GSU: getTilerowAddress: screenHeight is an unexpected value.")
	}

	tra = uint32(tn)*(bitplanes<<3) + gsu.r.SCBR + uint32((y&7)<<1)
	return
}

func (gsu *GSU) flushPixelCache(pc *pixelCache) {
	tra, bpp := gsu.getTileRowAddress(uint16(pc.xOffset), uint16(pc.y))
	for i := range bpp {
		addr := tra + ((i >> 1) << 4) + (i & 1)
		var data byte
		for j := range uint32(8) {
			data |= ((pc.colorIdx[j] >> i) & 1) << j
		}
		if pc.flags != 0xFF {
			data &= pc.flags
			bp, _ := gsu.Read8(byte(addr>>16), uint16(addr))
			data |= bp & ^pc.flags
		}
		gsu.Write8(byte(addr>>16), uint16(addr), data)
	}
}

func (gsu *GSU) rpix(x, y uint16) (data byte) {
	gsu.flushPixelCache(&gsu.pixelCaches[1])
	gsu.flushPixelCache(&gsu.pixelCaches[0])

	tra, bpp := gsu.getTileRowAddress(x, y)
	x = (x & 7) ^ 7
	for i := range bpp {
		addr := tra + ((i >> 1) << 4) + (i & 1)
		val, _ := gsu.Read8(byte(addr>>16), uint16(addr))
		data |= ((val >> x) & 1) << i
	}
	return
}

func (gsu *GSU) plot(x, y byte) {
	bitplanes := 2 << uint32((gsu.r.SCMR&MD0)+((gsu.r.SCMR&MD1)>>1))
	if gsu.r.POR&FlagPlotTransparent == 0 {
		transparentShift := bitplanes
		if transparentShift == 8 {
			transparentShift >>= (gsu.r.POR & FlagColorFreezeHigh) >> 3
		}
		if gsu.r.COLR&((1<<transparentShift)-1) == 0 {
			return
		}
	}
	color := gsu.r.COLR
	if gsu.r.POR&FlagPlotDither != 0 && bitplanes != 8 {
		color >>= (((x & 1) ^ (y & 1)) << 2)
		color &= 0xF
	}

	if gsu.pixelCaches[0].isNewRow(x, y) {
		gsu.flushPixelCache(&gsu.pixelCaches[1])
		gsu.pixelCaches[1] = gsu.pixelCaches[0]
		gsu.pixelCaches[0].flags = 0
		gsu.pixelCaches[0].xOffset = x & 0xF8
		gsu.pixelCaches[0].y = y
	}

	x = (x & 7) ^ 7
	gsu.pixelCaches[0].colorIdx[x] = color
	gsu.pixelCaches[0].flags |= 1 << x
	if gsu.pixelCaches[0].flags == 0xFF {
		gsu.flushPixelCache(&gsu.pixelCaches[1])
		gsu.pixelCaches[1] = gsu.pixelCaches[0]
		gsu.pixelCaches[0].flags = 0
	}
}
