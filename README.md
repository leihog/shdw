# Shadow (`shdw`)

A local-first, zero-cloud encrypted secrets manager for developers.

- **AES-256-GCM** encryption with **Argon2id** key derivation
- **Tree-based organisation** — paths work like a filesystem
- **OS keychain integration** — unlock once per session, stays unlocked
- **No account, no cloud, no telemetry** — your secrets never leave your machine

---

## Why shdw?
 
Most secrets managers fall into two camps: **cloud-backed tools** that require
an account and sync your secrets to someone else's server, or **enterprise tools**
that are overkill for a single developer.
 
| | shdw | Doppler / Infisical | 1Password CLI | pass |
|---|---|---|---|---|
| Local-only | ✓ | ✗ | ✗ | ✓ |
| No account required | ✓ | ✗ | ✗ | ✓ |
| `run` / subprocess injection | ✓ | ✓ | ✓ | ✗ |
| Tree-based namespaces | ✓ | ✗ | ✗ | ✗ |
| Argon2id key derivation | ✓ | — | — | ✗ |
| Single binary | ✓ | ✗ | ✗ | ✗ |
| OS keychain integration | ✓ | — | ✓ | ✗ |
 
**Compared to Doppler/Infisical:** Those are excellent tools for teams — secrets
are centralised, auditable, and shareable. That's also their downside for solo
developers: you depend on an external service, your secrets leave your machine,
and there's a monthly cost. shdw works offline, forever.
 
**Compared to 1Password CLI:** `op run` is genuinely good, but it requires a
1Password subscription and is tightly coupled to the 1Password ecosystem.
shdw has no vendor dependency.
 
**Compared to `pass`:** The closest in philosophy — local, encrypted, no account.
But `pass` requires GPG setup (notoriously painful), has no native subprocess
injection, and its "namespacing" is just a folder of files. shdw is
purpose-built for the developer workflow.
 
**The target user** is a solo developer or small team who wants secrets off disk
and out of `.env` files, without signing up for anything or sending secrets over
the network.
 
---

## Install

```bash
git clone https://github.com/leihog/shdw
cd shdw
go mod tidy
make install              # installs to /usr/local/bin
make install PREFIX=~/bin # installs to ~/bin
```

### Cross-compile for Linux

```bash
make build-linux          # produces shdw-linux-amd64
make build-linux-arm      # produces shdw-linux-arm64
make build-all            # builds all targets at once

# Then copy to a server
scp shdw-linux-amd64 user@server:/usr/local/bin/shdw
```

---

## How paths work

The vault is organised as a tree — think of it like a filesystem with files
(keys) and folders (namespaces). A path uniquely identifies a node, and a
node is either one or the other, never both.

```
/                          ← root
├── token                  ← key:  shdw set token abc123
├── discord/               ← namespace
│   ├── api_key            ← key:  shdw set discord/api_key abc123
│   └── prod/              ← namespace
│       └── token          ← key:  shdw set discord/prod/token abc123
```

Bare keys with no path separator are stored at the root. When secrets are
injected as environment variables, only the **key name** (last segment) is
used, uppercased: `discord/prod/token` → `TOKEN`.

---

## Commands

### set — store a secret

```bash
shdw set token abc123                   # store at root
shdw set discord/api_key abc123         # store in namespace
shdw set discord/prod/token abc123      # deeply nested
shdw set discord/api_key newval --force # overwrite existing value
```

**Use `-i` to enter the value interactively (recommended for sensitive values):**

```bash
shdw set discord/api_key -i
# Value for 'discord/api_key': (hidden input, never touches shell history)
```

Passing a value directly on the command line is convenient but it will appear
in your shell history. Use `-i` when storing anything sensitive.

### get — retrieve a secret

```bash
shdw get discord/api_key
export TOKEN=$(shdw get discord/api_key)
```

### copy — copy a secret to the clipboard

```bash
shdw copy discord/api_key   # value goes to clipboard, never printed
shdw cp discord/api_key     # alias
```

### list — browse the vault tree

```bash
shdw list                   # show full tree from root
shdw list discord           # show subtree at discord/
shdw list discord/prod      # show subtree at discord/prod/
```

### run — inject secrets into a subprocess

Secrets exist only for the duration of the subprocess — they never persist
in your shell environment or history.

```bash
shdw run discord -- node app.js                  # all keys in discord/
shdw run discord/api_key -- node app.js          # single key
shdw run token discord -- node app.js            # root key + namespace
shdw run discord discord/prod -- ./deploy.sh     # layered, prod overrides
shdw run discord --add-path-prefix -- node app.js  # injects DISCORD_API_KEY
```

Multiple paths are resolved in order — later entries override earlier ones
on env var name collision.

### export — dump secrets to .env format

```bash
shdw export discord                       # to stdout
shdw export discord discord/prod          # merged, prod overrides discord
shdw export discord discord/prod -o .env  # write to file
shdw export discord --add-path-prefix     # DISCORD_API_KEY style names
```

### import — load secrets from a .env file

```bash
shdw import .env                          # imports to root
shdw import .env --namespace discord      # imports into discord/
shdw import .env -n discord/prod --force  # overwrite existing
```

### rename — move a key or namespace

```bash
shdw rename token root_token             # rename a key
shdw rename discord services/discord    # move entire namespace
shdw mv discord/api_key discord/prod/api_key  # alias
```

### delete — remove a key or namespace

```bash
shdw delete discord/api_key
shdw rm discord           # removes namespace and everything inside it
```

### info — vault status and stats

```bash
shdw info
```

Shows vault path, file size, last modified. If the vault is unlocked, also
shows namespace and key counts with a full tree view.

### unlock / lock — manage the cached password

```bash
shdw unlock   # prompt for password and cache it in the OS keychain
shdw lock     # clear the cached password
```

Any command that needs the vault will prompt automatically if locked.
`unlock` is just a convenience for pre-unlocking without running another command.

---

## Layering pattern

A common pattern is to store shared config in a base namespace and
environment-specific overrides in a child namespace:

```bash
shdw set discord/bot_name MyBot
shdw set discord/prod/api_key sk_live_...
shdw set discord/staging/api_key sk_test_...

shdw run discord discord/prod -- node app.js
# → BOT_NAME=MyBot, API_KEY=sk_live_...

shdw run discord discord/staging -- node app.js
# → BOT_NAME=MyBot, API_KEY=sk_test_...
```

---

## Vault location

- **macOS**: `~/Library/Application Support/shdw/vault`
- **Linux**: `~/.config/shdw/vault`
- **Windows**: `%AppData%\shdw\vault`

The vault is a single encrypted file, safe to back up or sync manually.

---

## Security

**Encryption:** AES-256-GCM with a key derived via Argon2id
(`time=1, memory=64MB, threads=4`). Argon2id is memory-hard, making
brute-force attacks expensive even with GPUs or ASICs.

**Vault format:** The first byte is a version identifier, allowing future
encryption scheme upgrades without breaking existing vaults.

**Master password:** Never written to disk. Cached in the OS keychain
(macOS Keychain, libsecret on Linux, Windows Credential Manager) so you
only type it once per session. Use `shdw lock` to clear it.

**Shell history:** Passing a secret value as a CLI argument (`shdw set path value`)
will appear in your shell history. Use `shdw set path -i` for sensitive values
to keep them out of your history entirely.

**Subprocess isolation:** `shdw run` injects secrets into the subprocess
environment only. They are never written to disk and disappear when the
process exits.

**File permissions:** The vault file is created with `0600` permissions
(owner read/write only).
