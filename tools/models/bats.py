"""Builds the bat swarm of the unholy lands: three bats flapping around each other, rigged and
animated as one creature.

    blender -b --python tools/models/bats.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; faces -Y (glTF +Z), about 1 m tall and wide (the game
floats it above the ground). Each bat is a furry body with big ears, red eyes and two wings of
dark membrane on finger bones; every part is bound to one bone (a body and two wings per bat).
Animations: Idle (the bats flap and bob around each other), Walk (faster), Attack (all three
dive forward), HitReact (they scatter) and Death (they tumble to the ground).
"""

import math
import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()

FUR = k.material("Fur", (62, 50, 54))
SKIN = k.material("Skin", (44, 36, 42))
MEMBRANE = k.material("Membrane", (82, 52, 66))
EYE = k.material("Eye", (230, 40, 40))
FANG = k.material("Fang", (236, 230, 214))

# The three bats: where each one hovers and how big it is.
BATS = [((0, 0, 0.62), 1.0), ((-0.32, 0.12, 0.38), 0.78), ((0.3, 0.1, 0.82), 0.72)]


def wing(c, s, size, bone):
    """A wing out to side s: an arm bone and three finger bones with the membrane between."""
    c = Vector(c)
    shoulder = c + Vector((s * 0.05, 0, 0.02)) * size
    wrist = shoulder + Vector((s * 0.2, 0, 0.06)) * size
    tips = [wrist + Vector((s * 0.22, 0, 0.08)) * size, wrist + Vector((s * 0.24, 0, -0.06)) * size,
            wrist + Vector((s * 0.14, 0, -0.17)) * size]
    k.rod(shoulder, wrist, 0.012 * size, SKIN, bone, 5)
    for t in tips:
        k.rod(wrist, t, 0.007 * size, SKIN, bone, 4)
    # The membrane: a fan of thin triangles from the body side to the finger tips.
    root = c + Vector((s * 0.02, 0, -0.08)) * size
    pts = [shoulder, wrist] + tips + [root]
    verts = [tuple(p) for p in pts] + [tuple(p + Vector((0, 0.006, 0))) for p in pts]
    n = len(pts)
    faces = [(1, 2, 3), (1, 3, 4), (1, 4, 5), (0, 1, 5)]
    faces += [tuple(i + n for i in reversed(f)) for f in faces]
    k.mesh("membrane", verts, faces, MEMBRANE, bone, smooth=False)


bones = [("root", (0, 0, 0), None)]
for i, (c, size) in enumerate(BATS):
    c = Vector(c)
    body = f"bat{i}"
    k.ellipsoid(c, (0.06 * size, 0.05 * size, 0.08 * size), FUR, body, 10, 6)
    k.ellipsoid(c + Vector((0, -0.03, 0.09)) * size, (0.045 * size, 0.04 * size, 0.04 * size), FUR, body, 10, 6)  # the head
    for s in (-1, 1):
        k.rod(c + Vector((s * 0.025, -0.02, 0.12)) * size, c + Vector((s * 0.05, -0.01, 0.2)) * size, 0.018 * size, SKIN, body, 5,
              taper=0.2)  # ears
        k.ellipsoid(c + Vector((s * 0.018, -0.068, 0.1)) * size, (0.01 * size, 0.006 * size, 0.01 * size), EYE, body, 6, 4)
        k.rod(c + Vector((s * 0.01, -0.07, 0.07)) * size, c + Vector((s * 0.01, -0.07, 0.05)) * size, 0.005 * size, FANG, body, 4,
              taper=0.2)
        wing(c, s, size, f"wing{i}_{s}")
    bones.append((body, tuple(c), "root"))
    for s in (-1, 1):
        bones.append((f"wing{i}_{s}", tuple(c + Vector((s * 0.05, 0, 0.02)) * size), body))
k.bones(bones)
k.bind("BatSwarm")
key = k.key


def flap(frame, i, up):
    """up: 1 with the wings raised high, -1 swept down."""
    for s in (-1, 1):
        key(frame, f"wing{i}_{s}", rot=(0, s * 55 * up, 0))


def hover(length, beats, bob=0.05):
    for i in range(len(BATS)):
        steps = beats * 2
        for j in range(steps + 1):
            f = 1 + round((length - 1) * j / steps)
            flap(f, i, 1 if (j + i) % 2 == 0 else -1)
        for j, f in enumerate((1, length // 3, 2 * length // 3, length)):
            a = (j + i * 1.3) * math.pi / 1.5
            key(f, f"bat{i}", loc=(math.cos(a) * 0.04, (0.5 + 0.5 * math.sin(a)) * bob, math.sin(a) * 0.03))


def idle():
    hover(49, 6)


def walk():
    hover(25, 5, 0.03)


def attack():
    for i in range(len(BATS)):
        flap(1, i, 1)
        flap(5, i, -1)
        flap(9, i, 1)
        key(1, f"bat{i}", rot=(0, 0, 0), loc=(0, 0, 0))
        key(8, f"bat{i}", rot=(40, 0, 0), loc=(0, -0.05, 0.1))  # wings back, then a dive forward
        key(14, f"bat{i}", rot=(55, 0, 0), loc=(0, -0.12, 0.55))
        flap(14, i, -1)
        key(22, f"bat{i}", rot=(0, 0, 0), loc=(0, 0, 0))
        flap(22, i, 1)


def hit():
    for i, (c, _) in enumerate(BATS):
        d = Vector(c).normalized() if Vector(c).length > 0.65 else Vector((0, 0, 1))
        key(5, f"bat{i}", rot=(-20, 0, 20 * (i - 1)), loc=(d.x * 0.1, 0.05, -d.y * 0.1))
        flap(5, i, 1)


def death():
    for i, (c, _) in enumerate(BATS):
        flap(4, i, 1)
        flap(10, i, -0.3)
        key(4, f"bat{i}", rot=(0, 0, 0), loc=(0, 0, 0))
        key(26, f"bat{i}", rot=(80, 30 * (i - 1), 90), loc=(0, -Vector(c).z + 0.03, 0))  # tumbling down onto the ground
        flap(26, i, 0.2)


k.action("Idle", 49, idle)
k.action("Walk", 25, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 26, death)
k.export(args[0])
if len(args) > 1 and args[1]:
    k.preview(args[1], distance=2.6, target_z=0.55)
