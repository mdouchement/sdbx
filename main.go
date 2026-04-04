package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

const binaryname = "podman"

func main() {
	c := &cobra.Command{
		Use:     "sdbx",
		Short:   "Sandbox for AI agent",
		Version: Version(),
		Args:    cobra.NoArgs,
	}
	c.AddCommand(CommandList())
	c.AddCommand(CommandCreate())
	c.AddCommand(CommandUpdate())
	c.AddCommand(CommandRun())
	c.AddCommand(CommandDelete())
	c.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Version for arx",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(c.Version)
		},
	})

	if err := c.Execute(); err != nil {
		log.Fatal(err)
	}
}
