package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func CommandRun() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run an image for the given boxname",
		Args:  cobra.RangeArgs(1, 1),
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

			usr := box.HostUser()

			currentdir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current dir: %w", err)
			}

			var script bytes.Buffer
			script.WriteString(SetupEnv(box.HostUser()))
			script.WriteString("\n")
			script.WriteString(ReChown())
			script.WriteString("\n")
			script.WriteString("printf \"sdbx: Running user content...\\n\"\n")
			script.WriteString(config.Run.Command)

			scriptname := filepath.Join(box.HostCacheDir(), boxname+"-entrypoint.sh")
			err = os.WriteFile(scriptname, script.Bytes(), 0o644)
			if err != nil {
				return fmt.Errorf("write setup-run file: %s: %w", scriptname, err)
			}
			defer os.Remove(scriptname)

			cfg := BoxConfig{
				Name:        fmt.Sprintf("%s-%s", boxname, strconv.FormatInt(time.Now().UnixNano(), 36)),
				SourceImage: box.ImageName(config.LocalImage),
				ExtraArgs: []string{
					"--user", fmt.Sprintf("%s:%s", usr.Uid, usr.Gid),
					"--userns", "keep-id",
					"--annotation", "run.oci.keep_original_groups=1",
					"--workdir", currentdir,
					"--rm",
				},
				Command:     "/bin/sh /usr/bin/entrypoint.sh",
				Volumes:     config.Run.Volumes,
				Environment: config.Run.Environment,
			}

			cfg.Volumes = append(cfg.Volumes,
				scriptname+":/usr/bin/entrypoint.sh:ro",
				fmt.Sprintf("%s:%s:rw,U", config.DataDir, usr.HomeDir),
				fmt.Sprintf("%s:%s:rw,U", currentdir, currentdir),
			)

			mountedpaths := make([]string, 0, len(cfg.Volumes))
			for _, volume := range cfg.Volumes {
				parts := strings.Split(volume, ":")
				if len(parts) < 2 {
					continue
				}

				mountedpaths = append(mountedpaths, parts[1])
			}

			if len(mountedpaths) > 0 {
				cfg.ExtraArgs = append(cfg.ExtraArgs,
					"--env", "HOME="+usr.HomeDir, // Override HOME just in case
					"--env", "MOUNTED_PATHS="+strings.Join(mountedpaths, ":"),
				)
			}

			runcmd := box.CraftRun(cfg)
			PrintCommand(runcmd...)
			if config.Run.AskBeforeRunning && !AskConfirmation("Do you want to execute?") {
				return nil
			}

			defer func() {
				err = box.CleanupHomeDir(config.DataDir, mountedpaths)
				if err != nil {
					fmt.Println(err)
				}
			}()

			err = box.Exec(runcmd)
			if err != nil {
				return fmt.Errorf("run: %w", err)
			}

			return nil
		},
	}
}
