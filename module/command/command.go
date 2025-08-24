package command

import (
	"encoding/json"
	"fmt"
)

type CmdType string

const (
	INSPECT_PANEL CmdType = "INSPECT_PANEL"
	THRUST        CmdType = "THRUST"
)

// Holds a passed command
type Command struct {
	CMD         string `json:"CMD"`
	CMD_COUNTER int    `json:"CMD_COUNTER"`
	CMD_HASH    string `json:"CMD_HASH"`
}

func ParseCommand(payload string) Command {
	var e Command
	// Parse the JSON payload into the Command struct
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		//panic(err)
		fmt.Println("ParseCommand error: %w", err)
		return Command{}
	}
	//if err := e.Command.Validate(); err != nil {
	//	panic(err)
	//}
	return e
}

// Validate ensures the Action is one of the allowed values
func (a CmdType) Validate() error {
	switch a {
	case INSPECT_PANEL, THRUST:
		return nil
	default:
		return fmt.Errorf("invalid action: %s", a)
	}
}
