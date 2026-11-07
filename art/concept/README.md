# Concept art

Favorite images from the Stable Diffusion runs (`tools/artgen`, SDXL Lightning), kept as the
style reference for the 3D models. They aren't used by the game.

| Images | What | Notes |
|---|---|---|
| `knight_s51_down/side/up.png` | The warrior as a plate knight (seed 51) | The preferred look: steel plate, red cloak and scarf, a great helm with a crest point, layered pauldrons, mail sleeves, a red sash, fanned knee guards, a longsword |
| `knight_s2_down/side/up.png` | The same knight (seed 2) | Bolder colors: a blue cloak and tabard, gold trim, a big red and blue heater shield |
| `mage_s51.png` | The mage | A violet hooded cloak open over a light robe, an embroidered front panel and cuffs, a big gold belt buckle, a long white beard |
| `warlock_s51.png` | The warlock | A magenta cloak and deep hood over a bone skull mask, ragged grey wrappings, red belts with a brass buckle, bound forearms |
| `rogue_s51.png` | The rogue | A green hood and mask, a brown cape over the shoulders, a ragged green tunic, belts and pouches, steel bracers, a red strip at the hip |
| `bandit_s51.png` | The bandit (an enemy) | An ochre hood and mask, a long rust coat open over a quilted olive tunic, belts and pouches, baggy trousers, tall boots |
| `nomad_s51.png` | The nomad raider (an enemy) | A red hood, mask and cape over a cream tunic, a red sash, steel bracers, dark baggy trousers, a scimitar |
| `berserker_s51.png` | The berserker (an enemy of the North) | Bare-chested with crossed straps, a fur pelt over the shoulders, wild red hair and beard, a green kilt and a red sash, fur-wrapped legs |
| `troll_s51.png` | The ice troll (an enemy of the North) | Pale blue-grey and long-armed, long white hair, a fringed fur loincloth (the model is stockier, with ice on its shoulders) |
| `corsair_s51.png` | The corsair (an enemy of the Mediterranean) | A tricorn, a long red coat lined in teal with big cuffs, a red neck scarf, a bandolier, olive breeches, tall boots |
| `hoplite_s51.png` | The hoplite (an enemy of the Mediterranean) | A bronze Corinthian helm with a red crest, a bronze cuirass, a skirt of leather strips, greaves, a red cloak, spear and round shield |
| `vampire_s51.png` | The vampire (an enemy of the unholy lands) | Pale with long black hair, a tall stiff white collar, a red cape lined in white, a red waistcoat, a cream coat |
| `cultist_s51.png` | The cultist (an enemy of the unholy lands) | A crimson hooded cloak over a grey robe, a pale mask, chains, bound forearms (the model's mask is blank, to tell it from the warlock's skull) |
| `shaman_s51.png` | The shaman | A great pale fur collar, a rust-red coat open over a hide tunic, a medallion, long brown hair and beard |

Both knight sets were made from the old 2D map sprites' poses (front, side, back), so the
three views of a set show the same design. The other classes and the human enemies are front views made the same way
from `classes.json`:

    python3 tools/artgen/artgen.py art/concept/classes.json --try 51

which writes them to `art/concept/.work/candidates/` (not kept). The paladin's run gave nothing
usable, so its model is the knight's plate in white and gold.
