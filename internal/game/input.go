package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Keyboard and gamepad input on raylib. Gamepad buttons use raylib's standard layout:
// the "right face" buttons are A (down), B (right), X (left) and Y (up) on an Xbox pad.

const maxPads = 4

// keyPressed reports a fresh press of any of the keys.
func keyPressed(keys ...int32) bool {
	for _, k := range keys {
		if rl.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

// padPressed reports a fresh press of any of the buttons on any gamepad.
func padPressed(buttons ...int32) bool {
	for id := int32(0); id < maxPads; id++ {
		if !rl.IsGamepadAvailable(id) {
			continue
		}
		for _, b := range buttons {
			if rl.IsGamepadButtonPressed(id, b) {
				return true
			}
		}
	}
	return false
}

// padDown reports whether a button is held on any gamepad.
func padDown(b int32) bool {
	for id := int32(0); id < maxPads; id++ {
		if rl.IsGamepadAvailable(id) && rl.IsGamepadButtonDown(id, b) {
			return true
		}
	}
	return false
}

// heldKeys and heldPads count how many frames each tracked key or button has been held
// (0 when up).
var (
	trackedKeys = []int32{rl.KeyW, rl.KeyA, rl.KeyS, rl.KeyD, rl.KeyUp, rl.KeyDown, rl.KeyLeft, rl.KeyRight, rl.KeyBackspace}
	trackedPads = []int32{rl.GamepadButtonLeftFaceUp, rl.GamepadButtonLeftFaceDown, rl.GamepadButtonLeftFaceLeft, rl.GamepadButtonLeftFaceRight}
	heldKeys    = map[int32]int{}
	heldPads    = map[int32]int{}
)

// updateHeld advances the hold counters; call once per frame before reading input.
func updateHeld() {
	for _, k := range trackedKeys {
		if rl.IsKeyDown(k) {
			heldKeys[k]++
		} else {
			heldKeys[k] = 0
		}
	}
	for _, b := range trackedPads {
		if padDown(b) {
			heldPads[b]++
		} else {
			heldPads[b] = 0
		}
	}
}

// typedChars returns the characters typed this frame.
func typedChars() []rune {
	var out []rune
	for r := rl.GetCharPressed(); r != 0; r = rl.GetCharPressed() {
		out = append(out, r)
	}
	return out
}
