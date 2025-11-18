// bad practce in general but not going to use dependency injection
// or dereferencing in general 100k times per frame just to write "correct" code
// the rendering pipeline gets to have a few fast magical package level variables
// there are never two ppu instances anyway.
package ppu

import "time"

// used in renderMainScreen and renderSubScreen for extremely fast access
var colorCache [7]uint16
var spritePrio byte
var spriteMath bool

// used in step for tracking frame times
var frameStartTime time.Time

// global mosaic values
var mosaicSize byte
var mosaicStartLine uint16
var mosaicLineCnt uint16
var hasMosaic bool

// are we in hires or pseudo hires modes
var hires byte
var interlace uint16
var interlaceStep uint16 //odd or even frame. 0 even 1 odd.

var bgmode byte
