# h2status

Lightweight status bar for sway/i3, written in Go.

## Philosophy

Show the bare minimum. Most widgets appear only when something needs attention — low battery, high CPU load, weak WiFi signal. If everything is fine, the bar stays clean.

## Features

- **Clock** — locale-aware time/date, click to cycle formats
- **Battery** — multi-battery support, click to toggle % / remaining time
- **Bluetooth** — connected audio devices (A2DP)
- **CPU** — load with per-core hotspot info
- **RAM** — memory usage
- **Disk** — free space for multiple paths
- **WiFi** — signal strength, known/unknown network detection
- **Network** — warns when no internet (no default route)
- **VPN** — tun/wg/tap interfaces, full/split tunnel indicator
- **Temperature** — CPU temp with auto-detection

All widgets are threshold-based: they only appear when something needs attention.

## Installation

### Binary

Download from [Releases](../../releases):

```sh
curl -L https://github.com/USER/h2status/releases/latest/download/h2status-linux-amd64 -o h2status
chmod +x h2status
sudo mv h2status /usr/local/bin/
```

### From source

```sh
go install github.com/USER/h2status@latest
```

Or:

```sh
git clone https://github.com/USER/h2status
cd h2status
go build
```

## Configuration

Copy example config:

```sh
mkdir -p ~/.config/h2status
cp config.example.toml ~/.config/h2status/config.toml
```

See [config.example.toml](config.example.toml) for all options.

## Usage

### Sway

In `~/.config/sway/config`:

```
bar {
    status_command h2status
}
```

### i3

In `~/.config/i3/config`:

```
bar {
    status_command h2status
}
```

## Temperature sensors

To list available temperature sensors:

```sh
h2status --print-sensors
```

## License

MIT
