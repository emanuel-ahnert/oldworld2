// Command game runs Oldworld2.
//
// Test runs: -shot saves a screenshot after -shot-frame frames and quits; -script replaces
// keyboard input. Both open the window unfocused, titled "Oldworld2 test run", so a window
// rule can keep it out of the way, and without audio.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/assets"
	"oldworld2/internal/character"
	"oldworld2/internal/game"
	"oldworld2/internal/procgen"
	"oldworld2/internal/save"
)

func main() {
	seed := flag.Int64("seed", 1, "world seed")
	class := flag.String("class", "warrior", "character class: warrior, mage, rogue, hunter, warlock, priest, paladin, shaman or druid")
	level := flag.Int("level", 1, "start at this level (for testing)")
	newGame := flag.Bool("new", false, "start a new game (replaces the save); -seed, -class and -level only apply to new games")
	at := flag.String("at", "", "start a new game at tile X,Y instead of the start point (for testing)")
	zone := flag.String("zone", "grassland", "start a new game in the nearest zone of this type: "+zoneNames())
	title := flag.Bool("title", false, "show the title screen even in test runs")
	demo := flag.Bool("demo", false, "start in demo mode: the game plays itself with a random class and zone")
	savePath := flag.String("save", "", "save file (default: ~/.local/share/oldworld2/save.db; test runs don't save unless set)")
	assetsDir := flag.String("assets", "", "assets directory (default: ./assets)")
	shot := flag.String("shot", "", "save a screenshot to this PNG file and quit")
	shotFrame := flag.Int("shot-frame", 60, "frame to take the screenshot at")
	script := flag.String("script", "", `scripted input, e.g. "right:40,run-down:30,wait:20"`)
	mute := flag.Bool("mute", false, "no music or sound (test runs are always silent)")
	flag.Parse()

	if err := run(*seed, *class, *level, *at, *zone, *newGame, *title, *demo, *mute, *savePath, *assetsDir, *shot, *shotFrame, *script); err != nil {
		fmt.Fprintln(os.Stderr, err)
		showFatal(err)
		os.Exit(1)
	}
}

// zoneNames lists the zone types a game can start in, as -zone takes them.
func zoneNames() string {
	var names []string
	for _, z := range procgen.Implemented {
		names = append(names, strings.ReplaceAll(strings.ToLower(z.String()), " ", "_"))
	}
	return strings.Join(names, ", ")
}

func run(seed int64, className string, level int, at, zoneName string, newGame, forceTitle, demo, mute bool, savePath, assetsFlag, shot string, shotFrame int, script string) error {
	var startAt *[2]int
	if at != "" {
		var x, y int
		if _, err := fmt.Sscanf(at, "%d,%d", &x, &y); err != nil {
			return fmt.Errorf("-at %q: want X,Y", at)
		}
		startAt = &[2]int{x, y}
	}
	zone, ok := procgen.ParseZoneType(zoneName)
	if !ok || !slices.Contains(procgen.Implemented, zone) {
		return fmt.Errorf("unknown zone %q (want %s)", zoneName, zoneNames())
	}
	class, ok := character.ParseClass(className)
	if !ok {
		return fmt.Errorf("unknown class %q (want warrior, mage, rogue, hunter, warlock, priest, paladin, shaman or druid)", className)
	}
	dir, err := assets.FindDir(assetsFlag)
	if err != nil {
		return err
	}
	steps, err := game.ParseScript(script)
	if err != nil {
		return err
	}
	testRun := shot != "" || script != ""
	var store *save.Store
	if savePath != "" || !testRun {
		if savePath == "" {
			if savePath, err = save.DefaultPath(); err != nil {
				return err
			}
		}
		if store, err = save.Open(savePath); err != nil {
			return fmt.Errorf("open save %s: %w", savePath, err)
		}
		defer store.Close()
	}

	title := "Oldworld2"
	if testRun {
		title = "Oldworld2 test run" // a window rule can send these to a hidden workspace
	}
	// Under native Wayland with display scaling, raylib draws into only part of the window;
	// XWayland avoids that.
	os.Unsetenv("WAYLAND_DISPLAY")
	flags := uint32(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	if testRun {
		flags |= rl.FlagWindowUnfocused
	} else {
		flags |= rl.FlagVsyncHint // not in test runs: a hidden window gets no frame callbacks
	}
	rl.SetConfigFlags(flags)
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(game.ScreenW*4, game.ScreenH*4, title)
	defer rl.CloseWindow()
	rl.SetExitKey(0) // Escape opens the menu
	rl.SetTargetFPS(60)

	g, err := game.New(game.Options{
		Seed: seed, Class: class, Level: level, StartAt: startAt, StartZone: zone, AssetsDir: dir, Store: store, NewGame: newGame,
		// Without command line choices, the title screen lets the player continue or set up a new game.
		Title: forceTitle || (!testRun && !newGame && !demo && startAt == nil), Demo: demo, Script: steps, ShotPath: shot, ShotFrame: shotFrame,
		Sound: !testRun && !mute, // test runs must stay silent
	})
	if err != nil {
		return err
	}
	defer g.Close() // saves what changed since the last autosave

	for !rl.WindowShouldClose() {
		if err := g.Update(); err != nil {
			if errors.Is(err, game.ErrDone) {
				return nil
			}
			return err
		}
		rl.BeginDrawing()
		g.Draw()
		rl.EndDrawing()
		if g.ShotDue() {
			img := rl.LoadImageFromScreen()
			rl.ExportImage(*img, shot)
			rl.UnloadImage(img)
		}
	}
	return nil
}
