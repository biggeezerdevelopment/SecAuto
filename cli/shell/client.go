package shell

import (
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterClientCommands registers client-related commands
func RegisterClientCommands(sh *ishell.Shell, ctx *Context) {
	clientCmd := &ishell.Cmd{
		Name: "client",
		Help: "Manage clients",
	}

	// client list
	clientCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all clients",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching clients..."
			}

			clients, err := ctx.Client.ListClients()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list clients: %v", err))
				return
			}

			if len(clients) == 0 {
				c.Printf("No clients found\n")
				return
			}

			if err := ctx.Printer.Print(clients); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// client get
	clientCmd.AddCmd(&ishell.Cmd{
		Name: "get",
		Help: "Get client details (get <client-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "client get <client-id>") {
				return
			}

			clientID := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching client details..."
			}

			clientInfo, err := ctx.Client.GetClient(clientID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get client '%s': %v", clientID, err))
				return
			}

			if err := ctx.Printer.Print(clientInfo); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// client delete
	clientCmd.AddCmd(&ishell.Cmd{
		Name: "delete",
		Help: "Delete a client (delete <client-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "client delete <client-id>") {
				return
			}

			clientID := c.Args[0]

			// Confirm deletion
			c.Printf("Are you sure you want to delete client '%s'? (y/N): ", clientID)
			confirmation := c.ReadLine()
			if confirmation != "y" && confirmation != "yes" && confirmation != "Y" && confirmation != "YES" {
				c.Printf("Deletion cancelled\n")
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Deleting client..."
			}

			err := ctx.Client.DeleteClient(clientID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("delete failed: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Client '%s' deleted successfully", clientID))
		},
	})

	sh.AddCmd(clientCmd)
}