// Package audio plays music and sound effects (made by tools/audio/makeaudio.py) with raylib.
//
// Music crossfades between tracks; a track that fades out is paused, so the map music picks
// up where it left off after a battle. Everything pauses while the game's window is in the
// background. Sounds are loaded on first use. A disabled player (test runs, -mute) does
// nothing and never opens the audio device, so test runs are silent.
package audio

import (
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	musicVolume = 0.45
	soundVolume = 0.7
	fadeFrames  = 45 // crossfade length at 60 frames per second
)

// Player plays the game's audio. The zero value and a disabled player do nothing.
type Player struct {
	on     bool
	paused bool   // everything paused while the window is in the background (SetPaused)
	dir    string // assets/generated/audio
	sounds map[string]rl.Sound
	tracks map[string]*track
	cur    *track // the track playing or fading in
}

type track struct {
	music rl.Music
	vol   float32 // 0..1 of musicVolume, for fades
}

// New opens the audio device, unless on is false. assetsDir is the game's assets directory.
func New(assetsDir string, on bool) *Player {
	p := &Player{on: on, dir: filepath.Join(assetsDir, "generated", "audio"),
		sounds: map[string]rl.Sound{}, tracks: map[string]*track{}}
	if on {
		rl.InitAudioDevice()
		p.on = rl.IsAudioDeviceReady() // no sound device: play nothing
	}
	return p
}

// Sound plays a sound effect: a file in sounds/ without ".wav", e.g. "menu_accept".
func (p *Player) Sound(name string) {
	p.SoundAt(name, 1)
}

// SoundAt plays a sound effect at a volume from 0 to 1.
func (p *Player) SoundAt(name string, volume float32) {
	if p == nil || !p.on || p.paused || name == "" {
		return
	}
	s, ok := p.sounds[name]
	if !ok {
		s = rl.LoadSound(filepath.Join(p.dir, "sounds", name+".wav"))
		p.sounds[name] = s
	}
	rl.SetSoundVolume(s, soundVolume*volume)
	rl.PlaySound(s)
}

// Music switches to a track (a file in music/ without ".ogg"); "" fades the music out.
// With restart, the track starts from the beginning instead of where it was paused. While
// everything is paused (SetPaused), the switch waits until it isn't.
func (p *Player) Music(name string, restart bool) {
	if p == nil || !p.on || p.paused {
		return
	}
	var t *track
	if name != "" {
		if t = p.tracks[name]; t == nil {
			t = &track{music: rl.LoadMusicStream(filepath.Join(p.dir, "music", name+".ogg"))}
			t.music.Looping = true
			p.tracks[name] = t
		}
	}
	if t == p.cur {
		return
	}
	p.cur = t
	if t == nil {
		return
	}
	if restart || !rl.IsMusicStreamPlaying(t.music) {
		if restart {
			rl.SeekMusicStream(t.music, 0)
		}
		rl.SetMusicVolume(t.music, 0)
		if restart || t.vol == 0 {
			rl.PlayMusicStream(t.music)
		}
		rl.ResumeMusicStream(t.music)
	}
}

// SetPaused pauses all music and sounds (or resumes them). The game pauses them while its
// window is in the background: it may stop drawing frames then, and a music stream that isn't
// updated loops its last buffer.
func (p *Player) SetPaused(paused bool) {
	if p == nil || !p.on || paused == p.paused {
		return
	}
	p.paused = paused
	for _, t := range p.tracks {
		if t.vol == 0 && t != p.cur {
			continue // paused already, by its fade-out
		}
		if paused {
			rl.PauseMusicStream(t.music)
		} else {
			rl.ResumeMusicStream(t.music)
		}
	}
	for _, s := range p.sounds {
		if paused {
			rl.PauseSound(s)
		} else {
			rl.ResumeSound(s)
		}
	}
}

// Update streams the music and runs the fades; call it once per frame.
func (p *Player) Update() {
	if p == nil || !p.on || p.paused {
		return
	}
	for _, t := range p.tracks {
		if t != p.cur && t.vol == 0 {
			continue
		}
		if t == p.cur {
			t.vol = min(1, t.vol+1.0/fadeFrames)
		} else if t.vol = max(0, t.vol-1.0/fadeFrames); t.vol == 0 {
			rl.PauseMusicStream(t.music)
			continue
		}
		rl.SetMusicVolume(t.music, musicVolume*t.vol)
		rl.UpdateMusicStream(t.music)
	}
}

// Close frees the audio and closes the device.
func (p *Player) Close() {
	if p == nil || !p.on {
		return
	}
	for _, s := range p.sounds {
		rl.UnloadSound(s)
	}
	for _, t := range p.tracks {
		rl.UnloadMusicStream(t.music)
	}
	rl.CloseAudioDevice()
}
