# Tunnelium

VPN service manager for managing [gost](https://github.com/ginuerzh/gost) VPN and proxy services via Docker Compose.

## Installation

### One-line install (Linux)

```bash
curl -fSL -o /usr/local/bin/tunnelium \
  https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-linux-amd64 \
  && chmod +x /usr/local/bin/tunnelium
```

### macOS (Intel)

```bash
curl -fSL -o /usr/local/bin/tunnelium \
  https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-darwin-amd64 \
  && chmod +x /usr/local/bin/tunnelium
```

### macOS (Apple Silicon)

```bash
curl -fSL -o /usr/local/bin/tunnelium \
  https://github.com/strelga/tunnelium/releases/latest/download/tunnelium-darwin-arm64 \
  && chmod +x /usr/local/bin/tunnelium
```

### Specific version

Replace `latest/download` with `download/v<VERSION>` to pin a version:

```bash
curl -fSL -o /usr/local/bin/tunnelium \
  https://github.com/strelga/tunnelium/releases/download/v0.0.2/tunnelium-linux-amd64 \
  && chmod +x /usr/local/bin/tunnelium
```

## Usage

```
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
