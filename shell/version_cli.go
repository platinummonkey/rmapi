package shell

import (
	"fmt"

	"github.com/juruen/rmapi/version"
)

func versionCommand(ctx *Context) Command {
	return Command{
		Name: "version",
		Help: "show rmapi version",
		Func: func(ctx *Context, args []string) error {
			fmt.Println("rmapi version:", version.Version)
			return nil
		},
	}
}
