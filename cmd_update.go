package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func CommandUpdate() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update an image for the given boxname",
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

			err = os.MkdirAll(config.DataDir, 0o755)
			if err != nil {
				return fmt.Errorf("mkdir_all: %w", err)
			}

			var script bytes.Buffer
			script.WriteString(SetupEnv(box.HostUser()))
			script.WriteString("\n")
			script.WriteString("printf \"sdbx: Running user content...\\n\"\n")
			script.WriteString(config.Update.ShellScript)

			scriptname := filepath.Join(box.HostCacheDir(), boxname+"-update.sh")
			err = os.WriteFile(scriptname, script.Bytes(), 0o644)
			if err != nil {
				return fmt.Errorf("write update file: %s: %w", scriptname, err)
			}
			defer os.Remove(scriptname)

			usr := box.HostUser()
			cfg := BoxConfig{
				Name:             boxname,
				SourceImage:      box.ImageName(config.LocalImage),
				DestinationImage: box.ImageName(config.LocalImage),
				ExtraArgs: []string{
					"--user", fmt.Sprintf("%s:%s", usr.Uid, usr.Gid),
					"--userns", "keep-id",
					"--annotation", "run.oci.keep_original_groups=1",
				},
				Command: "/bin/bash /usr/bin/update.sh",
				Volumes: []string{
					fmt.Sprintf("%s:%s:rw,U", config.DataDir, usr.HomeDir),
					scriptname + ":/usr/bin/update.sh:ro",
				},
			}

			{
				runcmd := box.CraftRun(cfg)
				PrintCommand(runcmd...)
				if config.Update.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
					return nil
				}

				err = box.Exec(runcmd)
				if err != nil {
					return fmt.Errorf("run: %w", err)
				}
			}

			{
				commitcmd := box.CraftCommit(cfg)
				PrintCommand(commitcmd...)
				if config.Update.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
					return nil
				}

				err = box.Exec(commitcmd)
				if err != nil {
					return fmt.Errorf("commit: %w", err)
				}
			}

			{
				removecmd := box.CraftRemove(cfg)
				PrintCommand(removecmd...)
				if config.Update.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
					return nil
				}

				err = box.Exec(removecmd)
				if err != nil {
					return fmt.Errorf("remove: %w", err)
				}
			}

			return nil
		},
	}
}
