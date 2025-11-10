// bad practce in general but not going to use dependency injection
// or dereferencing in general 100k times per frame just to write "correct" code
// the rendering pipeline gets to have a few fast magical package level variables
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
var mosaicStartLine byte
