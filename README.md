# sdbx

It spawns containers for any terminal application like opencode leveraging Podman.\
Only the specified volumes are mounted so your data won't be leaked when an LLM ask you to read `/etc/*`.\
You can find [opencode box file](opencode.yml) as example file.\
You can provide `ask_before_running` boolean in each box configuration file so you can see what Podman command will be executed.

## Installation

Grab a binary from the release section or install it through `go install github.com/mdouchement/sdbx@latest` command.

## Default folders

- `$XDG_CONFIG_HOME/sdbx` or `~/.config/sdbx` is where the boxes (YAML files) are stored
- `$XDG_DATA_HOME/sdbx` or `~/.local/share/sdbx` is where the container's home are stored
- `$XDG_CACHE_HOME/sdbx` or `~/.cache/sdbx` is where the temp files are stored

## Usage

A box `<name>` refers to `<name>.yml` loaded from `$XDG_CONFIG_HOME/sdbx/<name>.yml` or `~/.config/sdbx/<name>.yml` as fallback.

### sbgx list

List the available boxes.

### sbgx create <name>

Create a new box image based on `<name>.yml` in sdbx config directory.

You can recreate a box image by calling this action again.

sdbx tries to keep things as clean as possible but it may leaves some olf images. You can perform some cleanup by running `podman image prune`.

### sbgx update <name>

Update the box image based on `<name>.yml` file in sdbx config directory.

### sbgx run <name>

Run a box based on `<name>.yml` file in sdbx config directory.\
It will mount the current terminal folder as the container workdir.

### sbgx delete <name>

Deletes data used by the given box.\
It'll ask you confirmation for each step.
