package example

import (
	"os"
	"path/filepath"
	"testing"

	"pocketmine-go/pocketmine/command"
	"pocketmine-go/pocketmine/console"
	"pocketmine-go/pocketmine/log"
	"pocketmine-go/pocketmine/server"
)

func TestExamplePluginLoads(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte("level-seed=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := server.New(dir, log.NewSimpleLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer s.ForceShutdown()

	p := s.GetPluginManager().GetPlugin("ExamplePlugin")
	if p == nil || !p.IsEnabled() {
		t.Fatal("ExamplePlugin wasn't loaded and enabled")
	}
	if _, err := os.Stat(filepath.Join(p.GetDataFolder(), "config.toml")); err != nil {
		t.Errorf("the default config wasn't saved: %v", err)
	}
	if got := p.(*Main).GetConfig().Get("tip-interval", nil); got != 5 {
		t.Errorf("tip-interval = %v (%T), want 5 from config.toml", got, got)
	}
	if p.(*Main).tipTask == nil {
		t.Error("the tip task wasn't scheduled")
	}

	for _, name := range []string{"example", "ex", "exampleplugin:example"} {
		cmd, ok := s.GetSimpleCommandMap().GetCommand(name).(command.PluginOwned)
		if !ok || cmd.GetOwningPlugin() != p {
			t.Errorf("/%s isn't the plugin's command", name)
		}
	}

	sender := console.NewConsoleCommandSender(s, s.GetLanguage())
	if !s.DispatchCommand(sender, "example", false) || !s.DispatchCommand(sender, "example reload", false) {
		t.Error("/example didn't run")
	}
}
