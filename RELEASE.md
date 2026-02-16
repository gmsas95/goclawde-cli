# Releasing Myrai

This document describes how to release new versions of Myrai.

## Prerequisites

1. GitHub CLI (`gh`) installed and authenticated
2. npm account with publish permissions
3. Go 1.21+ installed

## Release Process

### 1. Update Version

Update the version in:
- `Makefile` (VERSION variable)
- `npm/package.json`

### 2. Build Release Binaries

```bash
make release VERSION=0.2.0
```

This creates binaries for:
- Linux (amd64, arm64)
- macOS (amd64, arm64) 
- Windows (amd64, arm64)

### 3. Create GitHub Release

```bash
make release-gh VERSION=0.2.0
```

Or manually:

```bash
gh release create v0.2.0 ./bin/release/* \
  --title "v0.2.0" \
  --notes "Release v0.2.0

## What's Changed
- [List changes here]

## Installation
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash"
```

### 4. Publish to npm

```bash
cd npm
npm version 0.2.0
npm publish --access public
```

Or use:

```bash
make publish-npm
```

## Installation Methods

### curl (Recommended)

```bash
# Latest version
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash

# Specific version
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash -s -- --version 0.2.0

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash -s -- --dir ~/.local/bin
```

### npm

```bash
# Global install
npm install -g myrai

# Use with npx (no install)
npx myrai --help
```

### Manual

```bash
# Download binary
curl -LO https://github.com/gmsas95/goclawde-cli/releases/download/v0.2.0/myrai-linux-amd64
chmod +x myrai-linux-amd64
sudo mv myrai-linux-amd64 /usr/local/bin/myrai

# Or build from source
git clone https://github.com/gmsas95/goclawde-cli.git
cd goclawde-cli
make build
sudo make install
```

## Version Checking

Users can check their version:

```bash
myrai version
```

## Rollback

If a release has issues:

```bash
# Delete release
gh release delete v0.2.0 --yes

# Delete tag
git push --delete origin v0.2.0

# Deprecate npm version
npm deprecate myrai@0.2.0 "This version has issues, please upgrade to 0.2.1"
```
