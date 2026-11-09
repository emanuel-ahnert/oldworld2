// Command balance plays simulated battles of every class against every enemy, the player's
// moves chosen by the demo bot's planner (battle.PlanPlayerMove), and prints the share won.
//
//	go run ./cmd/balance                     # level 5 against level 5
//	go run ./cmd/balance -level 10 -gap 2    # level 10 against level 12
//	go run ./cmd/balance -show hp            # the HP left after a win instead
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"text/tabwriter"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
	"oldworld2/internal/creature"
)

func main() {
	level := flag.Int("level", 5, "the player's level")
	gap := flag.Int("gap", 0, "how many levels above the player the enemies are")
	n := flag.Int("n", 100, "battles per class and enemy")
	show := flag.String("show", "win", "what the cells show: win (% won), hp (% HP left after a win) or rounds (per battle)")
	flag.Parse()

	classes := make([]character.Class, character.NumClasses)
	for i := range classes {
		classes[i] = character.Class(i)
	}
	kinds := creature.All()
	type cell struct {
		won, rounds int
		hp          float64
	}
	cells := make([][]cell, len(kinds))
	for i := range cells {
		cells[i] = make([]cell, len(classes))
	}

	// One job per class and enemy, spread over the CPUs.
	type job struct{ k, c int }
	jobs := make(chan job)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				ch := character.New(classes[j.c])
				for ch.Level < *level {
					ch.GainXP(character.XPToNext(ch.Level))
				}
				ch.EquipBotMoves() // the loadout the demo bot plays with
				ch.Heal()
				var c cell
				for i := range *n {
					o := battle.Simulate(ch.Combatant(), character.Enemy(kinds[j.k], *level+*gap), uint64(i+1))
					c.rounds += o.Rounds
					if o.Won {
						c.won++
						c.hp += o.HPLeft
					}
				}
				cells[j.k][j.c] = c
			}
		}()
	}
	for k := range kinds {
		for c := range classes {
			jobs <- job{k, c}
		}
	}
	close(jobs)
	wg.Wait()

	fmt.Printf("Level %d against level %d, %d battles each: %s\n\n", *level, *level+*gap, *n, map[string]string{
		"win": "% of battles won", "hp": "% HP left after a win", "rounds": "rounds per battle"}[*show])
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprint(tw, "\t")
	for _, c := range classes {
		fmt.Fprintf(tw, "%s\t", c.Info().Name)
	}
	fmt.Fprintln(tw)
	value := func(c cell) string {
		switch *show {
		case "hp":
			if c.won == 0 {
				return "-"
			}
			return fmt.Sprintf("%.0f", 100*c.hp/float64(c.won))
		case "rounds":
			return fmt.Sprintf("%.1f", float64(c.rounds)/float64(*n))
		}
		return fmt.Sprintf("%.0f", 100*float64(c.won)/float64(*n))
	}
	for k, kind := range kinds {
		fmt.Fprintf(tw, "%s\t", creature.Get(kind).Name)
		for c := range classes {
			fmt.Fprintf(tw, "%s\t", value(cells[k][c]))
		}
		fmt.Fprintln(tw)
	}
	tw.Flush()
}
