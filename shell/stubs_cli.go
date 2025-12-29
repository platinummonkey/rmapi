package shell

import (
	"fmt"
)

// Stub implementations for commands that will be removed in later tasks
// These are temporary to keep the code compiling

func lsCommand(ctx *Context) Command {
	return Command{
		Name: "ls",
		Help: "list contents (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("ls command not yet converted to CLI")
		},
	}
}

func pwdCommand(ctx *Context) Command {
	return Command{
		Name: "pwd",
		Help: "print working directory (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("pwd command not yet converted to CLI")
		},
	}
}

func cdCommand(ctx *Context) Command {
	return Command{
		Name: "cd",
		Help: "change directory (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("cd command not yet converted to CLI")
		},
	}
}

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

func mkdirCommand(ctx *Context) Command {
	return Command{
		Name: "mkdir",
		Help: "create directory (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("mkdir command not yet converted to CLI")
		},
	}
}

func rmCommand(ctx *Context) Command {
	return Command{
		Name: "rm",
		Help: "remove file/directory (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("rm command not yet converted to CLI")
		},
	}
}

func mvCommand(ctx *Context) Command {
	return Command{
		Name: "mv",
		Help: "move file/directory (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("mv command not yet converted to CLI")
		},
	}
}

func putCommand(ctx *Context) Command {
	return Command{
		Name: "put",
		Help: "upload file (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("put command not yet converted to CLI")
		},
	}
}

func mputCommand(ctx *Context) Command {
	return Command{
		Name: "mput",
		Help: "recursive upload (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("mput command not yet converted to CLI")
		},
	}
}

func statCommand(ctx *Context) Command {
	return Command{
		Name: "stat",
		Help: "show file status (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("stat command not yet converted to CLI")
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

func findCommand(ctx *Context) Command {
	return Command{
		Name: "find",
		Help: "search for files (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("find command not yet converted to CLI")
		},
	}
}

func nukeCommand(ctx *Context) Command {
	return Command{
		Name: "nuke",
		Help: "delete all files (will be removed in simplification)",
		Func: func(ctx *Context, args []string) error {
			return fmt.Errorf("nuke command not yet converted to CLI")
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
