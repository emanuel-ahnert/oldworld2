package procgen

import (
	"strings"
	"sync"
)

// Zone names: every zone gets a generated name in the style of its theme, shown when the
// player enters it. Names are deterministic per world seed and zone.
//
// Two things keep repeats rare:
//   - A large name space: patterns combine word lists and syllable-built proper names, giving
//     hundreds of thousands to millions of names per theme.
//   - Local uniqueness: each zone has a few candidate names and takes the first one that no
//     zone within uniqueRadius could also take (compared against the neighbors' candidates,
//     not their final names, so there are no chain reactions). Names never repeat within that
//     area.

const (
	saltName     uint64 = 500
	uniqueRadius        = 4 // no repeated names within this many zones
	candidates          = 6 // candidate names per zone
)

var nameLists = map[string][]string{
	// Grassland: English place names.
	"gstart": {"Bramble", "Willow", "Oak", "Ash", "Thorn", "Mead", "Fern", "Hazel", "Clover", "Elder",
		"Briar", "Moss", "Barley", "Rowan", "Sorrel", "Alder", "Heather", "Linden", "Marigold", "Primrose",
		"Thistle", "Yarrow", "Bracken", "Cress", "Mallow", "Sedge", "Tansy", "Wheat", "Rye", "Larch",
		"Holly", "Ivy", "Juniper", "Laurel", "Myrtle", "Pine", "Quince", "Sage", "Teasel", "Vetch"},
	"gend": {"wick", "dale", "ford", "mere", "stead", "holm", "brook", "vale", "ley", "combe", "field",
		"hollow", "bury", "ton", "worth", "ham", "thorpe", "by", "well", "moor", "shaw", "hurst", "den",
		"stow", "cote", "garth", "hythe", "lowe", "mead", "wold"},
	"gplace": {"Meadows", "Fields", "Downs", "Heath", "Lowlands", "Greens", "Commons", "Pastures", "Wolds",
		"Glades", "Hills", "Vale", "Moors", "Leas", "Hollows", "Brooklands", "Marches", "Uplands", "Shires",
		"Copses"},
	"gadj": {"Quiet", "Golden", "Rolling", "Windy", "Gentle", "Verdant", "Misty", "Sunlit", "Emerald",
		"Whispering", "Hidden", "Wild", "Old", "Bright", "Drowsy", "Amber", "Silver", "Lonely", "Merry",
		"Dappled", "Hazy", "Blooming", "Weeping", "Crooked", "Far"},
	"gperson": {"Aldric", "Bertha", "Cedric", "Edith", "Godwin", "Hilda", "Osric", "Wynn", "Alma", "Brand",
		"Elric", "Freya", "Gareth", "Mabel", "Oswin", "Rowena", "Tilda", "Wulf", "Agnes", "Edmund",
		"Aelfric", "Beornot", "Cuthbert", "Dunstan", "Eadgyth", "Hereward", "Leofric", "Mildred",
		"Ethel", "Wilfred", "Alwin", "Bede", "Hugh", "Isolde", "Maud", "Odo", "Rosamund", "Walter"},

	// Desert: Middle Eastern style names built from syllables.
	"dplace": {"Sands of", "Dunes of", "Wastes of", "Oasis of", "Winds of", "Mirage of", "Plains of",
		"Expanse of", "Steppe of", "Wells of", "Gate of", "Shadow of"},
	"dplace2": {"Sands", "Dunes", "Wastes", "Expanse", "Flats", "Barrens", "Reach", "Basin"},
	"ds1": {"Qa", "Ra", "Ma", "Sa", "Za", "Ha", "Ka", "Ja", "Na", "Fa", "Ba", "Ta", "Da", "Sha", "Kha",
		"Ya", "Ghu", "Dhu", "Su", "Mu", "Nu", "Zu", "Bi", "Fi", "Hi", "Ri"},
	"ds2": {"", "", "li", "ra", "ma", "di", "ha", "zi", "bi", "ya", "si", "ru"},
	"ds3": {"dir", "mal", "fir", "sim", "rah", "kal", "bar", "zan", "hir", "rif", "sar", "mun", "dun", "jar",
		"bek", "qim", "sul", "lan", "zir", "hab", "nar", "tah", "far", "wan", "rak", "sab"},

	// Ancient desert: Egyptian style names built from syllables.
	"eplace": {"Valley of", "Sands of", "Necropolis of", "Dunes of", "Tombs of", "Plains of", "Gates of",
		"Shadow of", "Temple Fields of", "Throne of", "Eye of", "Steps of"},
	"es1": {"Kheper", "Neb", "Sekh", "Amen", "Hat", "Mer", "Sen", "Ankh", "Ptah", "Nefer", "Tef", "Khon",
		"Seth", "Hor", "Wadj", "Ra", "Iset", "Khaf", "Mut", "Sob", "Ten", "Djed", "Heq", "Wser", "Taw"},
	"es2": {"", "", "", "em", "en", "et", "i", "ab"},
	"es3": {"u", "et", "hotep", "mose", "ankh", "ra", "ti", "ka", "nut", "su", "ib", "ses", "ef", "ari",
		"at", "aten", "ep", "is", "mes", "kare"},
	// North: Norse style names built from a first part and an ending.
	"nstart": {"Frost", "Ulf", "Rav", "Thor", "Skog", "Hvit", "Jarn", "Bjorn", "Kald", "Snae", "Hrim", "Ask",
		"Stein", "Dal", "Fjall", "Isa", "Svart", "Grim", "Hald", "Ormr", "Brekk", "Lind", "Vind", "Myrk",
		"Sigr", "Holt", "Eik", "Kol", "Rand", "Sval"},
	"nend": {"heim", "vik", "fjord", "dal", "gard", "holm", "stad", "nes", "berg", "by", "mark", "vang",
		"land", "fell", "voll", "haug", "skar", "botn", "rud", "mo"},
	"nplace": {"Fells", "Wastes", "Reaches", "Wilds", "Marches", "Woods", "Heights", "Barrows", "Shores",
		"Holds", "Peaks", "Hollows", "Fjords", "Moors"},
	"nadj": {"Frozen", "Silent", "Grey", "White", "Howling", "Bitter", "Iron", "Pale", "Wolfhaunted",
		"Starlit", "Deep", "Endless", "Black", "Windswept", "Shrouded", "Old", "Hoarfrost", "Bleak"},
	"nperson": {"Sigrun", "Ragnar", "Astrid", "Halvard", "Ingrid", "Bjarki", "Solveig", "Eirik", "Gudrun",
		"Thorvald", "Hilde", "Ketil", "Ylva", "Orm", "Runa", "Leif", "Sigmund", "Torunn", "Ulfar", "Vigdis",
		"Arne", "Brynja", "Dagny", "Gunnar", "Hakon", "Kari"},

	// Mediterranean: Greek style names built from syllables.
	"ms1": {"Kal", "Myr", "Thal", "Ae", "Lyk", "Ner", "Phi", "Ari", "Kor", "Del", "Xan", "Zak", "Eu", "Ky", "Pel", "Ser",
		"Theo", "Ist", "Mel", "Ol", "Par", "Sky", "Tri", "Hel", "Ach", "Ly"},
	"ms2": {"", "", "a", "o", "i", "e", "an", "es"},
	"ms3": {"os", "ia", "is", "on", "essa", "ene", "ados", "inthos", "ora", "aia", "ikos", "ope", "eia", "assos", "ymna",
		"ithra", "ae", "ymos"},
	"mplace": {"Olive Groves of", "Shores of", "Hills of", "Vineyards of", "Ruins of", "Bay of", "Cliffs of", "Plains of",
		"Gardens of", "Temple Fields of", "Isles of", "Springs of"},
	"mplace2": {"Coast", "Hills", "Groves", "Heights", "Shallows", "Terraces", "Headlands", "Meadows"},
	"mepithet": {"the Golden", "the Wine-Dark", "the Sunlit", "the Blessed", "of the Muses", "of the Oracle", "the Fair",
		"of the Winds", "the White", "the Old"},

	// Unholy: gothic names, dark first parts and endings, and grim titles.
	"ustart": {"Raven", "Grim", "Dread", "Black", "Ash", "Bone", "Wolf", "Mourn", "Shade", "Gloam", "Thorn", "Hollow", "Blood",
		"Crow", "Grave", "Night", "Wither", "Dusk", "Morrow", "Vex", "Mor", "Drak", "Vlad", "Stryx", "Kar", "Nos"},
	"uend": {"moor", "wald", "mire", "holt", "vale", "fell", "grave", "keep", "mora", "vania", "stein", "burg", "gard",
		"hollow", "reach", "dun", "skar", "vod"},
	"uplace": {"Woods", "Moors", "Marshes", "Barrows", "Hollows", "Crypts", "Wastes", "Heights", "Gloom", "Thickets", "Graves",
		"Ruins"},
	"uadj": {"Cursed", "Withered", "Silent", "Weeping", "Forsaken", "Shadowed", "Bleak", "Bloodmoon", "Haunted", "Rotting",
		"Black", "Dreadful", "Moonlit", "Accursed"},
	"uperson": {"Count Draven", "Lady Morwen", "Baron Vlakos", "Mother Hesk", "Lord Malgrim", "Countess Szara", "Old Ivo",
		"the Pale Duke", "Abbot Krell", "Margravine Ilsa", "Sir Corvin", "Witch Agnet"},

	"eepithet": {"the Eternal", "the Golden", "the Silent", "the Forgotten", "the Buried", "the Radiant",
		"the Undying", "the Sunborn", "the Veiled", "the Exalted", "the Lost", "the Mighty"},
}

// Patterns per theme: "{list}" picks a word from a list, everything else is literal.
var namePatterns = map[ZoneType][]string{
	// Every pattern contains at least one compound or syllable-built name, so even the
	// smallest pattern has tens of thousands of names.
	Grassland: {
		"{gstart}{gend} {gplace}",
		"{gadj} {gplace} of {gstart}{gend}",
		"{gplace} of {gstart}{gend}",
		"{gadj} {gstart}{gend}",
		"{gperson}'s {gstart}{gend}",
		"{gperson}'s {gadj} {gplace}",
	},
	Desert: {
		"{dplace} Al-{ds1}{ds2}{ds3}",
		"{dplace} {ds1}{ds2}{ds3}",
		"{ds1}{ds2}{ds3} {dplace2}",
		"Al-{ds1}{ds2}{ds3} {dplace2}",
		"{dplace} {ds1}{ds3}-{ds1}{ds3}",
	},
	North: {
		"{nstart}{nend} {nplace}",
		"{nadj} {nplace} of {nstart}{nend}",
		"{nplace} of {nstart}{nend}",
		"{nadj} {nstart}{nend}",
		"{nperson}'s {nstart}{nend}",
		"{nperson}'s {nadj} {nplace}",
	},
	Mediterranean: {
		"{mplace} {ms1}{ms2}{ms3}",
		"{ms1}{ms2}{ms3} {mplace2}",
		"{mplace} {ms1}{ms2}{ms3} {mepithet}",
		"{ms1}{ms3} {mepithet}",
	},
	Unholy: {
		"{ustart}{uend} {uplace}",
		"{uadj} {uplace} of {ustart}{uend}",
		"{uplace} of {ustart}{uend}",
		"{uadj} {ustart}{uend}",
		"{uperson}'s {ustart}{uend}",
	},
	AncientDesert: {
		"{eplace} {es1}{es2}{es3}",
		"{eplace} {es1}{es2}{es3} {eepithet}",
		"{es1}{es2}{es3}'s {dplace2}",
		"{eplace} {es1}{es3}-{es1}{es3}",
	},
}

var nameCache sync.Map // layoutKey → string

// ZoneName returns the name of a zone (unique within uniqueRadius zones).
func ZoneName(seed int64, zx, zy int, t ZoneType) string {
	key := layoutKey{seed, zx, zy}
	if n, ok := nameCache.Load(key); ok {
		return n.(string)
	}
	taken := map[string]bool{}
	for dy := -uniqueRadius; dy <= uniqueRadius; dy++ {
		for dx := -uniqueRadius; dx <= uniqueRadius; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nt := ZoneTypeOf(seed, zx+dx, zy+dy)
			for k := 0; k < candidates; k++ {
				taken[candidateName(seed, zx+dx, zy+dy, nt, k)] = true
			}
		}
	}
	name := candidateName(seed, zx, zy, t, 0)
	for k := 0; k < candidates; k++ {
		if c := candidateName(seed, zx, zy, t, k); !taken[c] {
			name = c
			break
		}
	}
	nameCache.Store(key, name)
	return name
}

// candidateName builds the k-th candidate name of a zone.
func candidateName(seed int64, zx, zy int, t ZoneType, k int) string {
	patterns, ok := namePatterns[t]
	if !ok {
		patterns = namePatterns[Grassland]
	}
	salt := saltName + uint64(k)*1009
	p := patterns[int(rand01(seed, salt, zx, zy)*float64(len(patterns)))]
	var b strings.Builder
	part := 0
	for len(p) > 0 {
		i := strings.IndexByte(p, '{')
		if i < 0 {
			b.WriteString(p)
			break
		}
		b.WriteString(p[:i])
		j := strings.IndexByte(p, '}')
		list := nameLists[p[i+1:j]]
		part++
		b.WriteString(list[int(rand01(seed, salt+uint64(part)*31, zx, zy)*float64(len(list)))])
		p = p[j+1:]
	}
	return b.String()
}
