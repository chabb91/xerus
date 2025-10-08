package cpu

import (
	"SNES_emulator/memory"
)

const (
	irqId = iota
	nmiId
	abortId
	resetId
)

const (
	normalState = iota
	waitState
	stopState
)

type CPU struct {
	r *registers

	instructions       map[byte]Instruction
	currentInstruction Instruction

	hwInterrupts map[int]Instruction

	bus memory.Bus

	//placeholder. TODO make it into a channel or something nice
	abortSignal bool
	resetSignal bool
	NmiSignal   bool
	IrqSignal   bool

	executionState int
}

func NewCPU(bus memory.Bus) *CPU {
	cpu := &CPU{
		bus:                bus,
		r:                  &registers{},
		hwInterrupts:       NewHWInterruptMap(),
		instructions:       NewInstructionMap(),
		currentInstruction: nil,

		resetSignal: true,
	}
	return cpu
}

// TODO these signals should all be channels in the future
func (c *CPU) StepCycle() bool {
	if c.handleReset() {
		return false
	}
	if c.executionState == stopState {
		return false
	}
	if c.handleAbort() {
		return false
	}
	if c.handleNMI() {
		return false
	}
	if c.handleIRQ() {
		return false
	}
	if c.executionState != normalState {
		return false // stopped or waiting
	}
	return c.executeNextInstruction()
}
func (c *CPU) handleReset() bool {
	if !c.resetSignal {
		return false
	}
	c.currentInstruction = c.hwInterrupts[resetId]
	c.currentInstruction.Reset(c)
	c.resetSignal = false
	c.executionState = normalState
	return true
}

func (c *CPU) handleAbort() bool {
	if !c.abortSignal {
		return false
	}
	abort := c.hwInterrupts[abortId]
	//reset before assigning it to CPU to not break things.
	//this is a footgun so it should be addressed but this is the only place where its relevant
	//so ill just write this comment instead
	abort.Reset(c)
	c.currentInstruction = abort
	c.abortSignal = false
	c.executionState = normalState
	return true
}

func (c *CPU) handleNMI() bool {
	if !c.NmiSignal || c.currentInstruction != nil {
		return false
	}
	c.currentInstruction = c.hwInterrupts[nmiId]
	c.currentInstruction.Reset(c)
	c.NmiSignal = false
	c.executionState = normalState
	//nmi should be cleared from the source that called it i just dont have one yet
	//TODO
	return true
}

func (c *CPU) handleIRQ() bool {
	if !c.IrqSignal || c.currentInstruction != nil {
		return false
	}
	if !c.r.hasFlag(FlagI) {
		c.currentInstruction = c.hwInterrupts[irqId]
		c.currentInstruction.Reset(c)
		c.IrqSignal = false
		//irq should be cleared from the source that called it i just dont have one yet
		//If /IRQ is kept LOW then same (old) interrupt is executed again as soon as setting I=0. If /NMI is kept LOW then no further NMIs can be executed.
		//TODO
		c.executionState = normalState
		return true
	}
	if c.executionState == waitState {
		c.IrqSignal = false
		//irq should be cleared from the source that called it i just dont have one yet
		//If /IRQ is kept LOW then same (old) interrupt is executed again as soon as setting I=0. If /NMI is kept LOW then no further NMIs can be executed.
		//TODO
		c.executionState = normalState
	}
	return false
}

func (c *CPU) executeNextInstruction() bool {
	if c.currentInstruction == nil {
		c.r.instrPC = c.r.PC
		c.currentInstruction = c.instructions[c.fetchByte()]
		c.currentInstruction.Reset(c)
		return false
	}
	if c.currentInstruction.Step(c) {
		c.currentInstruction = nil
		return true
	}
	return false
}

// fetchByte maps PC to 24 bit then goes and reads a byte from memory
// then increases PC by 1
func (c *CPU) fetchByte() byte {
	ret := c.bus.ReadByte(mapOffsetToBank(c.r.PB, c.r.PC))
	c.r.PC++

	return ret
}

// PushByte pushes one byte onto the stack and updates SP.
func (cpu *CPU) PushByte(val byte) {
	addr := cpu.r.GetStackAddr()
	cpu.bus.WriteByte(addr, val)
	//cpu.r.S--
	cpu.r.SetStack(cpu.r.S - 1)
}

// PopByte pops one byte from the stack and updates SP.
func (cpu *CPU) PopByte() byte {
	//cpu.r.S++
	cpu.r.SetStack(cpu.r.S + 1)
	addr := cpu.r.GetStackAddr()
	return cpu.bus.ReadByte(addr)
}
