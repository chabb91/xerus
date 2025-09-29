package dma

import (
	"fmt"
)

type DmaChannel struct {
	dmap  byte //control register
	bbad  byte //destination register
	a1tl  byte //dma source address low/hdma table address register
	a1th  byte //dma source address high/hdma table address register
	a1b   byte //dma source address bank/hdma table address register
	dasl  byte //dma size register low/hdma indirect address register
	dash  byte //dma size register high/hdma indirect address register
	dasb  byte //hdma indirect address register
	a2al  byte //hdma mid frame table address register low
	a2ah  byte //hdma mid frame table address register high
	ntlrx byte //hdma line counter register
}

type Dma struct {
	//both the previous and the current values have to be non zero for DMA to be triggered
	//update every cpu cycle. this is an easy way to give the cpu an extra cycle of operation
	MdmaenPrevious, Mdmaen byte
	Hdmaen                 byte

	Channels [8]DmaChannel
}

func (dma *Dma) Read(addr uint16) (byte, error) {
	b1, err := getChannelNum(addr)
	if err != nil {
		return 0, err
	}

	b2, err := getRegister(&dma.Channels[b1], addr)
	if err != nil {
		return 0, err
	}

	return *b2, nil
}

func (dma *Dma) Write(addr uint16, value byte) error {
	b1, err := getChannelNum(addr)
	if err != nil {
		return err
	}

	b2, err := getRegister(&dma.Channels[b1], addr)
	if err != nil {
		return err
	}

	*b2 = value
	return nil
}

func getRegister(channel *DmaChannel, address uint16) (*byte, error) {

	switch address & 0xF {
	case 0x0:
		return &channel.dmap, nil
	case 0x1:
		return &channel.bbad, nil
	case 0x2:
		return &channel.a1tl, nil
	case 0x3:
		return &channel.a1th, nil
	case 0x4:
		return &channel.a1b, nil
	case 0x5:
		return &channel.dasl, nil
	case 0x6:
		return &channel.dash, nil
	case 0x7:
		return &channel.dasb, nil
	case 0x8:
		return &channel.a2al, nil
	case 0x9:
		return &channel.a2ah, nil
	case 0xA:
		return &channel.ntlrx, nil
	default:
		return nil, fmt.Errorf("undefined DMA register $%04X", address)
	}
}

func getChannelNum(address uint16) (byte, error) {
	ret := (address & 0x00F0) >> 4
	if ret < 8 {
		return byte(ret), nil
	} else {
		return 0, fmt.Errorf("undefined DMA channel $%04X", ret)
	}
}
