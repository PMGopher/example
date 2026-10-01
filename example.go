// Package example is a small PocketMine-go plugin: copy this repository to start your own.
//
// It shows the parts most plugins need:
//   - registering the plugin (init, below) and its plugin.toml / resources (embedded)
//   - OnEnable / OnDisable
//   - a config file with defaults (resources/config.toml)
//   - an event listener (PlayerJoinEvent)
//   - a command declared in plugin.toml, handled by OnCommand
//   - a repeating task on the plugin's scheduler
//
// To load it, add it to the server (see README.md):
//
//	go get github.com/PMGopher/example@latest
//
// and import it in the server's cmd/pocketmine-go/plugins.go:
//
//	import _ "github.com/PMGopher/example"
package example

import (
	"embed"
	"fmt"
	"strings"

	"pocketmine-go/pocketmine/command"
	playerevent "pocketmine-go/pocketmine/event/player"
	"pocketmine-go/pocketmine/player"
	"pocketmine-go/pocketmine/plugin"
	"pocketmine-go/pocketmine/scheduler"
	"pocketmine-go/pocketmine/server"
	"pocketmine-go/pocketmine/utils"
)

// files is the plugin's folder: plugin.toml at the root, default files in resources/. They are
// compiled into the server binary, so there is nothing to copy next to it.
//
//go:embed plugin.toml resources
var files embed.FS

// init registers the plugin. The server loads it on startup like any other plugin: it checks the
// api version in plugin.toml, plugin_list.toml, dependencies and load order, then calls OnEnable.
func init() {
	plugin.RegisterGoPlugin(files, func() plugin.Plugin { return &Main{} })
}

// Main is the plugin's main type (what "main" in plugin.toml names). Embedding plugin.PluginBase
// gives it everything a plugin needs: GetServer, GetLogger, GetConfig, GetScheduler, ...
type Main struct {
	plugin.PluginBase

	tipTask *scheduler.TaskHandler
}

// OnEnable runs when the plugin is enabled. Returning an error disables it again.
func (m *Main) OnEnable() error {
	// Copies resources/config.toml to plugin_data/ExamplePlugin/config.toml unless it's already there.
	m.SaveDefaultConfig()

	// Every exported method of the listener that takes one event is registered as its handler.
	if err := m.srv().GetPluginManager().RegisterEvents(&joinListener{plugin: m}, m); err != nil {
		return err
	}

	m.scheduleTip()
	m.GetLogger().Info(utils.Green + "ExamplePlugin enabled! Try /example in game.")
	return nil
}

// OnDisable runs when the plugin is disabled (at the latest when the server stops). Tasks and
// event handlers of the plugin are removed by the server; save your own data here.
func (m *Main) OnDisable() {
	m.GetLogger().Info("ExamplePlugin disabled, bye!")
}

// OnCommand handles the commands declared in plugin.toml. The server has already checked the
// permission. Return false to show the usage message.
func (m *Main) OnCommand(sender command.Sender, cmd command.CommandLike, label string, args []string) bool {
	if len(args) == 1 && strings.EqualFold(args[0], "reload") {
		m.ReloadConfig()
		m.scheduleTip()
		sender.SendMessage(utils.Green + "ExamplePlugin config reloaded.")
		return true
	}
	if len(args) != 0 {
		return false
	}

	p, isPlayer := sender.(*player.Player)
	if !isPlayer {
		sender.SendMessage("Hello from the console! " + m.status())
		return true
	}
	pos := p.GetPosition()
	sender.SendMessage(fmt.Sprintf("%sHello, %s!%s You are at %.1f, %.1f, %.1f in \"%s\".",
		utils.Green, p.GetName(), utils.Reset, pos.X, pos.Y, pos.Z, p.GetWorld().GetFolderName()))
	sender.SendMessage(m.status())
	return true
}

// status is a line about the server, from the server API.
func (m *Main) status() string {
	s := m.srv()
	return fmt.Sprintf("%s%d player(s) online, %.2f TPS, %s %s", utils.Gray,
		len(s.GetOnlinePlayers()), s.GetTicksPerSecond(), s.GetName(), s.GetPocketMineVersion())
}

// scheduleTip (re)starts the repeating task that broadcasts the tip from the config.
func (m *Main) scheduleTip() {
	if m.tipTask != nil {
		m.tipTask.Cancel()
		m.tipTask = nil
	}
	minutes := configInt(m.GetConfig().Get("tip-interval", 5))
	tip, _ := m.GetConfig().Get("tip", "").(string)
	if minutes <= 0 || tip == "" {
		return
	}
	// Delays and periods are in server ticks: 20 ticks = 1 second. The first tip comes after one
	// period (ScheduleRepeatingTask would run it straight away).
	period := minutes * 60 * 20
	task, err := m.GetScheduler().ScheduleDelayedRepeatingTask(scheduler.NewClosureTask(func() {
		m.srv().BroadcastMessage(tip, nil)
	}), period, period)
	if err != nil {
		m.GetLogger().Error("Could not schedule the tip: " + err.Error())
		return
	}
	m.tipTask = task
}

// srv is the full server API. plugin.Server only has what the plugin package itself needs.
func (m *Main) srv() *server.Server { return m.GetServer().(*server.Server) }

// joinListener greets players. Listeners are plain types: the method's parameter type chooses
// the event.
type joinListener struct{ plugin *Main }

// OnJoin shows the welcome title from the config to the player who joined.
func (l *joinListener) OnJoin(e *playerevent.PlayerJoinEvent) {
	p, ok := e.GetPlayer().(*player.Player)
	if !ok {
		return
	}
	config := l.plugin.GetConfig()
	title, _ := config.Get("welcome-title", "").(string)
	subtitle, _ := config.Get("welcome-subtitle", "").(string)
	title = strings.ReplaceAll(title, "{player}", p.GetName())
	subtitle = strings.ReplaceAll(subtitle, "{player}", p.GetName())
	p.SendTitle(title, subtitle, -1, -1, -1)
}

// configInt reads a whole number from a config value.
func configInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
