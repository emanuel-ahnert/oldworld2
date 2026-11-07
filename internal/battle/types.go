// Package battle resolves turn-based 1v1 battles (see design/mechanics.md). It has no
// rendering code: a turn produces a list of events the battle screen plays back.
package battle

// Element is both an attack element and a character type.
type Element int

const (
	Physical Element = iota
	Fire
	Water
	Ice
	Nature
	Air
	Holy
	Dark
	numElements
)

var elementNames = [numElements]string{"Physical", "Fire", "Water", "Ice", "Nature", "Air", "Holy", "Dark"}

func (e Element) String() string { return elementNames[e] }

// chart[attack][defender] is the multiplier; 0 means "use 1" so the table stays readable.
// See design/mechanics.md, Element Chart.
var chart = [numElements][numElements]float64{
	Fire:   {Fire: 0.5, Water: 0.5, Ice: 2, Nature: 0.5, Dark: 2},
	Water:  {Fire: 2, Water: 0.5, Nature: 2},
	Ice:    {Fire: 0.5, Water: 0.5, Ice: 0.5, Nature: 2, Air: 2},
	Nature: {Fire: 2, Ice: 2, Air: immune},
	Air:    {Water: 2, Nature: 0.5, Air: 0.5},
	Holy:   {Holy: 0.5, Dark: 2},
	Dark:   {Holy: 2, Dark: 0.5},
}

// immune is stored as a negative marker so that 0 can mean "neutral" in the table above.
const immune = -1

// Effectiveness multiplies the chart values of an attack element against all defender types.
func Effectiveness(attack Element, defender []Element) float64 {
	m := 1.0
	for _, d := range defender {
		switch v := chart[attack][d]; v {
		case 0:
		case immune:
			return 0
		default:
			m *= v
		}
	}
	return m
}
