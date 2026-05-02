package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Box struct {
	binary            string
	hostname          string
	boxname           string
	user              *user.User
	configdir         string
	cachedir          string
	datadir           string
	distroboxInitFile string
}

type BoxConfig struct {
	Name             string
	SourceImage      string
	DestinationImage string
	ExtraArgs        []string
	Command          string
	Volumes          []string
	Environment      []string
}

func New(binary, boxname string) (*Box, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("user: %w", err)
	}

	//

	cachedir := os.Getenv("XDG_CACHE_HOME")
	if cachedir == "" {
		cachedir = filepath.Join(u.HomeDir, ".cache")
	}
	cachedir = filepath.Join(cachedir, "sdbx")

	err = os.MkdirAll(cachedir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("mkdir_all: %w", err)
	}

	//

	datadir := os.Getenv("XDG_DATA_HOME")
	if datadir == "" {
		datadir = filepath.Join(u.HomeDir, ".local", "share")
	}
	datadir = filepath.Join(datadir, "sdbx")

	err = os.MkdirAll(datadir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("mkdir_all: %w", err)
	}

	//

	configdir := os.Getenv("XDG_CONFIG_HOME")
	if configdir == "" {
		configdir = filepath.Join(u.HomeDir, ".config")
	}
	configdir = filepath.Join(configdir, "sdbx")

	err = os.MkdirAll(configdir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("mkdir_all: %w", err)
	}

	//

	box := &Box{
		binary:    binary,
		hostname:  hostname,
		boxname:   boxname,
		user:      u,
		configdir: configdir,
		cachedir:  cachedir,
		datadir:   datadir,
	}

	return box, nil
}

func (b *Box) HostUser() *user.User {
	return b.user
}

func (b *Box) HostConfigDir() string {
	return b.configdir
}

func (b *Box) HostCacheDir() string {
	return b.cachedir
}

func (b *Box) HostDataDir() string {
	return b.datadir
}

func (b *Box) ConfigFilename() string {
	return filepath.Join(b.configdir, b.boxname+".yml")
}

func (b *Box) ImageName(name string) string {
	return fmt.Sprintf("localhost/sdbx-%s", name)
}

func (b *Box) DynamicBoxname() string {
	return fmt.Sprintf("%s-%s", b.boxname, strconv.FormatInt(time.Now().UnixNano(), 36))
}

func (b *Box) CraftPull(image string) []string {
	return []string{b.binary, "pull", image}
}

func (b *Box) CraftRemove(cfg BoxConfig) []string {
	return []string{b.binary, "rm", cfg.Name}
}

func (b *Box) CraftRemoveImage(cfg BoxConfig) []string {
	return []string{b.binary, "rmi", cfg.DestinationImage}
}

func (b *Box) CraftCommit(cfg BoxConfig) []string {
	return []string{b.binary, "commit", cfg.Name, cfg.DestinationImage}
}

func (b *Box) CraftRun(cfg BoxConfig) []string {
	cmd := []string{
		b.binary, "run",
		"--hostname", fmt.Sprintf("%s.%s", cfg.Name, b.hostname),
		"--name", cfg.Name,
		// "--network", "host", bridge by default
		"--label", "manager=sdbx",
		"--env", "container=" + b.binary,
		"--env", "TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo",
	}

	for _, env := range cfg.Environment {
		cmd = append(cmd, "--env", env)
	}

	for _, volume := range cfg.Volumes {
		cmd = append(cmd, "--volume", volume)
	}

	cmd = append(cmd, cfg.ExtraArgs...)

	cmd = append(cmd,
		"--interactive",
		"--tty",
		cfg.SourceImage,
	)
	cmd = append(cmd, strings.Split(cfg.Command, " ")...)

	return cmd
}

func (*Box) Exec(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return fmt.Errorf("exec command: %w", err)
	}

	return nil
}

func (b *Box) CleanupHomeDir(boxdir string, boxpaths []string) error {
	root, err := os.OpenRoot(boxdir)
	if err != nil {
		return fmt.Errorf("cleanup: root-home: %w", err)
	}

	fmt.Println("Cleaning up empty directories in", boxdir)

	var errs []error
	for _, path := range boxpaths {
		if path == b.user.HomeDir || !strings.HasPrefix(path, b.user.HomeDir) {
			// Not a path in the box HOME folder.
			continue
		}

		path = strings.TrimPrefix(path, b.user.HomeDir) // Remove from path PoV the home prefix so we can use in os.Root
		path = strings.TrimPrefix(path, "/")            // Make it relative

		err = b.cleanupHomeDirRecursive(root, path)
		if err != nil {
			errs = append(errs, err)
		}
	}

	//

	err = nil
	for _, e := range errs {
		if err == nil {
			err = e
			continue
		}

		err = fmt.Errorf("%w; %w", err, e)
	}

	return err
}

func (b *Box) cleanupHomeDirRecursive(root *os.Root, relpath string) error {
	if relpath == "" || relpath == "." {
		return nil
	}

	fi, err := root.Stat(relpath)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return nil
	}

	f, err := root.Open(relpath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == nil {
		// Folder contains files, keep it
		return nil
	}
	if err != io.EOF {
		return err
	}

	// EOF is returned when directory is empty

	fmt.Println("  removing empty directory", relpath)
	err = root.Remove(relpath)
	if err != nil {
		return err
	}

	return b.cleanupHomeDirRecursive(root, filepath.Dir(relpath))
}
