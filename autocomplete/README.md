# Genesys Shell Completions

This directory contains shell completion scripts for Genesys that provide tab completion for commands, options, and file paths.

## Supported Shells

- **Bash** (`genesys-completion.bash`)
- **Zsh** (`genesys-completion.zsh`)
- **Fish** (`genesys-completion.fish`)

## Installation

### Automatic Installation

The completions are automatically installed when you run:

```bash
make install
```

And removed with:

```bash
make uninstall
```

### Manual Installation

#### Bash

```bash
# Copy to bash completions directory
sudo cp genesys-completion.bash /usr/share/bash-completion/completions/genesys

# Or source directly in your ~/.bashrc
echo "source /path/to/genesys-completion.bash" >> ~/.bashrc
```

#### Zsh

```bash
# Copy to zsh completions directory
sudo cp genesys-completion.zsh /usr/share/zsh/site-functions/_genesys

# Make sure completions are enabled in ~/.zshrc
echo "autoload -U compinit && compinit" >> ~/.zshrc
```

#### Fish

```bash
# Copy to fish completions directory
cp genesys-completion.fish ~/.config/fish/completions/genesys.fish

# Or system-wide
sudo cp genesys-completion.fish /usr/share/fish/vendor_completions.d/genesys.fish
```

## Features

The completion scripts provide:

1. **Command completion**: All main commands (interact, execute, discover, config, version)
2. **Subcommand completion**: Config subcommands (setup, list, show, validate)
3. **Provider completion**: AWS, GCP, Azure, Tencent
4. **File completion**: YAML and TOML files for execute command
5. **Flag completion**: All command-specific flags and global flags
6. **Resource type completion**: For discover command

## Examples

```bash
# Complete commands
genesys <TAB>
# Shows: interact execute discover config version help

# Complete config files
genesys execute <TAB>
# Shows: *.yaml *.yml *.toml files

# Complete providers
genesys discover <TAB>
# Shows: aws gcp azure tencent

# Complete flags
genesys execute myconfig.yaml --<TAB>
# Shows: --dry-run --force --output --parallel

# Complete config subcommands
genesys config <TAB>
# Shows: setup list show validate
```

## Testing Completions

After installation, test the completions:

```bash
# Bash - reload completions
source ~/.bashrc

# Zsh - reload completions
source ~/.zshrc

# Fish - automatically loaded

# Test
genesys <TAB>
```

## Customization

You can customize the completions by editing the respective shell script. Each script contains:

- Command definitions
- Flag definitions
- Custom completion logic

## Troubleshooting

### Bash
- Ensure bash-completion package is installed: `sudo apt-get install bash-completion`
- Check if completions are loaded: `complete -p | grep genesys`

### Zsh
- Ensure compinit is called in your ~/.zshrc
- Check completion directories: `echo $fpath`

### Fish
- Check if file exists: `ls ~/.config/fish/completions/`
- Reload fish config: `source ~/.config/fish/config.fish`