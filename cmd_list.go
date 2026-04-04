package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func CommandList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available boxnames",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			box, err := New(binaryname, "noname")
			if err != nil {
				return err
			}

			entries, err := os.ReadDir(box.HostConfigDir())
			if err != nil {
				return fmt.Errorf("read boxes: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
					continue
				}

				boxname := filepath.Base(entry.Name())

				cfg, err := Load(filepath.Join(box.HostConfigDir(), boxname))
				if err != nil {
					return fmt.Errorf("config: %w", err)
				}

				boxname = strings.TrimSuffix(boxname, filepath.Ext(boxname))
				fmt.Println(boxname, "-", cfg.Description)
			}

			return nil
		},
	}
}
