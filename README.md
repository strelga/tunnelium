# Tunnelium

VPN service manager for managing [gost](https://github.com/ginuerzh/gost) VPN and proxy services via Docker Compose.

## Installation

> **Important:** The binary must be owned by your user (not root) for `tunnelium self-update` to work. All install examples below use `-o $(whoami)` so that `sudo install` sets the owner to the current user.

### One-line install (Linux, system-wide)

```bash
curl -fSL -o /tmp/tunnelium \
  https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-linux-amd64 \
  && sudo install -m 0755 -o $(whoami) /tmp/tunnelium /usr/local/bin/tunnelium \
  && rm /tmp/tunnelium
```

Uses `sudo install` to place the binary in `/usr/local/bin` with correct permissions, owned by the current user.

### One-line install (Linux, no sudo)

```bash
mkdir -p ~/.local/bin \
  && curl -fSL -o ~/.local/bin/tunnelium \
     https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-linux-amd64 \
  && chmod +x ~/.local/bin/tunnelium
```

Make sure `~/.local/bin` is in your `$PATH`. On most modern Linux distros it already is. If not, add to your shell profile:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### macOS (Intel)

```bash
curl -fSL -o /tmp/tunnelium \
  https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-darwin-amd64 \
  && sudo install -m 0755 -o $(whoami) /tmp/tunnelium /usr/local/bin/tunnelium \
  && rm /tmp/tunnelium
```

### macOS (Apple Silicon)

```bash
curl -fSL -o /tmp/tunnelium \
  https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-darwin-arm64 \
  && sudo install -m 0755 -o $(whoami) /tmp/tunnelium /usr/local/bin/tunnelium \
  && rm /tmp/tunnelium
```

### Specific version

Replace `latest/download` with `download/v<VERSION>` to pin a version:

```bash
curl -fSL -o /tmp/tunnelium \
  https://github.com/strelga/tunnelium/releases/download/v0.0.2/tunnelium-linux-amd64 \
  && sudo install -m 0755 -o $(whoami) /tmp/tunnelium /usr/local/bin/tunnelium \
  && rm /tmp/tunnelium
```

## Usage

```
tunnelium self-update          # Update to the latest version from GitHub
tunnelium service add          # Add a new service (interactive or via flags)
tunnelium service list         # List configured services
tunnelium service remove       # Remove a service
tunnelium service start        # Start a service
tunnelium service stop         # Stop a service
```

### Add a gost client

```bash
tunnelium service add \
  --type gost \
  --name my-tunnel \
  --role client \
  --next-hop-host example.com \
  --socks-port 1080
```

### Add a gost server

```bash
tunnelium service add \
  --type gost \
  --name my-server \
  --role server \
  --port 443
```

## License

[MIT](LICENSE)
