# AGENTS.md: converting PocketMine-MP plugins to PocketMine-go

This guide is for anyone, human or AI agent, who wants to take a plugin written in PHP for
**PocketMine-MP** and turn it into a Go plugin for
**[PocketMine-go](https://github.com/PMGopher/pocketmine-go)**. This repository is the reference
result: a working plugin that uses every piece described below. Read [example.go](example.go)
alongside this file.

PocketMine-go has the same plugin API as PocketMine-MP, in Go: the same manifest keys (in
`plugin.toml`), the same events, commands, permissions, configs and scheduler, and the same class and method names in Go
style (`getServer()` → `GetServer()`). Most conversions are a straight translation, file by file.

---

## 1. The big differences

| PocketMine-MP | PocketMine-go |
|---|---|
| A plugin is a folder or `.phar` dropped in `plugins/` | A plugin is a **Go module**, compiled into the server (`go get` + one import line) |
| `plugin.yml`, `config.yml` and other YAML files | **TOML**: `plugin.toml`, `config.toml` (see §2 and §7) |
| `main: vendor\plugin\Main` names the class to load | `init()` calls `plugin.RegisterGoPlugin`, which creates your main type; `main` in `plugin.toml` is only informational |
| `resources/` is read from the plugin folder | `plugin.toml` and `resources/` are embedded with `//go:embed` |
| Classes and inheritance (`extends PluginBase`) | Structs and embedding (`struct{ plugin.PluginBase }`) |
| Exceptions | `error` return values; a panic is a bug |
| `array` | slices (`[]T`) and maps (`map[K]V`) |
| `null` | `nil`, or a second `ok bool` return (`w, ok := GetWorldByName(...)`) |
| `$x instanceof Player` | type assertion: `p, ok := x.(*player.Player)` |
| Virions (libraries bundled into the phar) | Normal Go modules in `go.mod` |
| `AsyncTask` on worker threads | `scheduler.AsyncTask` on worker goroutines, same rules (see §8) |

## 2. Step by step

1. **Read the whole PHP plugin first**: `plugin.yml`, every class under `src/`, `resources/`.
   List its events, commands, permissions, config keys, tasks and any virions.
2. **Create the module**: copy this repository, then set your own module path in `go.mod`
   (`module github.com/you/yourplugin`) and rename the package in the `.go` files. Keep `go.work`
   for developing next to a server clone. `go.mod` needs no `require` for the server; add
   `require` lines only for other modules your plugin uses (its former virions, ...).
3. **Turn `plugin.yml` into `plugin.toml`** with the same keys: `name`, `version`, `api`,
   `depend`, `softdepend`, `loadbefore`, `load`, `commands` and `permissions` work the same.
   Change `main` to something like `yourplugin.Main` (informational). `src-namespace-prefix`
   isn't used. Keep the permissions in the same order: a `default` carries forward to the
   permissions declared after it, as in PocketMine-MP. See the side-by-side example below.
4. **Copy `resources/`** (language files, ...), turn the default `config.yml` into
   `config.toml` with the same keys (§7), and embed them: `//go:embed plugin.toml resources`.
5. **Translate the main class** into a `Main` struct embedding `plugin.PluginBase` (§3).
6. **Translate each listener, command and task** (§4 to §8), keeping the plugin's behaviour.
7. **Replace virions and PHP-only tricks** (§10).
8. **Write a test** like [example_test.go](example_test.go): it starts a real server in a
   temporary folder, checks that the plugin loads, and runs its commands.
9. **Run `go vet ./...` and `go test ./...`** until both are clean.
10. **Tag a release** matching `version` in `plugin.toml` (`git tag v1.0.0`).
11. **Add it to a server** (see [README.md](README.md#adding-it-to-your-server)) and try it in game.

### `plugin.yml` → `plugin.toml`

```yaml
name: HomePlugin
version: 2.1.0
main: alex\home\Main
api: [5.0.0]
depend: [EconomyAPI]
commands:
  home:
    description: Teleports you home
    usage: /home [name]
    aliases: [h]
    permission: home.command.home
permissions:
  home.command.home:
    default: true
  home.command.sethome:
    description: Set your home
```

```toml
name = "HomePlugin"
version = "2.1.0"
main = "home.Main"
api = ["5.0.0"]
depend = ["EconomyAPI"]

[commands.home]
description = "Teleports you home"
usage = "/home [name]"
aliases = ["h"]
permission = "home.command.home"

[permissions."home.command.home"]
default = true

[permissions."home.command.sethome"]
description = "Set your home"
```

## 3. The main class

```php
class Main extends PluginBase{
    protected function onLoad() : void{ ... }
    protected function onEnable() : void{ ... }
    protected function onDisable() : void{ ... }
}
```

```go
//go:embed plugin.toml resources
var files embed.FS

func init() {
	plugin.RegisterGoPlugin(files, func() plugin.Plugin { return &Main{} })
}

type Main struct {
	plugin.PluginBase
	// the PHP class's properties go here
}

func (m *Main) OnLoad()          { ... }                  // optional
func (m *Main) OnEnable() error { ...; return nil }       // optional; an error disables the plugin
func (m *Main) OnDisable()       { ... }                  // optional
```

Everything `PluginBase` offers keeps its name: `GetServer()`, `GetLogger()`, `GetDataFolder()`,
`GetConfig()`, `SaveDefaultConfig()`, `ReloadConfig()`, `SaveResource()`, `GetResource()`,
`GetScheduler()`, `GetDescription()`, `GetName()`, `IsEnabled()`.

`GetServer()` returns a small interface. For the whole server API, assert it once:

```go
func (m *Main) srv() *server.Server { return m.GetServer().(*server.Server) }
```

## 4. Events

```php
class EventListener implements Listener{
    /** @priority HIGH */
    public function onJoin(PlayerJoinEvent $event) : void{
        $event->getPlayer()->sendMessage("Hi");
    }
}
$this->getServer()->getPluginManager()->registerEvents(new EventListener($this), $this);
```

```go
type listener struct{ plugin *Main }

func (l *listener) OnJoin(e *playerevent.PlayerJoinEvent) {
	if p, ok := e.GetPlayer().(*player.Player); ok {
		p.SendMessage("Hi")
	}
}

// The docblock tags (@priority, @handleCancelled, @notHandler) become this method.
func (l *listener) EventHandlerTags() map[string]map[string]string {
	return map[string]map[string]string{"OnJoin": {event.ListenerTagPriority: "HIGH"}}
}

// in OnEnable:
err := m.srv().GetPluginManager().RegisterEvents(&listener{plugin: m}, m)
```

- Every **exported** method with one parameter, a pointer to an event, and no return value is a
  handler. The parameter type chooses the event.
- Events live in one package per PHP namespace, with the same class names:
  `pocketmine\event\player\PlayerJoinEvent` → `playerevent.PlayerJoinEvent` from
  `pocketmine-go/pocketmine/event/player`. The same goes for `event/block`,
  `event/entity`, `event/inventory`, `event/world`, `event/server` and `event/plugin`.
- `$event->cancel()` → `e.Cancel()`, `$event->isCancelled()` → `e.IsCancelled()`.
- `$event->getPlayer()` returns a small interface (name, position, ...). Assert it to reach the
  full player, as above: `p, ok := e.GetPlayer().(*player.Player)`.
- A single handler without a listener type: `plugin.RegisterEvent(pm, func(e *playerevent.PlayerQuitEvent) { ... }, event.Normal, m, false)`.
- Priorities: `event.Lowest`, `Low`, `Normal`, `High`, `Highest`, `Monitor`.

## 5. Commands

Commands declared in `plugin.toml` are registered for you, exactly like PocketMine-MP.

```php
public function onCommand(CommandSender $sender, Command $command, string $label, array $args) : bool{
    if(!$sender instanceof Player){
        $sender->sendMessage("Run this in game");
        return true;
    }
    ...
}
```

```go
func (m *Main) OnCommand(sender command.Sender, cmd command.CommandLike, label string, args []string) bool {
	p, ok := sender.(*player.Player)
	if !ok {
		sender.SendMessage("Run this in game")
		return true
	}
	...
	return true // false shows the usage message
}
```

With several commands, `switch cmd.Label() { case "heal": ... }`.

## 6. Permissions

Declare them in `plugin.toml` as before. `$sender->hasPermission("x.y")` →
`sender.HasPermission("x.y")`.

## 7. Configs

```php
$this->saveDefaultConfig();
$max = $this->getConfig()->get("max-homes", 3);
$this->getConfig()->set("max-homes", 5);
$this->getConfig()->save();
$data = new Config($this->getDataFolder() . "homes.yml", Config::YAML, []);
```

```go
m.SaveDefaultConfig()
max := configInt(m.GetConfig().Get("max-homes", 3))
m.GetConfig().Set("max-homes", 5)
err := m.GetConfig().Save()
data, err := utils.NewConfig(filepath.Join(m.GetDataFolder(), "homes.toml"), utils.ConfigTOML, map[string]any{})
```

The plugin's config is `config.toml` in its data folder, copied from `resources/config.toml`.
Server owners who used the PocketMine-MP version of your plugin keep their settings: a
`config.yml` already in the data folder is converted to `config.toml` on first start (and kept as
`config.yml.bak`). Use the same key names as the original.

`Get` returns `any`, so assert the type: `name, _ := cfg.Get("name", "").(string)`. Whole numbers
come back as `int`, decimals as `float64`; a small helper such as `configInt` in
[example.go](example.go) accepts both. `GetNested`/`SetNested` work with dotted keys.

YAML to TOML, the parts that change:

| YAML | TOML |
|---|---|
| `key: value` | `key = "value"` (strings are quoted) |
| `list: [a, b]` or `- a` lines | `list = ["a", "b"]` |
| `section:` + indented keys | `[section]` header, then `key = value` lines |
| `a.b.c:` as a key (dots in the name) | quoted key: `"a.b.c" = ...` or `[permissions."a.b.c"]` |
| `key: ~` (null) | leave the key out (TOML has no null) |
| `# comment` | `# comment` |

## 8. Tasks and async work

```php
$this->getScheduler()->scheduleRepeatingTask(new ClosureTask(function() : void{ ... }), 20);
$this->getScheduler()->scheduleDelayedTask(new ClosureTask(fn() => ...), 100);
```

```go
m.GetScheduler().ScheduleRepeatingTask(scheduler.NewClosureTask(func() { ... }), 20)
m.GetScheduler().ScheduleDelayedTask(scheduler.NewClosureTask(func() { ... }), 100)
m.GetScheduler().ScheduleDelayedRepeatingTask(task, delay, period)
```

Times are in ticks (20 ticks = 1 second). The returned `*scheduler.TaskHandler` has `Cancel()`.
The plugin's tasks are cancelled when it's disabled.

**Async work.** PocketMine-MP's `AsyncTask` becomes a struct embedding `scheduler.AsyncTaskBase`:

```go
type lookupTask struct {
	scheduler.AsyncTaskBase
	name string
}

func (t *lookupTask) OnRun()        { t.SetResult(slowLookup(t.name)) } // worker goroutine
func (t *lookupTask) OnCompletion() { use(t.GetResult()) }             // main thread

m.srv().GetAsyncPool().SubmitTask(&lookupTask{name: "Steve"})
```

The same rule as in PocketMine-MP applies, and Go makes it easier to break: **game state
(players, worlds, entities, inventories) may only be touched on the main thread**, that is in event
handlers, commands, scheduled tasks and `OnCompletion`. Do slow I/O (HTTP, databases) in `OnRun` or
your own goroutine, and bring the result back through `OnCompletion` or a scheduled task. Never call
a player or world method from a goroutine you started.

## 9. Common API translations

| PocketMine-MP | PocketMine-go |
|---|---|
| `$this->getLogger()->info("x")` | `m.GetLogger().Info("x")` (also `Warning`, `Error`, `Debug`) |
| `TextFormat::GREEN . "Hi"` | `utils.Green + "Hi"` |
| `$server->getOnlinePlayers()` | `srv.GetOnlinePlayers()` |
| `$server->getPlayerExact($name)` / `getPlayerByPrefix` | `srv.GetPlayerExact(name)` / `srv.GetPlayerByPrefix(name)` (nil if offline) |
| `$server->broadcastMessage($msg)` | `srv.BroadcastMessage(msg, nil)` |
| `$server->dispatchCommand($sender, "say hi")` | `srv.DispatchCommand(sender, "say hi", false)` |
| `$server->getWorldManager()->getWorldByName("w")` | `w, ok := srv.GetWorldManager().GetWorldByName("w")` |
| `$server->getPluginManager()->getPlugin("X")` | `srv.GetPluginManager().GetPlugin("X")` |
| `$player->sendMessage` / `sendTip` / `sendPopup` / `sendTitle` | `p.SendMessage` / `SendTip` / `SendPopup` / `SendTitle(title, subtitle, fadeIn, stay, fadeOut)` |
| `$player->getName()` / `getPosition()` / `getWorld()` | `p.GetName()` / `p.GetPosition()` / `p.GetWorld()` |
| `$player->teleport(new Vector3(0, 70, 0))` | `p.Teleport(math.NewVector3(0, 70, 0))` |
| `$player->setGamemode(GameMode::CREATIVE())` | `p.SetGamemode(player.GameModeCreative)` |
| `$player->getInventory()->addItem($item)` | `p.GetInventory().AddItem(it)` (returns what didn't fit) |
| `$player->kick("reason")` | `p.Kick("reason", nil, nil)` |
| `$player->sendForm($form)` | `p.SendForm(f)`, where `f` implements `form.Form` |
| `VanillaItems::DIAMOND()` | `item.VanillaItem("diamond")` |
| `VanillaBlocks::STONE()` | `block.VanillaBlock("stone")` |
| `StringToItemParser::getInstance()->parse("diamond_sword")` | `it, ok := item.GetStringToItemParser().Parse("diamond_sword")` |
| `$item->setCount(16)` / `setCustomName("x")` | `SetCount(16)` / `SetCustomName("x")` |
| `$player->getNetworkSession()->sendDataPacket($pk)` | `p.GetNetworkSession().SendDataPacket(&packet.X{...})` with packets from [gophertunnel](https://github.com/sandertv/gophertunnel) (`minecraft/protocol/packet`) |

When a name isn't in this table, look for the PHP method in Go style on the same type: the server
code is organised like PocketMine-MP (`pocketmine/player`, `pocketmine/world`, `pocketmine/item`,
`pocketmine/block`, ...) and every ported type says which PHP class it comes from
(`grep -rn "is a port of pocketmine" pocketmine`).

## 10. What doesn't carry over

- **Loading plugins at runtime** (`.phar`, `plugins/` folder, `/reload`-style tricks): plugins are
  compiled in. Changing a plugin means rebuilding the server.
- **Virions**: replace each with a Go module, or port the parts you use into your plugin.
- **PHP-only features**: `eval`, reflection over private members, `__get`/`__call` magic, dynamic
  class names. Write the logic out explicitly.
- **Packet classes** from `pocketmine\network\mcpe\protocol`: use gophertunnel's packets instead
  (same names in Go style).
- **Blocking the main thread**: `sleep()` or synchronous HTTP in a handler freezes the whole server,
  as in PocketMine-MP. Use a task or `AsyncTask`.

## 11. Checklist before you publish

- [ ] `plugin.toml` has the same name, commands and permissions as the original.
- [ ] Every event handler, command and task of the original has a Go counterpart.
- [ ] Default config keys and file names match, so existing server owners keep their settings.
- [ ] No game state is touched from a goroutine.
- [ ] A test starts a server and loads the plugin ([example_test.go](example_test.go)).
- [ ] `gofmt -l .`, `go vet ./...` and `go test ./...` are clean.
- [ ] The README says how to add the plugin to a server.
