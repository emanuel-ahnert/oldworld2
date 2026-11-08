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

import bpy
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
    """A wing out to side s: an arm bone to the wrist, four long finger bones, and the membrane
    stretched between them, its trailing edge scalloped between the finger tips like a real
    bat's. Forward is -Y."""
    c = Vector(c)
    shoulder = c + Vector((s * 0.05, -0.01, 0.03)) * size
    wrist = shoulder + Vector((s * 0.19, -0.03, 0.07)) * size
    tips = [wrist + Vector(v) * size for v in
            ((s * 0.21, 0.0, 0.07), (s * 0.25, 0.05, -0.03), (s * 0.18, 0.09, -0.12), (s * 0.07, 0.09, -0.17))]
    root = c + Vector((s * 0.03, 0.04, -0.06)) * size
    k.rod(shoulder, wrist, 0.011 * size, SKIN, bone, 5, taper=0.7)
    for t in tips:
        k.rod(wrist, t, 0.006 * size, SKIN, bone, 4, taper=0.4)
    k.rod(wrist, wrist + Vector((s * 0.01, -0.03, 0.02)) * size, 0.006 * size, SKIN, bone, 4, taper=0.2)  # the thumb claw

    # The membrane's outline from the wrist: the leading finger, then each finger tip with a
    # scallop (a point pulled back toward the wrist) before the next, and in to the body.
    def scallop(a, b):
        mid = (a + b) / 2
        return mid + (wrist - mid) * 0.28
    edge = [tips[0]]
    for a, b in zip(tips, tips[1:]):
        edge += [scallop(a, b), b]
    edge += [scallop(tips[-1], root), root, shoulder]
    verts = [wrist] + edge
    faces = [(0, i, i + 1) for i in range(1, len(edge))]
    if s < 0:
        faces = [tuple(reversed(f)) for f in faces]
    # Both sides, a hair apart, so the membrane shows from above and below.
    n = len(verts)
    back = [v + Vector((0, 0, -0.003)) for v in verts]
    faces += [tuple(i + n for i in reversed(f)) for f in faces]
    k.mesh("membrane", verts + back, faces, MEMBRANE, bone, smooth=False)


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

# The game scales the swarm to HEIGHT and floats it HOVER above the ground (internal/game/
# scene.go), so the ground lies this far below the model's origin, in model units: where the
# bats come to rest when they die.
HOVER, HEIGHT = 0.3, 1.0
_z = [(o.matrix_world @ v.co).z for o in bpy.data.objects if o.type == "MESH" for v in o.data.vertices]
GROUND = -HOVER * (max(_z) - min(_z)) / HEIGHT
key = k.key


def flap(frame, i, up):
    """up: 1 with the wings raised high, -1 swept down. The wing bones point up, so the beat is
    a turn about their Z axis (the body's forward axis)."""
    for s in (-1, 1):
        key(frame, f"wing{i}_{s}", rot=(0, 0, s * (50 if up > 0 else 40) * up))


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
    """One bat after another: a jolt with the wings thrown up, a last weak beat, then a tumble
    down onto the ground, where it lands belly down with its wings spread flat, bounces once
    and lies still. The first one twitches a wing at the end."""
    end = DEATH_FRAMES
    for i, (c, size) in enumerate(BATS):
        bat, z, d = f"bat{i}", Vector(c).z, i * 3
        yaw = 25 * (i - 1)  # they come to rest turned a little apart
        lie = GROUND - z + 0.05 * size  # down to the ground, the body's half depth above it
        key(1, bat)
        flap(1, i, 0.3)
        key(3 + d, bat, rot=(-25, 0, 0), loc=(0, 0.04, 0))
        flap(3 + d, i, 1)
        flap(7 + d, i, -0.8)
        key(9 + d, bat, rot=(30, yaw * 0.4, 0), loc=(0, lie * 0.35, 0))
        flap(11 + d, i, 0.5)
        flap(15 + d, i, -0.3)
        key(17 + d, bat, rot=(88, yaw, 0), loc=(0, lie, 0))
        flap(17 + d, i, -0.15)
        key(20 + d, bat, rot=(82, yaw, 0), loc=(0, lie + 0.03, 0))
        key(23 + d, bat, rot=(90, yaw, 0), loc=(0, lie, 0))
        flap(23 + d, i, -0.2)
        key(end, bat, rot=(90, yaw, 0), loc=(0, lie, 0))
        if i == 0:
            flap(28, i, 0.25)
            flap(31, i, -0.2)
        flap(end, i, -0.2)


DEATH_FRAMES = 36


k.action("Idle", 49, idle)
k.action("Walk", 25, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", DEATH_FRAMES, death)
k.export(args[0])
if len(args) > 1 and args[1]:
    k.preview(args[1], distance=2.6, target_z=0.55)
