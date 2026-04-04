package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func CommandDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete a box data & image for the given boxname",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			boxname := args[0]
			if !IsBoxnameValid(boxname) {
				return errors.New("invalid boxname")
			}

			box, err := New(binaryname, boxname)
			if err != nil {
				return err
			}

			config, err := Load(box.ConfigFilename())
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if config.DataDir == "" {
				config.DataDir = filepath.Join(box.HostDataDir(), boxname)
			}

			if AskConfirmation(fmt.Sprintf("Do you want to delete %s folder?", config.DataDir)) {
				err = os.RemoveAll(config.DataDir)
				if err != nil {
					return err
				}
			}

			cfg := BoxConfig{
				DestinationImage: box.ImageName(config.LocalImage),
			}

			removecmd := box.CraftRemoveImage(cfg)
			PrintCommand(removecmd...)
			if AskConfirmation("Do you want to execute?") {
				err = box.Exec(removecmd)
				if err != nil {
					fmt.Println("remove:", err)
				}
			}

			return nil
		},
	}
}
