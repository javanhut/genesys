# Zsh Setup for Genesys Completions

Add the following to your `~/.zshrc`:

```bash
# Enable Zsh completions
autoload -Uz compinit
compinit

# Add the completion directory to fpath if not already there
fpath=(/usr/local/share/zsh/site-functions $fpath)
```

Then reload your shell:

```bash
# Either restart your terminal or run:
source ~/.zshrc

# Or just reload completions:
rm -f ~/.zcompdump; compinit
```

## Testing

After setup, test the completions:

```bash
genesys <TAB>
# Should show: config discover execute help interact version

genesys execute <TAB>
# Should show .toml files

genesys discover <TAB>
# Should show: aws azure gcp tencent
```

## Troubleshooting

If completions don't work:

1. Check if the completion file exists:
   ```bash
   ls -la /usr/local/share/zsh/site-functions/_genesys
   ```

2. Verify fpath includes the directory:
   ```bash
   echo $fpath | tr ' ' '\n' | grep site-functions
   ```

3. Force reload completions:
   ```bash
   rm -f ~/.zcompdump
   compinit -u
   ```

4. Check for errors:
   ```bash
   compinit -D  # Debug mode
   ```