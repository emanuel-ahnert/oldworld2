// Package save stores the single autosave in an SQLite file (see design/game-design.md,
// Save System). The save is split into parts that are written independently in small
// transactions, so walking around only rewrites one row now and then.
package save

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

const version = 1

const schema = `
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS player (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	x INTEGER NOT NULL, y INTEGER NOT NULL, facing INTEGER NOT NULL,
	checkpoint_x INTEGER NOT NULL, checkpoint_y INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS character (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	class INTEGER NOT NULL, level INTEGER NOT NULL, xp INTEGER NOT NULL,
	hp REAL NOT NULL, mp REAL NOT NULL, money INTEGER NOT NULL,
	known TEXT NOT NULL, active TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS defeated (npc INTEGER PRIMARY KEY);
`

// Player is the saved position.
type Player struct {
	X, Y, Facing             int
	CheckpointX, CheckpointY int
}

// Character is the saved main character.
type Character struct {
	Class, Level, XP int
	HP, MP           float64
	Money            int
	Known, Active    []string
}

// State is a complete save.
type State struct {
	Seed      int64
	Home      [2]int // the zone the game started in (see world.World.Home)
	Player    Player
	Character Character
	Defeated  []uint64
}

// Store is an open save file.
type Store struct {
	db *sql.DB
}

// DefaultPath returns the save location: %AppData%\oldworld2\save.db on Windows, elsewhere
// $XDG_DATA_HOME/oldworld2/save.db or ~/.local/share/oldworld2/save.db.
func DefaultPath() (string, error) {
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "oldworld2", "save.db"), nil
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "oldworld2", "save.db"), nil
}

// Open opens or creates a save file.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// WAL keeps writes small and a crash mid-write can't corrupt the save.
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(2000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("save schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the file.
func (s *Store) Close() error { return s.db.Close() }

// ErrNoSave means the file holds no game yet.
var ErrNoSave = errors.New("no saved game")

// Load reads the whole save.
func (s *Store) Load() (*State, error) {
	var st State
	var v int
	err := s.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM meta WHERE key = 'version'`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoSave
	}
	if err != nil {
		return nil, err
	}
	if v != version {
		return nil, fmt.Errorf("save version %d, want %d", v, version)
	}
	if err := s.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM meta WHERE key = 'seed'`).Scan(&st.Seed); err != nil {
		return nil, err
	}
	// The home zone, as "x,y"; saves from before the start zone option have none (0,0).
	var home string
	if err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'home'`).Scan(&home); err == nil {
		fmt.Sscanf(home, "%d,%d", &st.Home[0], &st.Home[1])
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	p := &st.Player
	if err := s.db.QueryRow(`SELECT x, y, facing, checkpoint_x, checkpoint_y FROM player WHERE id = 1`).
		Scan(&p.X, &p.Y, &p.Facing, &p.CheckpointX, &p.CheckpointY); err != nil {
		return nil, fmt.Errorf("load player: %w", err)
	}
	c := &st.Character
	var known, active string
	if err := s.db.QueryRow(`SELECT class, level, xp, hp, mp, money, known, active FROM character WHERE id = 1`).
		Scan(&c.Class, &c.Level, &c.XP, &c.HP, &c.MP, &c.Money, &known, &active); err != nil {
		return nil, fmt.Errorf("load character: %w", err)
	}
	if err := json.Unmarshal([]byte(known), &c.Known); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(active), &c.Active); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT npc FROM defeated`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		st.Defeated = append(st.Defeated, uint64(id))
	}
	return &st, rows.Err()
}

// New replaces the save with a fresh game.
func (s *Store) New(st *State) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, q := range []string{`DELETE FROM meta`, `DELETE FROM player`, `DELETE FROM character`, `DELETE FROM defeated`} {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`INSERT INTO meta (key, value) VALUES ('version', ?), ('seed', ?), ('home', ?)`, version, st.Seed,
			fmt.Sprintf("%d,%d", st.Home[0], st.Home[1])); err != nil {
			return err
		}
		if err := writePlayer(tx, st.Player); err != nil {
			return err
		}
		return writeCharacter(tx, st.Character)
	})
}

// SavePlayer writes the position and checkpoint.
func (s *Store) SavePlayer(p Player) error {
	return s.tx(func(tx *sql.Tx) error { return writePlayer(tx, p) })
}

// SaveCharacter writes the main character.
func (s *Store) SaveCharacter(c Character) error {
	return s.tx(func(tx *sql.Tx) error { return writeCharacter(tx, c) })
}

// AddDefeated remembers a defeated enemy.
func (s *Store) AddDefeated(npc uint64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO defeated (npc) VALUES (?)`, int64(npc))
	return err
}

func writePlayer(tx *sql.Tx, p Player) error {
	_, err := tx.Exec(`INSERT INTO player (id, x, y, facing, checkpoint_x, checkpoint_y) VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET x = excluded.x, y = excluded.y, facing = excluded.facing,
		checkpoint_x = excluded.checkpoint_x, checkpoint_y = excluded.checkpoint_y`,
		p.X, p.Y, p.Facing, p.CheckpointX, p.CheckpointY)
	return err
}

func writeCharacter(tx *sql.Tx, c Character) error {
	known, err := json.Marshal(c.Known)
	if err != nil {
		return err
	}
	active, err := json.Marshal(c.Active)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO character (id, class, level, xp, hp, mp, money, known, active) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET class = excluded.class, level = excluded.level, xp = excluded.xp,
		hp = excluded.hp, mp = excluded.mp, money = excluded.money, known = excluded.known, active = excluded.active`,
		c.Class, c.Level, c.XP, c.HP, c.MP, c.Money, string(known), string(active))
	return err
}

func (s *Store) tx(f func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
