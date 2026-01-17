# SNES-emulator (better name pending)

This is a SNES emulator written in GO.

## Reason for creation

I tried to write an emulator that despite of potentially never living up to the established ones out there, I could use for playing games I would actually want to play instead of just writing one as a proof of concept and never using it after. The SNES has many titles that are beloved and highly playable even today and I wanted to experience them, so despite the considerable challenges, it seemed like the obvious choice.

Why not use the C/C++? GO is the language I'm trying to learn and to my knowledge there is no other GO based SNES emulator.

## Building

**SNES-emulator** only has one dependency at this moment i.e. [Ebitengine](https://ebitengine.org/) which is used to handle displaying the image, reading controller input and (hopefully soon) audio playback.
In order to build the project **Ebitengine** and all of its dependencies need to be [installed](https://ebitengine.org/en/documents/install.html).

Then the project can be ran like:

```bash
go run .
```

Or built like:

```bash
go build -o SNES-emulator
```

## Specifying a Super Nintendo image (.sfc)

Right now there are absolutely no conveniences in place. `.sfc` files can be executed by editing the path directly in `soc/soc.go` and manually specifying the rom type _(lo or hi)_ right below it.  
PAL/NTSC modes can be manually selected in `ppu/ppu.go`.

A CLI is planned however to make this seamless.

## Compatibility

Despite my best efforts trying to get the timings right by following Anomie's docs, and getting the overall execution pace very close to **BSNES**, it is not even close to being cycle accurate on the micro scale. This results in some games locking up or exhibiting various visual glitches. Many games do boot and run mostly fine however. My sample size is quite limited but I assume there are a good amount of games able to be completed from start to finish.

## Progress

### [Ricoh 5A22](https://en.wikipedia.org/wiki/Ricoh_5A22)

The SNES SoC is fully implemented with a cycle accurate 65c816 cpu at its core meaning instructions always take the correct amount of cycles to complete and these cycles also respect the variable access speeds of the underlying memory bus.

### PPU

The PPU is completely implemented respecting all rendering modes and visual effects, interlacing and windowing.

### APU

#### SPC700

The SPC700 audio chip is fully implemented with its own memory controller, timers and cycle accurate instructions. It passes blargg's `spc_smp.sfc` test.

#### DSP

The dsp chip is currently work in progress.

### Scheduler

As mentioned before, the scheduler tries its best to respect Anomie's `timing.txt` but its nowhere near fully cycle accurate.

### Coprocessors

Haven't started working on them yet.

## Future Goals

- Learn enough about waveform audio to be able to finish the DSP.
- Create a CLI and implement automatic rom header detection
- Implement coprocessors
- Improve timings

## References

- [The best 65c816 reference](http://www.6502.org/tutorials/65c816opcodes.html)
- [snes.nesdev.org](https://snes.nesdev.org)
- [Super Nintendo Development Wiki](https://wiki.superfamicom.org/)
- Anomie's .txt docs
- [Peter Lemon's test roms](https://github.com/PeterLemon/SNES)
- [bbbradsmith's test roms](https://github.com/bbbradsmith/SNES_stuff)
- [nesdoug's SNES demo roms](https://github.com/nesdoug)
- [The higan snes test rom repo](https://gitlab.com/higan/snes-test-roms)
- The mame c++ implementation of the spc700
- [Single step tests for both the cpu and apu](https://github.com/SingleStepTests/ProcessorTests)
