package ppu

// video RAM
type VRAMController struct {
	VRAM []uint16

	vmain *VMAIN
	vmadd uint16

	tv tileValidator

	//absolute cringe VERY speshul case for VRAM register reads
	vmLatchedValue uint16
}

func NewVRAM(tv tileValidator) *VRAMController {
	return &VRAMController{
		VRAM:  make([]uint16, 0x8000),
		vmain: newVMAIN(),
		tv:    tv,
	}
}

func (vram *VRAMController) ReadDataLow() byte {
	ret := byte(vram.vmLatchedValue)

	if !vram.vmain.incrementOnHighByte {
		vram.vmLatchedValue = vram.VRAM[vram.vmain.remapAndMask(vram.vmadd)]
		vram.vmadd += vram.vmain.incrementAmount
	}
	return ret
}

func (vram *VRAMController) ReadDataHigh() byte {
	ret := byte(vram.vmLatchedValue >> 8)

	if vram.vmain.incrementOnHighByte {
		vram.vmLatchedValue = vram.VRAM[vram.vmain.remapAndMask(vram.vmadd)]
		vram.vmadd += vram.vmain.incrementAmount
	}
	return ret
}

func (vram *VRAMController) UpdateAddressLow(value byte) {
	vram.vmadd = (vram.vmadd & 0xFF00) | uint16(value)
}

func (vram *VRAMController) UpdateAddressHigh(value byte) {
	vram.vmadd = (vram.vmadd & 0xFF) | (uint16(value) << 8)
}

func (vram *VRAMController) WriteDataLow(value byte) {
	remapped_addr := vram.vmain.remapAndMask(vram.vmadd)
	vram.VRAM[remapped_addr] = (vram.VRAM[remapped_addr] & 0xFF00) | uint16(value)

	vram.tv.tryInvalidate(remapped_addr)

	if !vram.vmain.incrementOnHighByte {
		vram.vmadd += vram.vmain.incrementAmount
	}
}

func (vram *VRAMController) WriteDataHigh(value byte) {
	remapped_addr := vram.vmain.remapAndMask(vram.vmadd)
	vram.VRAM[remapped_addr] = (vram.VRAM[remapped_addr] & 0x00FF) | (uint16(value) << 8)

	vram.tv.tryInvalidate(remapped_addr)

	if vram.vmain.incrementOnHighByte {
		vram.vmadd += vram.vmain.incrementAmount
	}
}

type vmainRemap func(uint16) uint16

type VMAIN struct {
	incrementOnHighByte bool
	incrementAmount     uint16
	remap               vmainRemap
}

func newVMAIN() *VMAIN {
	vmain := &VMAIN{}
	vmain.Setup(0)
	return vmain
}

func (vm *VMAIN) Setup(vmain byte) {
	if vmain >= 0x80 {
		vm.incrementOnHighByte = true
	} else {
		vm.incrementOnHighByte = false
	}

	switch vmain & 0x3 {
	case 0:
		vm.incrementAmount = 0x01
	case 1:
		vm.incrementAmount = 0x20
	default:
		vm.incrementAmount = 0x80
	}

	switch (vmain & 0xC) >> 2 {
	case 0:
		vm.remap = vmainRemap00
	case 1:
		vm.remap = vmainRemap01
	case 2:
		vm.remap = vmainRemap10
	case 3:
		vm.remap = vmainRemap11
	}

}

func (vm *VMAIN) remapAndMask(value uint16) uint16 {
	return vm.remap(value) & 0x7FFF
}

// TODO write a test for these mappers im not made of binary and its hard to tell what all this shifting amounts to
func vmainRemap00(value uint16) uint16 {
	return value
}

func vmainRemap01(value uint16) uint16 {
	bbb := (value >> 5) & 0x7
	return (((value << 3) | bbb) & 0x00FF) | (value & 0xFF00)
}

func vmainRemap10(value uint16) uint16 {
	bbb := (value >> 6) & 0x7
	return (((value << 3) | bbb) & 0x01FF) | (value & 0xFE00)
}

func vmainRemap11(value uint16) uint16 {
	bbb := (value >> 7) & 0x7
	return (((value << 3) | bbb) & 0x03FF) | (value & 0xFC00)
}
