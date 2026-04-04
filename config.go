package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mdouchement/upathex"
	"go.yaml.in/yaml/v4"
)

type (
	CreateConfig struct {
		AskBeforeRunning bool   `yaml:"ask_before_running"`
		Image            string `yaml:"image"`
		ShellScript      string `yaml:"shell_script"`
	}

	UpdateConfig struct {
		AskBeforeRunning bool   `yaml:"ask_before_running"`
		ShellScript      string `yaml:"shell_script"`
	}

	RunConfig struct {
		AskBeforeRunning bool     `yaml:"ask_before_running"`
		Command          string   `yaml:"command"`
		Volumes          []string `yaml:"volumes"`
	}

	Config struct {
		Description string       `yaml:"description"`
		DataDir     string       `yaml:"data_dir"`
		LocalImage  string       `yaml:"local_image"`
		Create      CreateConfig `yaml:"create"`
		Update      UpdateConfig `yaml:"update"`
		Run         RunConfig    `yaml:"run"`
	}
)

func Load(filename string) (Config, error) {
	var c Config

	payload, err := os.ReadFile(filename)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(payload, &c)
	if err != nil {
		return c, err
	}

	if c.LocalImage == "" {
		c.LocalImage = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}

	for i, v := range c.Run.Volumes {
		parts := strings.Split(v, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return c, fmt.Errorf("volume %s: invalid format", v)
		}

		parts[0], err = upathex.ExpandTilde(parts[0])
		if err != nil {
			return c, fmt.Errorf("volume %s: %w", v, err)
		}

		parts[1], err = upathex.ExpandTilde(parts[1])
		if err != nil {
			return c, fmt.Errorf("volume %s: %w", v, err)
		}

		c.Run.Volumes[i] = strings.Join(parts, ":")
	}

	return c, nil
}
