package game

import (
	"fmt"
	"strconv"
	"strings"

	"oldworld2/internal/world"
)

// ScriptStep holds an input for a number of frames. Used for automated test runs.
type ScriptStep struct {
	Input  world.Input
	Camp   bool // press the camp key on the first frame of the step
	Use    bool // press the interact key on the first frame of the step
	Menu   bool // press the abilities menu key on the first frame of the step
	Esc    bool // press Escape (main menu) on the first frame of the step
	Frames int
}

// ParseScript parses steps like "right:40,run-down:30,wait:20,camp:60,use:30,menu:60".
func ParseScript(s string) ([]ScriptStep, error) {
	if s == "" {
		return nil, nil
	}
	dirs := map[string]world.Dir{"up": world.Up, "down": world.Down, "left": world.Left, "right": world.Right}
	var steps []ScriptStep
	for _, part := range strings.Split(s, ",") {
		name, n, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			return nil, fmt.Errorf("script step %q: want name:frames", part)
		}
		frames, err := strconv.Atoi(n)
		if err != nil {
			return nil, fmt.Errorf("script step %q: %w", part, err)
		}
		var in world.Input
		if name == "camp" || name == "use" || name == "menu" || name == "esc" {
			steps = append(steps, ScriptStep{Camp: name == "camp", Use: name == "use", Menu: name == "menu", Esc: name == "esc", Frames: frames})
			continue
		}
		rest, run := strings.CutPrefix(name, "run-")
		if rest != "wait" {
			d, ok := dirs[rest]
			if !ok {
				return nil, fmt.Errorf("script step %q: unknown direction", part)
			}
			in = world.Input{Dir: d, HasDir: true, Run: run}
		}
		steps = append(steps, ScriptStep{Input: in, Frames: frames})
	}
	return steps, nil
}

// scriptCamp reports whether the camp key is pressed at this frame.
func scriptCamp(steps []ScriptStep, frame int) bool {
	s, start := scriptStep(steps, frame)
	return s.Camp && start
}

// scriptUse reports whether the interact key is pressed at this frame.
func scriptUse(steps []ScriptStep, frame int) bool {
	s, start := scriptStep(steps, frame)
	return s.Use && start
}

func scriptStep(steps []ScriptStep, frame int) (ScriptStep, bool) {
	for _, s := range steps {
		if frame < s.Frames {
			return s, frame == 0
		}
		frame -= s.Frames
	}
	return ScriptStep{}, false
}

func scriptInput(steps []ScriptStep, frame int) world.Input {
	for _, s := range steps {
		if frame < s.Frames {
			return s.Input
		}
		frame -= s.Frames
	}
	return world.Input{}
}
