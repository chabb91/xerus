package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chabb91/xerus/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

type InputDeviceConfig struct {
	Port     int
	Source   string // "keyboard", "controller"
	DeviceID int    // 0, 1, etc.
}

type InputDeviceMapping []InputDeviceConfig //implements flag.Value

func (dm *InputDeviceMapping) String() string {
	return fmt.Sprint(*dm)
}

func (dm *InputDeviceMapping) Set(value string) error {
	pairs := strings.SplitSeq(value, ",")
	for pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: %s (expected portX:typeY)", pair)
		}

		portStr := strings.TrimPrefix(parts[0], "port")
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", parts[0])
		} else if port > 3 {
			return fmt.Errorf("invalid port: %v, port numbers must be between 0-3", port)
		}

		source := parts[1]
		id := 0
		if after, ok := strings.CutPrefix(source, "controller"); ok {
			idStr := after
			id, err = strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid controller format: %s (expected controllerX)", source)
			}
			source = "controller"
		}

		*dm = append(*dm, InputDeviceConfig{
			Port:     port,
			Source:   source,
			DeviceID: id,
		})
	}
	return nil
}

func (dm *InputDeviceMapping) generateControllerTemplate() []ui.SnesInput {
	ret := make([]ui.SnesInput, 4)
	ret[0] = &ui.SNESKeyboardInput{} //fallback
	ret[1] = &ui.NullInput{}
	ret[2] = &ui.NullInput{}
	ret[3] = &ui.NullInput{}

	for _, v := range *dm {
		if v.Port > len(ret) {
			continue
		}
		if v.Source == "keyboard" {
			ret[v.Port] = &ui.SNESKeyboardInput{}
		}
		if v.Source == "controller" {
			ret[v.Port] = ui.NewSnesControllerInput(ebiten.GamepadID(v.DeviceID))
		}
	}
	return ret
}
