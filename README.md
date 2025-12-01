# igloc

Scan your local machine and discover files ignored by `.gitignore`.

**igloc** helps you find secrets and local configurations that won't be committed to version control. Useful for auditing your projects or setting up a new machine.

## Installation

```bash
go install github.com/O6lvl4/igloc/cmd/igloc@latest
```

Or build from source:

```bash
git clone https://github.com/O6lvl4/igloc.git
cd igloc
make build
```

## Usage

### Scan for secrets

```bash
# Scan current directory
igloc scan

# Scan specific directory
igloc scan ~/projects/my-app

# Show only .env files
igloc scan --category env

# Show all ignored files (not just secrets)
igloc scan --all

# Include dependency directories (node_modules, etc.)
igloc scan --include-deps

# Recursively scan all git repos
igloc scan -r ~/projects
```

### Sync patterns from GitHub

```bash
# Fetch latest patterns from github/gitignore
igloc sync

# List supported languages
igloc sync --list
```

## Example Output

```
📂 /Users/you/projects/my-app

   🔑 ENV (3)
      .env 🔐
      .env.local 🔐
      config/.env.production 🔐

   Total: 3 files (🔐 3 secrets)
```

## How It Works

1. **Scan**: Uses `git status --ignored` to find files ignored by `.gitignore`
2. **Categorize**: Groups files by type (env, key, config, build, cache, ide)
3. **Filter**: By default, excludes dependency directories and shows only likely secrets

### Dependency Exclusion

By default, igloc excludes common dependency directories:

| Language | Directories |
|----------|-------------|
| Node.js | `node_modules/` |
| Python | `.venv/`, `venv/`, `__pycache__/`, `.tox/` |
| Ruby | `vendor/bundle/`, `.bundle/` |
| Go | `vendor/`, `pkg/mod/` |
| Rust | `target/` |
| Java | `.gradle/`, `.m2/`, `build/` |
| .NET | `packages/`, `bin/`, `obj/` |
| iOS | `Pods/`, `Carthage/` |
| And more... | |

Run `igloc sync` to fetch the latest patterns from [github/gitignore](https://github.com/github/gitignore).

## Configuration

Patterns are stored in `~/.config/igloc/patterns.yaml` after running `igloc sync`.

## Use Cases

- **Audit secrets**: Find all `.env` files hiding in your projects
- **New machine setup**: List required secret files when cloning repos
- **Security review**: Discover credentials that might be accidentally committed

## License

MIT
