"""Builds the ember spirit: a living flame with a coal-dark face, little flame arms and sparks
circling it, rigged and animated.

    blender -b --python tools/models/ember.py -- OUT.gltf [PREVIEW_DIR] [frost|wind]

Runs inside Blender without a window; faces -Y (glTF +Z), about 1.2 m from the wisp at the
bottom to the tip of the flame (the game floats it above the ground). Every part is bound to
one bone; the flames flicker by scaling. Animations: Idle (flickers, sparks circle), Walk
(leans in, flames trail), Attack (swells and throws itself forward), HitReact and Death
(gutters out to a spark). With "frost", the same spirit in ice blue and white: the North's
frost wisp, with a dark blue face; with "wind", in pale teal and white: the Mediterranean's
wind spirit.
"""

import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()

variant = args[2] if len(args) > 2 else "ember"
# The flame's colors from the outside in, its face and its sparks, per variant.
COLORS = {
    "ember": [(235, 85, 35), (255, 150, 45), (255, 225, 110), (45, 25, 20), (255, 245, 180)],
    "frost": [(90, 150, 220), (150, 205, 245), (225, 245, 255), (20, 30, 60), (240, 250, 255)],
    "wind": [(120, 200, 190), (175, 228, 220), (236, 250, 246), (34, 84, 90), (250, 255, 252)],
}[variant]
RED = k.material("Red", COLORS[0])
ORANGE = k.material("Orange", COLORS[1])
YELLOW = k.material("Yellow", COLORS[2])
COAL = k.material("Coal", COLORS[3])
SPARK = k.material("Spark", COLORS[4])

# Body: a round flame narrowing to a wisp at the bottom, lighter at the heart.
k.ellipsoid((0, 0, 0.42), (0.25, 0.24, 0.26), ORANGE, "core")
k.rod((0, 0, 0.3), (0, 0.03, 0.0), 0.19, ORANGE, "core", 12, taper=0.05)
k.ellipsoid((0, -0.09, 0.4), (0.18, 0.16, 0.19), YELLOW, "core")
# Face: coal eyes with a glint, a grinning mouth.
for s in (-1, 1):
    k.ellipsoid((s * 0.08, -0.22, 0.48), (0.04, 0.03, 0.055), COAL, "core", 10, 6)
    k.ellipsoid((s * 0.075, -0.245, 0.5), (0.012, 0.01, 0.014), SPARK, "core", 6, 4)
k.ellipsoid((0, -0.235, 0.36), (0.07, 0.025, 0.025), COAL, "core", 10, 6)

# Flames: a tall middle one, one to each side and one behind. Each is a teardrop that bends
# toward its tip, with a lighter inside.
FLAMES = {
    "flame_m": ((0, 0.0, 0.5), (0.03, 0.08, 1.2), 0.21, (0.06, 0.0, 0.0)),
    "flame_l": ((-0.14, 0.01, 0.48), (-0.3, 0.1, 0.9), 0.12, (0.02, -0.02, 0.0)),
    "flame_r": ((0.14, 0.01, 0.48), (0.3, 0.1, 0.86), 0.12, (-0.02, -0.02, 0.0)),
    "flame_b": ((0, 0.12, 0.46), (0, 0.32, 0.92), 0.14, (0.0, -0.04, 0.0)),
}


def teardrop(a, b, r, bend, mat, bone):
    a, b, bend = Vector(a), Vector(b), Vector(bend)
    base = a + (b - a) * 0.2
    mid = a + (b - a) * 0.6 + bend
    k.ellipsoid(base, (r, r, r * 1.1), mat, bone, 12, 8)
    k.rod(base, mid, r, mat, bone, 12, taper=0.55)
    k.ellipsoid(mid, (r * 0.55, r * 0.55, r * 0.55), mat, bone, 10, 6)
    k.rod(mid, b, r * 0.55, mat, bone, 10, taper=0.03)


for bone, (a, b, r, bend) in FLAMES.items():
    teardrop(a, b, r, bend, RED, bone)
    f = Vector((0, -r * 0.6, 0))
    teardrop(Vector(a) + f, Vector(a) + (Vector(b) - Vector(a)) * 0.65 + f, r * 0.62, bend, YELLOW, bone)

for s in (-1, 1):  # arms: small flames reaching out and forward, a round hand at the end
    k.rod((s * 0.2, -0.04, 0.36), (s * 0.38, -0.14, 0.3), 0.07, ORANGE, f"arm_{s}", 10, taper=0.7)
    k.ellipsoid((s * 0.4, -0.15, 0.3), (0.06, 0.06, 0.06), YELLOW, f"arm_{s}", 10, 6)

for i, (x, y, z) in enumerate(((0.45, 0.0, 0.35), (-0.3, 0.32, 0.6), (-0.22, -0.38, 0.75))):
    k.ellipsoid((x, y, z), (0.035, 0.035, 0.035), SPARK, "orbit", 8, 5)  # sparks circling

bones = [("root", (0, 0, 0), None), ("core", (0, 0, 0.3), "root"), ("orbit", (0, 0, 0.45), "root")]
bones += [(name, spec[0], "core") for name, spec in FLAMES.items()]
bones += [(f"arm_{s}", (s * 0.2, -0.04, 0.36), "core") for s in (-1, 1)]
k.bones(bones)
k.bind({"ember": "EmberSpirit", "frost": "FrostWisp", "wind": "WindSpirit"}[variant])
key = k.key


def flames(frame, lean=0.0, size=1.0, wave=0.0):
    """lean: degrees forward (negative trails back); size: scale; wave: sideways flicker."""
    for i, name in enumerate(FLAMES):
        w = wave * (1 if i % 2 else -1)
        key(frame, name, rot=(lean, 0, w), scale=size * (1 + 0.08 * (i % 2) * (1 if wave >= 0 else -1)))


def arms(frame, reach=0.0, lift=0.0):
    for s in (-1, 1):
        key(frame, f"arm_{s}", rot=(-reach, 0, s * lift))


def orbit(frames, turns=1.0):
    """Sparks go round at a steady speed over the action (keys every 30 degrees)."""
    n = 12
    for i in range(n + 1):
        key(1 + round((frames - 1) * i / n), "orbit", rot=(0, 360 * turns * i / n, 0))


def idle():
    for f, w, sz in ((1, 0, 1.0), (11, 6, 1.06), (21, -5, 0.96), (31, 4, 1.05), (41, -6, 0.97), (51, 5, 1.04), (61, 0, 1.0)):
        flames(f, 0, sz, w)
        key(f, "core", scale=1 + (sz - 1) * 0.4, loc=(0, (sz - 1) * 0.3, 0))
    for f, l in ((1, 0), (30, 12), (61, 0)):
        arms(f, 0, l)
    orbit(61)


def walk():
    for f, w in ((1, 0), (7, 5), (13, 0), (19, -5), (25, 0)):
        key(f, "core", rot=(12, 0, w * 0.5))
        flames(f, -20, 1.05, w)  # trailing behind
        arms(f, -15, 10)
    orbit(25)


def attack():
    key(7, "core", rot=(-12, 0, 0), loc=(0, 0, -0.08), scale=1.12)  # draws back and swells
    flames(7, -10, 1.15, 4)
    arms(7, 40, 30)
    key(12, "core", rot=(18, 0, 0), loc=(0, 0, 0.3), scale=1.05)  # throws itself forward
    flames(12, 10, 1.25, -6)
    arms(12, 70, 0)
    key(18, "core", rot=(8, 0, 0), loc=(0, 0, 0.18))
    flames(18, 5, 1.15)
    orbit(24, 2)


def hit():
    key(4, "core", rot=(-15, 0, 8), loc=(0, 0, -0.12), scale=0.9)
    flames(4, -25, 0.7, 10)
    arms(4, -20, 40)
    flames(10, -10, 1.1, -5)


def death():
    flames(8, -10, 1.3, 10)  # one last flare
    key(8, "core", scale=1.1)
    flames(20, 0, 0.4, -8)
    key(20, "core", scale=0.7, loc=(0, -0.1, 0))
    key(36, "root", scale=0.08, loc=(0, -0.45, 0))  # gutters out to a spark on the ground
    key(20, "root", scale=0.6, loc=(0, -0.2, 0))
    flames(36, 0, 0.2)
    arms(20, 0, -30)
    key(36, "orbit", rot=(0, 200, 0), scale=0.05)
    key(16, "orbit", rot=(0, 120, 0), scale=0.6)


k.action("Idle", 61, idle)
k.action("Walk", 25, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 16, hit)
k.action("Death", 36, death)
k.export(args[0])
if len(args) > 1 and args[1]:
    k.preview(args[1], distance=3.2, target_z=0.55)
