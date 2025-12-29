package shell

import (
	"fmt"
)

// Stub implementations for commands that will be removed in later tasks
// These are temporary to keep the code compiling

func getCommand(ctx *Context) Command {
	return Command{
		Name: "get",
		Help: "download file (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("get command not yet converted to CLI")
		},
	}
}

func mgetCommand(ctx *Context) Command {
	return Command{
		Name: "mget",
		Help: "recursive download (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("mget command not yet converted to CLI")
		},
	}
}


func getaCommand(ctx *Context) Command {
	return Command{
		Name: "geta",
		Help: "download and convert to PDF (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("geta command not yet converted to CLI")
		},
	}
}

func refreshCommand(ctx *Context) Command {
	return Command{
		Name: "refresh",
		Help: "refresh file tree (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("refresh command not yet converted to CLI")
		},
	}
}
