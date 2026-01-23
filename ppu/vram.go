package ppu

type vmainRemap func(uint16) uint16

// video RAM
type VRAMController struct {
	VRAM []uint16

	remap               vmainRemap
	incrementOnHighByte bool
	incrementAmount     uint16
	vmadd               uint16

	vmLatchedValue uint16 // for VRAM register reads

	tv tileValidator
}

func NewVRAM(tv tileValidator) *VRAMController {
	return &VRAMController{
		VRAM: make([]uint16, 0x8000),
		tv:   tv,
	}
}

func (vram *VRAMController) ReadDataLow() byte {
	ret := byte(vram.vmLatchedValue)

	if !vram.incrementOnHighByte {
		vram.vmLatchedValue = vram.VRAM[vram.remap(vram.vmadd)]
		vram.vmadd += vram.incrementAmount
	}
	return ret
}

func (vram *VRAMController) ReadDataHigh() byte {
	ret := byte(vram.vmLatchedValue >> 8)

	if vram.incrementOnHighByte {
		vram.vmLatchedValue = vram.VRAM[vram.remap(vram.vmadd)]
		vram.vmadd += vram.incrementAmount
	}
	return ret
}

func (vram *VRAMController) UpdateAddressLow(value byte) {
	vram.vmadd = (vram.vmadd & 0xFF00) | uint16(value)
	vram.vmLatchedValue = vram.VRAM[vram.remap(vram.vmadd)]
}

func (vram *VRAMController) UpdateAddressHigh(value byte) {
	vram.vmadd = (vram.vmadd & 0xFF) | (uint16(value) << 8)
	vram.vmLatchedValue = vram.VRAM[vram.remap(vram.vmadd)]
}

func (vram *VRAMController) WriteDataLow(value byte) {
	remappedAddr := vram.remap(vram.vmadd)
	vram.VRAM[remappedAddr] = (vram.VRAM[remappedAddr] & 0xFF00) | uint16(value)

	vram.tv.tryInvalidate(remappedAddr)

	if !vram.incrementOnHighByte {
		vram.vmadd += vram.incrementAmount
	}
}

func (vram *VRAMController) WriteDataHigh(value byte) {
	remappedAddr := vram.remap(vram.vmadd)
	vram.VRAM[remappedAddr] = (vram.VRAM[remappedAddr] & 0x00FF) | (uint16(value) << 8)

	vram.tv.tryInvalidate(remappedAddr)

	if vram.incrementOnHighByte {
		vram.vmadd += vram.incrementAmount
	}
}

func (vram *VRAMController) setupVMAIN(vmain byte) {
	if vmain >= 0x80 {
		vram.incrementOnHighByte = true
	} else {
		vram.incrementOnHighByte = false
	}

	switch vmain & 0x3 {
	case 0:
		vram.incrementAmount = 0x01
	case 1:
		vram.incrementAmount = 0x20
	default:
		vram.incrementAmount = 0x80
	}

	switch (vmain & 0xC) >> 2 {
	case 0:
		vram.remap = vmainRemap00
	case 1:
		vram.remap = vmainRemap01
	case 2:
		vram.remap = vmainRemap10
	case 3:
		vram.remap = vmainRemap11
	}

}

// TODO write a test for these mappers im not made of binary and its hard to tell what all this shifting amounts to
func vmainRemap00(value uint16) uint16 {
	return value & 0x7FFF
}

func vmainRemap01(value uint16) uint16 {
	bbb := (value >> 5) & 0x7
	return (((value << 3) | bbb) & 0x00FF) | (value & 0x7F00)
}

func vmainRemap10(value uint16) uint16 {
	bbb := (value >> 6) & 0x7
	return (((value << 3) | bbb) & 0x01FF) | (value & 0x7E00)
}

func vmainRemap11(value uint16) uint16 {
	bbb := (value >> 7) & 0x7
	return (((value << 3) | bbb) & 0x03FF) | (value & 0x7C00)
}
