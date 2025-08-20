package shell

import (
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterIntegrationCommands registers integration-related commands
func RegisterIntegrationCommands(sh *ishell.Shell, ctx *Context) {
	integrationCmd := &ishell.Cmd{
		Name: "integration",
		Help: "Manage integrations",
	}

	// integration list
	integrationCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all integrations",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching integrations..."
			}

			integrations, err := ctx.Client.ListIntegrations()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list integrations: %v", err))
				return
			}

			if len(integrations) == 0 {
				c.Printf("No integrations found\n")
				return
			}

			// Convert to format suitable for table output
			var data []map[string]interface{}
			for i, integration := range integrations {
				data = append(data, map[string]interface{}{
					"#":    i + 1,
					"Name": integration,
				})
			}

			if err := ctx.Printer.Print(data); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// integration info
	integrationCmd.AddCmd(&ishell.Cmd{
		Name: "info",
		Help: "Get integration details (info <name>)",
		Func: func(c *ishell.Context) {
			c.Printf(`
Integrations in SecAuto are meant to be executed in specific contexts:

1. Through Playbooks:
   Create a playbook that calls the integration:
   [{"integration": {"name": "example1", "function": "list_file", "params": {"hash": "abc123"}}}]

2. Through Client-Specific Execution:
   Use the /clients/{id}/integrations/{name}/execute endpoint for client-specific configurations.

3. For Testing:
   Integrations should be tested within the proper client or playbook context.

To see integration details, use: integration get <name>
To execute an integration, create a playbook that calls it.
`)
		},
	})

	// integration get
	integrationCmd.AddCmd(&ishell.Cmd{
		Name: "get",
		Help: "Get integration configuration (get <name>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "integration get <name>") {
				return
			}

			name := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = fmt.Sprintf(" Getting integration '%s'...", name)
			}

			result, err := ctx.Client.GetIntegration(name)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get integration '%s': %v", name, err))
				return
			}

			if err := ctx.Printer.Print(result); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	sh.AddCmd(integrationCmd)
}