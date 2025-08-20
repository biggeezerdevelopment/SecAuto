package shell

import (
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterHealthCommands registers health-related commands
func RegisterHealthCommands(sh *ishell.Shell, ctx *Context) {
	sh.AddCmd(&ishell.Cmd{
		Name: "health",
		Help: "Check server health and connectivity",
		Func: func(c *ishell.Context) {
			if err := ctx.EnsureClient(); err != nil {
				ctx.PrintError(c, err)
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Checking server health..."
			}

			start := time.Now()
			err := ctx.Client.HealthCheck()
			duration := time.Since(start)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("health check failed: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Server is healthy (response time: %v)", duration))
		},
	})
}