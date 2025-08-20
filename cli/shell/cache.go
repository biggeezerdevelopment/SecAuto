package shell

import (
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterCacheCommands registers cache-related commands
func RegisterCacheCommands(sh *ishell.Shell, ctx *Context) {
	cacheCmd := &ishell.Cmd{
		Name: "cache",
		Help: "Manage cache operations",
	}

	// cache stats
	cacheCmd.AddCmd(&ishell.Cmd{
		Name: "stats",
		Help: "Show cache statistics",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching cache statistics..."
			}

			stats, err := ctx.Client.GetCacheStats()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get cache stats: %v", err))
				return
			}

			if err := ctx.Printer.Print(stats); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// cache get
	cacheCmd.AddCmd(&ishell.Cmd{
		Name: "get",
		Help: "Get cache value (get <key>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "cache get <key>") {
				return
			}

			key := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Getting cache value..."
			}

			value, err := ctx.Client.GetCacheKey(key)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get cache key '%s': %v", key, err))
				return
			}

			result := map[string]interface{}{
				"key":   key,
				"value": value,
			}

			if err := ctx.Printer.Print(result); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// cache set
	cacheCmd.AddCmd(&ishell.Cmd{
		Name: "set",
		Help: "Set cache value (set <key> <value>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 2, "cache set <key> <value>") {
				return
			}

			key := c.Args[0]
			value := c.Args[1]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Setting cache value..."
			}

			err := ctx.Client.SetCacheKey(key, value)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to set cache key '%s': %v", key, err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Cache key '%s' set successfully", key))
		},
	})

	// cache delete
	cacheCmd.AddCmd(&ishell.Cmd{
		Name: "delete",
		Help: "Delete cache key (delete <key>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "cache delete <key>") {
				return
			}

			key := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Deleting cache key..."
			}

			err := ctx.Client.DeleteCacheKey(key)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to delete cache key '%s': %v", key, err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Cache key '%s' deleted successfully", key))
		},
	})

	// cache clear
	cacheCmd.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "Clear all cache entries",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			// Confirm clearing
			c.Printf("Are you sure you want to clear all cache entries? (y/N): ")
			confirmation := c.ReadLine()
			if confirmation != "y" && confirmation != "yes" && confirmation != "Y" && confirmation != "YES" {
				c.Printf("Clear operation cancelled\n")
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Clearing cache..."
			}

			err := ctx.Client.ClearCache()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to clear cache: %v", err))
				return
			}

			ctx.PrintSuccess(c, "Cache cleared successfully")
		},
	})

	sh.AddCmd(cacheCmd)
}