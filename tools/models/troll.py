"""Builds the ice troll: a huge, hunched brute of the North with long heavy arms, rigged and
animated.

    blender -b --python tools/models/troll.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; faces -Y (glTF +Z). Modelled about 1.3 m tall, like
the goblin (the game scales it up). Pale blue-grey skin, a barrel chest and belly, a small
head low between the shoulders with a heavy brow, a big nose and tusks, shaggy white hair down
its back and over its shoulders, chunks of ice grown on its shoulders and forearms, a ragged
fur loincloth, huge bare feet. The arms hang to its knees and end in big fists. Same rig as
the goblin. Every part is bound to one bone. Animations: Idle (breathes, shifts its weight), Walk (a heavy, rolling
lumber on its knuckles' swing), Attack (raises both fists and smashes them down), HitReact and
Death (topples forward).
"""

import math
import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()

SKIN = k.material("Skin", (150, 170, 186))
SKIN_DARK = k.material("SkinDark", (112, 132, 152))
HAIR = k.material("Hair", (232, 236, 240))
HAIR_DARK = k.material("HairDark", (190, 198, 208))
FUR = k.material("Fur", (120, 96, 72))
FUR_DARK = k.material("FurDark", (84, 66, 50))
ICE = k.material("Ice", (180, 222, 245), metal=True)
ICE_DEEP = k.material("IceDeep", (120, 176, 218), metal=True)
EYE = k.material("Eye", (255, 214, 90))
PUPIL = k.material("Pupil", (25, 15, 10))
TOOTH = k.material("Tooth", (236, 228, 200))
MOUTH = k.material("Mouth", (60, 30, 40))
NAIL = k.material("Nail", (70, 70, 80))


def ice_cluster(center, out, size, bone, count=4):
    """A few ice crystals jutting out of the skin along out."""
    c, o = Vector(center), Vector(out).normalized()
    side = o.cross(Vector((0, 0, 1)))
    if side.length < 0.1:
        side = Vector((1, 0, 0))
    side.normalize()
    up = side.cross(o).normalized()
    for i in range(count):
        a = i * 2 * math.pi / count + 0.4
        d = (o * 1.0 + side * math.cos(a) * 0.45 + up * math.sin(a) * 0.45).normalized()
        k.shard(c + (side * math.cos(a) + up * math.sin(a)) * size * 0.3, d, size * (0.28 + 0.06 * (i % 2)), size * (1.0 + 0.4 * (i % 3)),
                ICE if i % 2 else ICE_DEEP, bone, flat=side)


# Torso: a big belly (hips), a barrel chest leaning forward (spine), the shoulders a hump.
k.loft("belly", [((0, 0.02, 0.36), 0.15, 0.13), ((0, -0.04, 0.46), 0.22, 0.21), ((0, -0.03, 0.57), 0.21, 0.19)], SKIN, "hips", n=18,
       power=2.1)
# The chest starts where the belly ends, as wide and deep, so the two read as one body.
k.loft("chest", [((0, -0.03, 0.55), 0.215, 0.2), ((0, -0.01, 0.66), 0.25, 0.2), ((0, 0.03, 0.78), 0.28, 0.18),
                 ((0, 0.06, 0.87), 0.22, 0.14), ((0, 0.07, 0.92), 0.1, 0.08)], SKIN, "spine", n=18, power=2.2)
k.ellipsoid((0, 0.12, 0.84), (0.24, 0.12, 0.12), SKIN, "spine", 14, 7)  # the hump between the shoulders
# Hair: shaggy white locks from the crown down the back.
for i in range(13):
    a = math.pi * (0.1 + 0.8 * i / 12)
    p = Vector((math.cos(a) * 0.16, 0.12 + math.sin(a) * 0.08, 0.95))
    k.shard(p, Vector((math.cos(a) * 0.25, 0.4 + math.sin(a) * 0.3, -1)), 0.05, 0.26 + 0.06 * (i % 3), HAIR if i % 2 else HAIR_DARK,
            "spine", flat=(math.cos(a), math.sin(a), 0))
# Long white hair from the head, falling over the shoulders and down the upper chest.
for i, (x, l) in enumerate(((-0.17, 0.2), (-0.12, 0.26), (-0.07, 0.18), (0.07, 0.19), (0.12, 0.27), (0.17, 0.21))):
    top = Vector((x * 0.7, -0.08, 0.95))
    k.loft(f"lock{i}", [(top, 0.03, 0.02), (Vector((x, -0.15 + abs(x) * 0.2, 0.86)), 0.032, 0.016),
                        (Vector((x * 1.05, -0.17 + abs(x) * 0.25, 0.92 - l)), 0.004, 0.004)], HAIR if i % 2 else HAIR_DARK, "spine",
           n=8, power=2.4, side=(1, 0, 0))
# Loincloth: a fur belt and a ragged flap front and back.
k.loft("fur_belt", [((0, -0.01, 0.42), 0.215, 0.195), ((0, -0.015, 0.36), 0.2, 0.18)], FUR_DARK, "hips", n=18, power=2.1)
for i in range(16):
    a = i * 2 * math.pi / 16
    p = Vector((math.cos(a) * 0.2, -0.01 + math.sin(a) * 0.18, 0.37))
    k.shard(p, Vector((math.cos(a) * 0.15, math.sin(a) * 0.15, -1)), 0.06, 0.1 + 0.05 * (i % 3), FUR if i % 2 else FUR_DARK, "hips",
            flat=(math.cos(a), math.sin(a), 0))

# Head: small, low and forward, a heavy brow, a big drooping nose, tusks from the underbite.
head = [(0.86, 0.07, 0.07, -0.1), (0.9, 0.11, 0.1, -0.08), (0.96, 0.125, 0.115, -0.06), (1.01, 0.12, 0.11, -0.05),
        (1.05, 0.09, 0.085, -0.04), (1.07, 0.04, 0.04, -0.035)]
k.loft("head", [((0, y, z), rx, rz) for z, rx, rz, y in head], SKIN, "head", n=20, power=2.2, keel=0.85)
k.loft("jaw", [((0, -0.1, 0.89), 0.12, 0.06), ((0, -0.17, 0.875), 0.1, 0.05), ((0, -0.2, 0.88), 0.07, 0.035)], SKIN, "head",
       n=14, power=2.3, side=(1, 0, 0))
k.ellipsoid((0, -0.2, 0.9), (0.07, 0.012, 0.012), MOUTH, "head", 10, 4)
k.loft("brow", [((-0.1, -0.15, 1.0), 0.03, 0.022), ((0, -0.165, 1.005), 0.035, 0.026), ((0.1, -0.15, 1.0), 0.03, 0.022)],
       SKIN_DARK, "head", n=8, side=(0, 1, 0))
k.loft("nose", [((0, -0.17, 0.99), 0.035, 0.03), ((0, -0.22, 0.96), 0.045, 0.04), ((0, -0.235, 0.925), 0.04, 0.035)], SKIN_DARK,
       "head", n=10, side=(1, 0, 0))
for s in (-1, 1):
    k.rod((s * 0.055, -0.2, 0.885), (s * 0.07, -0.215, 0.965), 0.018, TOOTH, "head", 6, taper=0.2)  # tusks
    k.ellipsoid((s * 0.05, -0.16, 0.975), (0.024, 0.014, 0.016), EYE, "head", 8, 5)
    k.ellipsoid((s * 0.051, -0.172, 0.975), (0.009, 0.006, 0.011), PUPIL, "head", 6, 4)
    k.ellipsoid((s * 0.12, -0.05, 0.98), (0.03, 0.02, 0.04), SKIN_DARK, "head", 8, 5)  # small ears
for x, y in ((-0.04, -0.02), (0.03, 0.0), (0.0, 0.03), (-0.06, 0.04), (0.06, 0.03)):  # a shock of white hair on top
    k.shard((x, y - 0.04, 1.06), Vector((x * 4, 0.8 + y * 4, 1)), 0.02, 0.1, HAIR, "head", flat=(0, 1, 0))

# Arms: massive, hanging to the knees, ice on the shoulders and forearms, huge fists.
for s in (-1, 1):
    shoulder, elbow, hand = Vector((s * 0.3, 0.04, 0.8)), Vector((s * 0.38, 0.04, 0.52)), Vector((s * 0.4, -0.06, 0.24))
    k.ellipsoid(shoulder, (0.12, 0.12, 0.11), SKIN, f"arm_{s}", 12, 7)
    k.loft(f"upper{s}", [(shoulder, 0.1, 0.1), (shoulder.lerp(elbow, 0.5), 0.085, 0.085), (elbow, 0.07, 0.07)], SKIN, f"arm_{s}",
           n=12, power=2.0)
    ice_cluster(shoulder + Vector((s * 0.04, 0.03, 0.08)), (s * 0.5, 0.2, 1), 0.13, f"arm_{s}", 5)
    k.ellipsoid(elbow, (0.075, 0.075, 0.075), SKIN_DARK, f"fore_{s}", 10, 6)
    k.loft(f"fore{s}", [(elbow, 0.07, 0.07), (elbow.lerp(hand, 0.55), 0.09, 0.085), (hand, 0.075, 0.07)], SKIN, f"fore_{s}", n=12)
    ice_cluster(elbow.lerp(hand, 0.45) + Vector((s * 0.07, 0.03, 0)), (s * 1, 0.4, 0.2), 0.08, f"fore_{s}", 3)
    fist = hand + Vector((0, -0.02, -0.06))
    k.ellipsoid(fist, (0.1, 0.1, 0.09), SKIN, f"fore_{s}", 12, 7)
    for t in range(4):  # knuckles and claws
        x = fist.x + (t - 1.5) * 0.04
        kn = Vector((x, fist.y - 0.08, fist.z + 0.02))
        k.ellipsoid(kn, (0.028, 0.028, 0.028), SKIN_DARK, f"fore_{s}", 6, 4)
        k.rod(kn + Vector((0, -0.01, -0.02)), kn + Vector((0, -0.02, -0.05)), 0.01, NAIL, f"fore_{s}", 4, taper=0.2)
    k.rod(fist + Vector((s * -0.07, -0.06, 0.03)), fist + Vector((s * -0.1, -0.09, 0.0)), 0.025, SKIN, f"fore_{s}", 6, taper=0.7)  # thumb
    # Legs: short, thick and bowed, huge flat feet with three toes.
    hip, knee, ankle = Vector((s * 0.13, 0, 0.38)), Vector((s * 0.17, -0.04, 0.2)), Vector((s * 0.16, 0, 0.06))
    k.loft(f"thigh{s}", [(hip, 0.1, 0.1), (hip.lerp(knee, 0.5), 0.095, 0.095), (knee, 0.075, 0.075)], SKIN, f"thigh_{s}", n=12)
    k.ellipsoid(knee, (0.08, 0.08, 0.08), SKIN_DARK, f"shin_{s}", 10, 6)
    k.loft(f"shin{s}", [(knee, 0.075, 0.075), (knee.lerp(ankle, 0.5), 0.08, 0.08), (ankle, 0.065, 0.065)], SKIN, f"shin_{s}", n=12)
    k.loft(f"foot{s}", [((s * 0.16, 0.06, 0.05), 0.07, 0.05), ((s * 0.16, -0.04, 0.045), 0.1, 0.05),
                        ((s * 0.165, -0.15, 0.03), 0.11, 0.035)], SKIN, f"shin_{s}", n=12, power=2.4, side=(1, 0, 0))
    for t in range(3):
        a = (t - 1) * 0.4
        base = Vector((s * 0.165 + math.sin(a) * 0.05, -0.17, 0.03))
        tip = base + Vector((math.sin(a) * 0.03, -0.05, -0.01))
        k.rod(base, tip, 0.028, SKIN, f"shin_{s}", 6, taper=0.7)
        k.rod(tip, tip + Vector((0, -0.02, -0.015)), 0.014, NAIL, f"shin_{s}", 4, taper=0.1)

bones = [("root", (0, 0, 0), None), ("hips", (0, 0, 0.42), "root"), ("spine", (0, 0, 0.52), "hips"),
         ("head", (0, -0.04, 0.86), "spine")]
for s in (-1, 1):
    bones += [(f"arm_{s}", (s * 0.3, 0.04, 0.8), "spine"), (f"fore_{s}", (s * 0.38, 0.04, 0.52), f"arm_{s}"),
              (f"thigh_{s}", (s * 0.13, 0, 0.38), "hips"), (f"shin_{s}", (s * 0.17, -0.04, 0.2), f"thigh_{s}")]
k.bones(bones)
k.bind("IceTroll")
key = k.key
HUNCH = 18  # the spine leans forward this much at rest


def pose(f, spine=0.0, head=0.0, turn=0.0):
    key(f, "spine", rot=(HUNCH + spine, turn, 0))
    key(f, "head", rot=(-HUNCH + head, -turn * 0.5, 0))


def arms(f, a=0.0, b=0.0, spread=8.0):
    for s in (-1, 1):
        key(f, f"arm_{s}", rot=(a, 0, s * spread))
        key(f, f"fore_{s}", rot=(b, 0, 0))


def idle():
    for f, w in ((1, 0), (30, 1), (60, 0), (90, -1), (120, 0)):  # slow breathing and weight shifts
        key(f, "hips", rot=(0, 0, 2 * w))
        key(f, "root", loc=(0, 0, -0.01 * abs(w)))
        pose(f, 3 * abs(w), 2 * w, 4 * w)
        arms(f, -4 - 3 * abs(w), -6, 8 + 2 * abs(w))


def walk():
    for f, ph in ((1, 1), (11, 0), (21, -1), (31, 0), (41, 1)):
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(-24 * ph * s, 0, s * 4))
            key(f, f"shin_{s}", rot=(22 * max(0, ph * s), 0, 0))
            key(f, f"arm_{s}", rot=(22 * ph * s, 0, s * 10))
            key(f, f"fore_{s}", rot=(-10, 0, 0))
        key(f, "hips", rot=(0, 5 * ph, -6 * ph), loc=(0, 0, -0.025 * (1 - abs(ph))))
        pose(f, 6, -4, -8 * ph)


def attack():
    # Both fists raised high, a lunge, then a double hammer blow down onto the ground.
    arms(8, -165, -40, 15)
    pose(8, -20, 12)
    key(8, "root", loc=(0, 0.02, 0.02))
    arms(14, -40, -10, 6)
    pose(14, 32, -15)
    key(14, "root", loc=(0, 0.12, 0.25))
    arms(20, -30, -15, 6)
    pose(20, 28, -10)
    key(20, "root", loc=(0, 0, 0.22))
    for f in (8, 14):
        key(f, "thigh_1", rot=(-30 if f == 14 else 0, 0, 0))
        key(f, "shin_1", rot=(25 if f == 14 else 0, 0, 0))


def hit():
    pose(5, -18, -15, 8)
    key(5, "root", loc=(0, 0, -0.08))
    arms(5, -30, -20, 25)


def death():
    pose(8, -15, -20)
    arms(8, -40, -20, 20)
    key(8, "root", loc=(0, 0, -0.05))
    for f in (26, 44):  # topples forward onto its face
        key(f, "root", rot=(85, 0, 0), loc=(0, -0.15, -0.2))
        pose(f, 10, 15)
        arms(f, -150, -10, 30)
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(10, 0, s * 10))
            key(f, f"shin_{s}", rot=(20, 0, 0))


k.action("Idle", 121, idle)
k.action("Walk", 41, walk)
k.action("Attack", 26, attack)
k.action("HitReact", 18, hit)
k.action("Death", 44, death)
k.export(args[0])
if len(args) > 1:
    k.preview(args[1], distance=3.0, target_z=0.6)
