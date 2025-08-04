package main

import (
	"SNES_emulator/debugger"
	"fmt"
)

func main() {
	t, err := debugger.LoadTests("testdata/4c.json")
	if err == nil {
		fmt.Println(t)
		for _, v := range t {

			fmt.Println(v.Cycles)
		}
	} else {
		fmt.Println(err)
	}

}
