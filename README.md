# ExamplePlugin

The official example plugin for [PocketMine-go](https://github.com/PMGopher/pocketmine-go). It is
small on purpose and every line is commented: copy it to start your own plugin.

It shows the parts almost every plugin needs:

| Feature | Where |
|---|---|
| Registering the plugin and embedding its files | `init()` in [example.go](example.go) |
| `plugin.toml`: name, version, API, commands, permissions | [plugin.toml](plugin.toml) |
| A config file with defaults, reloadable in game | [resources/config.toml](resources/config.toml), `SaveDefaultConfig`, `ReloadConfig` |
| An event listener: a welcome title when a player joins | `joinListener.OnJoin` |
| A command with an alias and a sub-command: `/example`, `/ex`, `/example reload` | `OnCommand` |
| A repeating task: a tip broadcast every few minutes | `scheduleTip` |
| A test that starts a real server and loads the plugin | [example_test.go](example_test.go) |

## Adding it to your server

Needs PocketMine-go v1.1.0 or newer (the first version with TOML config files). Plugins are
compiled into the server. In your clone of
[pocketmine-go](https://github.com/PMGopher/pocketmine-go):

```bash
go get github.com/PMGopher/example@v1.1.0
```

Then import it in `cmd/pocketmine-go/plugins.go`:

```go
package main

import _ "github.com/PMGopher/example"
```

Rebuild and start the server:

```bash
go build -o pocketmine-go ./cmd/pocketmine-go
./pocketmine-go
```

The console shows:

```text
[Server thread/INFO]: Loading ExamplePlugin v1.1.0
[Server thread/INFO]: Enabling ExamplePlugin v1.1.0
[Server thread/INFO]: [ExamplePlugin] ExamplePlugin enabled! Try /example in game.
```

## Configuration

On first start the plugin writes `plugin_data/ExamplePlugin/config.toml`:

```toml
welcome-title = "§aWelcome, {player}!"
welcome-subtitle = "§7Running on PocketMine-go"
tip-interval = 5          # minutes, 0 turns the tip off
tip = "§eTip: type /example to say hello!"
```

Edit it and run `/example reload`, no restart needed.

## Commands

| Command | Permission | Default | What it does |
|---|---|---|---|
| `/example` (`/ex`) | `exampleplugin.command.example` | everyone | Says hello and shows your position and the server's TPS |
| `/example reload` | `exampleplugin.command.example` | everyone | Reloads `config.toml` |

## Making your own plugin from this one

1. Copy this repository and change the module path in `go.mod`
   (`github.com/you/yourplugin`) and the package name.
2. Rename the plugin in `plugin.toml` (`name`, `main`, commands and permissions).
3. Write your code, then add it to your server as shown above.

Converting a plugin written for PocketMine-MP? **[AGENTS.md](AGENTS.md)** is a step-by-step
conversion guide, for people and AI agents alike.

## Developing

Clone this repository **next to** the server, so `go.work` finds it:

```text
workspace/
├── pocketmine-go/   git clone https://github.com/PMGopher/pocketmine-go.git
└── example/         git clone https://github.com/PMGopher/example.git
```

```bash
cd example
go vet ./...
go test ./...   # starts a real server in a temporary folder and loads the plugin
```

`go.mod` has no `require` lines: the plugin only imports the server, and the server provides those
packages itself when it builds the plugin. `go.work` points Go at the server clone while you work on
the plugin by itself; the server ignores it.

## Versions

Releases are git tags (`v1.0.0`, `v1.0.1`, ...), matching the `version` in `plugin.toml`. Server
owners pin one with `go get github.com/PMGopher/example@v1.1.0`; `@latest` picks the newest tag.

## Credits
- **[MEMOxiiii](https://github.com/MEMOxiiii)**: developer of PocketMine-go.
