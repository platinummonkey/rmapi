package shell

import (
	"fmt"
	"os"
	"sort"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/model"
)

// Command represents a CLI command
type Command struct {
	Name string
	Help string
	Func func(ctx *Context, args []string) error
}

// Context holds the execution context for commands
type Context struct {
	node           *model.Node
	api            api.ApiCtx
	path           string
	useHiddenFiles bool
	UserInfo       api.UserInfo
}

func useHiddenFiles() bool {
	val, ok := os.LookupEnv("RMAPI_USE_HIDDEN_FILES")
	if !ok {
		return false
	}
	return val != "0"
}

// RunCLI executes CLI commands without interactive shell
func RunCLI(apiCtx api.ApiCtx, userInfo *api.UserInfo, args []string) error {
	ctx := &Context{
		node:           apiCtx.Filetree().Root(),
		api:            apiCtx,
		path:           apiCtx.Filetree().Root().Name(),
		useHiddenFiles: useHiddenFiles(),
		UserInfo:       *userInfo,
	}

	// Register all commands
	commands := make(map[string]Command)
	registerCommand(commands, getCommand(ctx))
	registerCommand(commands, mgetCommand(ctx))
	registerCommand(commands, mgetaCommand(ctx))
	registerCommand(commands, versionCommand(ctx))
	registerCommand(commands, getaCommand(ctx))
	registerCommand(commands, accountCommand(ctx))
	registerCommand(commands, refreshCommand(ctx))

	if len(args) == 0 {
		printUsage(commands)
		return nil
	}

	cmdName := args[0]
	cmd, ok := commands[cmdName]
	if !ok {
		return fmt.Errorf("unknown command: %s\n\nRun 'rmapi help' for usage", cmdName)
	}

	return cmd.Func(ctx, args[1:])
}

func registerCommand(commands map[string]Command, cmd Command) {
	commands[cmd.Name] = cmd
}

func printUsage(commands map[string]Command) {
	fmt.Println("rmapi - reMarkable Cloud API CLI")
	fmt.Println("\nUsage: rmapi <command> [options]")
	fmt.Println("\nAvailable commands:")

	// Sort commands alphabetically
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := commands[name]
		fmt.Printf("  %-12s %s\n", name, cmd.Help)
	}

	fmt.Println("\nFor command-specific help, use: rmapi <command> -h")
}
