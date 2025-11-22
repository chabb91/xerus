package cpu

import (
	"SNES_emulator/memory"
	"fmt"
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

	instructions       []Instruction
	currentInstruction Instruction

	hwInterrupts []Instruction
	bus          memory.Bus

	//placeholder. TODO make it into a channel or something nice
	abortSignal bool
	resetSignal bool
	NmiSignal   bool
	IrqSignal   bool

	executionState int

	//PLP, CLI, SEI, SEP #$04, and REP #$04 update the flags during their final CPU cycle, so the IRQ check will use the old value
	previousIFlag int
}

func NewCPU(bus memory.Bus) *CPU {
	cpu := &CPU{
		bus:                bus,
		r:                  &registers{},
		hwInterrupts:       NewHWInterruptMap(),
		instructions:       NewInstructionMap(),
		currentInstruction: nil,
		previousIFlag:      -1,

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
	//reset clears all other interrupts
	c.resetSignal = false
	c.abortSignal = false
	c.NmiSignal = false
	c.IrqSignal = false
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
	return true
}

// irq is cleared by it reading TIMEUP
func (c *CPU) handleIRQ() bool {
	if !c.IrqSignal || c.currentInstruction != nil {
		return false
	}

	var hasFlag bool
	if c.previousIFlag >= 0 {
		hasFlag = c.previousIFlag > 0
	} else {
		hasFlag = c.r.hasFlag(FlagI)
	}

	if !hasFlag {
		//fmt.Println("IRQ<<I not masked")
		c.currentInstruction = c.hwInterrupts[irqId]
		c.currentInstruction.Reset(c)
		c.executionState = normalState
		return true
	}
	if c.executionState == waitState {
		c.executionState = normalState
		fmt.Println("breaking wai")
	}
	//fmt.Println("IRQ<<I masked")
	return false
}

func (c *CPU) executeNextInstruction() bool {
	if c.currentInstruction == nil {
		c.r.instrPC = c.r.PC
		c.previousIFlag = -1
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
	cpu.bus.WriteByte(uint32(cpu.r.GetStack()), val)
	//cpu.r.S--
	cpu.r.SetStack(cpu.r.S - 1)
}

// PopByte pops one byte from the stack and updates SP.
func (cpu *CPU) PopByte() byte {
	//cpu.r.S++
	cpu.r.SetStack(cpu.r.S + 1)
	return cpu.bus.ReadByte(uint32(cpu.r.GetStack()))
}

// TODO this method might mess up the stack pouinter in emulation mode after an abort interrupt!
// new instructions(the ones expanding the original 6502 instructionset
// dont wrap the stack pointer till they are done
func (cpu *CPU) PushByteNewOpCode(val byte) {
	cpu.bus.WriteByte(uint32(cpu.r.S), val)
	cpu.r.S--
}

// new instructions(the ones expanding the original 6502 instructionset
// dont wrap the stack pointer till they are done
func (cpu *CPU) PopByteNewOpCode() byte {
	cpu.r.S++
	return cpu.bus.ReadByte(uint32(cpu.r.S))
}
