package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func CommandCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create an image for the given boxname",
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

			{
				pullcmd := box.CraftPull(config.Create.Image)
				PrintCommand(pullcmd...)
				if config.Create.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
					return nil
				}

				err = box.Exec(pullcmd)
				if err != nil {
					return fmt.Errorf("pull: %w", err)
				}
			}

			var script bytes.Buffer
			script.WriteString(SetupEnv(box.HostUser()))
			script.WriteString("\n")
			script.WriteString(SetupSudoers())
			script.WriteString("\n")
			script.WriteString(SetupUser())
			script.WriteString("\n")
			script.WriteString("printf \"sdbx: Running user content...\\n\"\n")
			script.WriteString(config.Create.ShellScript)

			scriptname := filepath.Join(box.HostCacheDir(), boxname+"-setup.sh")
			err = os.WriteFile(scriptname, script.Bytes(), 0o644)
			if err != nil {
				return fmt.Errorf("write setup file: %s: %w", scriptname, err)
			}
			defer os.Remove(scriptname)

			cfg := BoxConfig{
				Name:             boxname,
				SourceImage:      config.Create.Image,
				DestinationImage: box.ImageName(config.LocalImage),
				ExtraArgs: []string{
					"--user", "root:root",
					"--userns", "keep-id",
					"--annotation", "run.oci.keep_original_groups=1",
				},
				Command: "/bin/sh /usr/bin/setup.sh",
				Volumes: []string{
					fmt.Sprintf("%s:%s:rw,U", config.DataDir, box.HostUser().HomeDir),
					scriptname + ":/usr/bin/setup.sh:ro",
				},
				Environment: config.Create.Environment,
			}

			{
				// Remove previous box container if aborted during create/update process.
				removecmd := box.CraftRemove(cfg)
				PrintCommand(removecmd...)
				if config.Create.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
					return nil
				}

				err = box.Exec(removecmd)
				if err != nil {
					fmt.Println("remove:", err)
				}
			}

			{
				// Remove previous box image if exists.
				removecmd := box.CraftRemoveImage(cfg)
				PrintCommand(removecmd...)
				if config.Create.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
					return nil
				}

				err = box.Exec(removecmd)
				if err != nil {
					fmt.Println("remove:", err)
				}
			}

			{
				runcmd := box.CraftRun(cfg)
				PrintCommand(runcmd...)
				if config.Create.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
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
				if config.Create.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
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
				if config.Create.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
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
