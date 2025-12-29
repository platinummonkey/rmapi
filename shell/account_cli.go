package shell

import (
	"fmt"
)

func accountCommand(ctx *Context) Command {
	return Command{
		Name: "account",
		Help: "account info",
		Func: func(ctx *Context, args []string) error {
			fmt.Printf("User: %s, SyncVersion: %v\n", ctx.UserInfo.User, ctx.UserInfo.SyncVersion)
			return nil
		},
	}
}
