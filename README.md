# SNES-emulator (better name pending)

This is a SNES emulator written in GO.

## Reason for creation

I tried to write an emulator that despite of potentially never living up to the established ones out there, I could use for playing games I would actually want to play instead of just writing one as a proof of concept and never using it after. The SNES has many titles that are beloved and highly playable even today and I wanted to experience them, so despite the considerable challenges, it seemed like the obvious choice.

Why not use the C/C++? GO is the language I'm trying to learn and most source code is C++ already, making this a bit more unique.

## Building

**SNES-emulator** only has one dependency at this moment i.e. [Ebitengine](https://ebitengine.org/) which is used to handle displaying the image, reading controller input and audio playback.
In order to build the project **Ebitengine** and all of its dependencies need to be [installed](https://ebitengine.org/en/documents/install.html).

Then the project can be ran like:

```bash
go run . [options] <rom-path>
```

Or built like:

```bash
go build -o SNES-emulator
```

_Note_: Building it with the tag `GOAMD64=v3` or `GOAMD64=v4` for AVX2 or AVX512 machines respectively may improve performance.

## Running

Running the project from the terminal without arguments will display a detailed usage information.
Currently flags are just there to force PAL/NTSC mode or to enable performance profiling.

A config file is planned for the future for controller configuration because for now the base setup is set in stone.

The rom header is automatically detected and PAL/NTSC execution modes are set accordingly.
Sram is automatically detected, created/loaded and saved on exiting. The resulting `.srm` file is compatible with **BSNES** and probably all other emulators as well.

## Compatibility

Despite my best efforts trying to get the timings right by following Anomie's docs, and getting the overall execution pace very close to **BSNES**, it is not even close to being cycle accurate on the micro scale. This results in some games locking up or exhibiting various visual glitches. Many games do boot and run mostly fine however. For some reason the HDMA execution speed seems way too fast (or there is some unimplemented quirk) and this breaks timing sensitive games like MK3. My sample size is quite limited but I assume there are a good amount of games able to be completed from start to finish.

## Progress

### [Ricoh 5A22](https://en.wikipedia.org/wiki/Ricoh_5A22)

The SNES SoC is fully implemented with a cycle accurate 65c816 cpu at its core meaning instructions always take the correct amount of cycles to complete and these cycles also respect the variable access speeds of the underlying memory bus.

### PPU

The PPU is completely implemented respecting all rendering modes and visual effects, interlacing and windowing.

### APU

- SPC700: The SPC700 audio chip is fully implemented with its own memory controller, timers and cycle accurate instructions. It passes blargg's `spc_smp.sfc` test.

- DSP: The dsp chip is currently work in progress. The current implementation is just a proof of concept. Audio playback is mostly fine but all voices are calculated in one batch. There is a rewrite coming up however in which I will attempt to mimic the 32 step sound sample generation pipeline that can be found on hardware.

### Scheduler

As mentioned before, the scheduler tries its best to respect Anomie's `timing.txt` but its nowhere near fully cycle accurate.

### Coprocessors

- [x] GSU: Passes all instruction tests by PeterLemon, and 8 out of the 10 existing games boot and run (albeit some of them exhibit glitches). Winter Gold plays the intro and hangs on menu and Dirt Trax FX shows no sign of life.
- [ ] Other ones TBD

## Future Goals

- Keep implementing coprocessors
- Improve timings
- Improve the S-DSP
- Improve compatibility
- Create config files

## References

- [The best 65c816 reference](http://www.6502.org/tutorials/65c816opcodes.html)
- [snes.nesdev.org](https://snes.nesdev.org)
- [Super Nintendo Development Wiki](https://wiki.superfamicom.org/)
- [Anomie's .txt docs](https://www.romhacking.net/?page=documents&category=&platform=9&game=&author=548&perpage=20&level=&title=&desc=&docsearch=Go)
- [Peter Lemon's test roms](https://github.com/PeterLemon/SNES)
- [bbbradsmith's test roms](https://github.com/bbbradsmith/SNES_stuff)
- [nesdoug's SNES demo roms](https://github.com/nesdoug)
- [The higan snes test rom repo](https://gitlab.com/higan/snes-test-roms)
- [The mame c++ implementation of the spc700](https://github.com/mamedev/mame/blob/master/src/devices/cpu/spc700/spc700.cpp)
- [Single step tests for both the cpu and apu](https://github.com/SingleStepTests/ProcessorTests)
