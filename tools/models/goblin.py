"""Builds the goblin: a small, hunched green raider with a spiked club, rigged and animated.

    blender -b --python tools/models/goblin.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; faces -Y (glTF +Z), about 1.1 m tall to the ear tips.
A big head with long drooping ears, a hooked nose and an underbite; a ragged leather tunic,
a belt with a pouch of sling stones, wrapped feet. The club, in the right hand, is a knobbly
branch studded with nails. Every part is bound to one bone. Animations: Idle (shifts its
weight, scratches with the free hand), Walk (a scurrying, bow-legged gait), Attack (swings the
club over its head and down), HitReact and Death (falls onto its back).
"""

import math
import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()

SKIN = k.material("Skin", (125, 160, 75))
SKIN_DARK = k.material("SkinDark", (95, 125, 55))
LEATHER = k.material("Leather", (110, 75, 45))
LEATHER_DARK = k.material("LeatherDark", (75, 50, 32))
CLOTH = k.material("Cloth", (150, 130, 95))
WOOD = k.material("Wood", (105, 72, 45))
IRON = k.material("Iron", (120, 120, 125))
EYE = k.material("Eye", (250, 220, 60))
PUPIL = k.material("Pupil", (25, 15, 10))
TOOTH = k.material("Tooth", (235, 225, 190))
MOUTH = k.material("Mouth", (70, 30, 30))

WART = k.material("Wart", (105, 140, 60))
NAIL = k.material("Nail", (60, 50, 40))


def tri(a, b, tip, thick, mat, bone):
    """A flat triangle with some thickness (ears)."""
    a, b, tip = Vector(a), Vector(b), Vector(tip)
    n = (b - a).cross(tip - a).normalized() * thick
    k.mesh("tri", [a, b, tip, a + n, b + n, tip + n * 0.5], [(0, 1, 2), (5, 4, 3), (0, 3, 4, 1), (1, 4, 5, 2), (2, 5, 3, 0)],
           mat, bone, smooth=False)


# Torso: a pot belly (hips) and a narrow, hunched chest (spine), under a ragged tunic.
k.loft("belly", [((0, 0.0, 0.35), 0.11, 0.1), ((0, -0.03, 0.44), 0.15, 0.15), ((0, -0.02, 0.54), 0.14, 0.13)], SKIN, "hips", n=16,
       power=2.1)
k.loft("chest", [((0, -0.01, 0.53), 0.13, 0.115), ((0, 0.01, 0.64), 0.155, 0.11), ((0, 0.035, 0.73), 0.17, 0.095),
                 ((0, 0.04, 0.78), 0.07, 0.06)], SKIN, "spine", n=16, power=2.2)
k.loft("tunic_low", [((0, 0.0, 0.33), 0.14, 0.13), ((0, -0.03, 0.44), 0.163, 0.163), ((0, -0.02, 0.54), 0.152, 0.143)], CLOTH, "hips",
       n=16, power=2.1)
k.loft("tunic_up", [((0, -0.01, 0.53), 0.143, 0.127), ((0, 0.01, 0.64), 0.167, 0.122), ((0, 0.035, 0.72), 0.17, 0.1)], CLOTH, "spine",
       n=16, power=2.2)
for i in range(11):  # its ragged hem
    a = i * 2 * math.pi / 11 + 0.2
    p = Vector((math.cos(a) * 0.14, math.sin(a) * 0.13, 0.34))
    k.shard(p, Vector((math.cos(a) * 0.3, math.sin(a) * 0.3, -1)), 0.04, 0.07 + 0.03 * (i % 3), CLOTH, "hips", flat=(0, 0, 1))
k.loft("belt", [((0, -0.025, 0.47), 0.168, 0.165), ((0, -0.028, 0.43), 0.17, 0.168)], LEATHER_DARK, "hips", n=16, power=2.1)
k.box((0, -0.195, 0.45), (0.045, 0.015, 0.04), IRON, "hips")  # buckle
k.ellipsoid((0.15, -0.07, 0.4), (0.05, 0.04, 0.06), LEATHER, "hips")  # pouch of sling stones
k.box((-0.13, 0.09, 0.66), (0.07, 0.02, 0.1), LEATHER, "spine", rot=(0, 0, 20))  # a leather patch
k.loft("hide", [((0, 0.04, 0.78), 0.1, 0.07), ((0, 0.04, 0.75), 0.21, 0.12), ((0, 0.03, 0.7), 0.21, 0.13)], LEATHER, "spine", n=16)
for i in range(9):  # the hide's shaggy rim over the shoulders
    a = i * math.pi / 8
    p = Vector((math.cos(a) * 0.2, 0.03 + math.sin(a) * 0.12, 0.7))
    k.shard(p, Vector((math.cos(a), math.sin(a), -0.8)), 0.03, 0.05, LEATHER, "spine", flat=(0, 0, 1))

# Head: big and lumpy, wide cheeks, a heavy brow, a long hooked nose, an underbite with tusks.
head = [(0.82, 0.05, 0.05, -0.07), (0.86, 0.1, 0.09, -0.05), (0.92, 0.135, 0.125, -0.035), (0.98, 0.145, 0.135, -0.025),
        (1.03, 0.14, 0.128, -0.015), (1.07, 0.11, 0.105, -0.005), (1.1, 0.05, 0.05, 0.0)]
k.loft("neck", [((0, 0.03, 0.76), 0.06, 0.055), ((0, 0.0, 0.84), 0.06, 0.055)], SKIN, "head", n=10, power=2.0)
k.loft("head", [((0, y, z), rx, rz) for z, rx, rz, y in head], SKIN, "head", n=20, power=2.2, keel=0.85)
k.loft("jaw", [((0, -0.06, 0.88), 0.11, 0.05), ((0, -0.13, 0.86), 0.09, 0.045), ((0, -0.165, 0.865), 0.06, 0.03)], SKIN, "head",
       n=14, power=2.3, side=(1, 0, 0))  # the jutting lower jaw
k.ellipsoid((0, -0.16, 0.89), (0.06, 0.012, 0.012), MOUTH, "head", 10, 4)
k.loft("brow", [((-0.11, -0.13, 1.015), 0.02, 0.018), ((0, -0.145, 1.02), 0.025, 0.02), ((0.11, -0.13, 1.015), 0.02, 0.018)],
       SKIN_DARK, "head", n=8, side=(0, 1, 0))
for s in (-1, 1):
    k.rod((s * 0.045, -0.165, 0.88), (s * 0.055, -0.17, 0.935), 0.013, TOOTH, "head", 5, taper=0.2)  # tusks
    k.ellipsoid((s * 0.055, -0.135, 0.99), (0.035, 0.022, 0.028), EYE, "head", 8, 5)
    k.ellipsoid((s * 0.057, -0.156, 0.99), (0.013, 0.008, 0.017), PUPIL, "head", 6, 4)
    k.ellipsoid((s * 0.1, -0.11, 0.92), (0.05, 0.04, 0.04), SKIN, "head")  # cheeks
    # Ears: long flat triangles drooping out to the sides, darker inside.
    tri((s * 0.12, -0.03, 0.95), (s * 0.12, 0.03, 1.03), (s * 0.34, 0.06, 0.99), 0.012, SKIN, "head")
    tri((s * 0.135, -0.015, 0.965), (s * 0.135, 0.02, 1.015), (s * 0.29, 0.045, 0.99), 0.004, SKIN_DARK, "head")
    if s > 0:  # a notch bitten out of one ear
        k.ellipsoid((0.25, 0.04, 1.005), (0.02, 0.02, 0.02), SKIN_DARK, "head", 6, 4)
k.loft("nose", [((0, -0.14, 1.0), 0.03, 0.03), ((0, -0.2, 0.97), 0.032, 0.03), ((0, -0.235, 0.935), 0.026, 0.024),
                ((0, -0.23, 0.91), 0.016, 0.014)], SKIN_DARK, "head", n=10, side=(1, 0, 0))
for p, r in (((0.07, -0.13, 1.06), 0.014), ((-0.1, -0.09, 0.95), 0.011), ((0.02, -0.215, 0.965), 0.008), ((0.12, -0.05, 1.03), 0.012)):
    k.ellipsoid(p, (r, r, r), WART, "head", 6, 4)
for x, y in ((-0.03, -0.02), (0.02, 0.0), (0.06, 0.02), (-0.06, 0.03), (0.0, 0.05)):  # wiry tufts on top
    k.shard((x, y, 1.09), Vector((x * 4, 0.6 + y * 4, 1)), 0.012, 0.07, LEATHER_DARK, "head", flat=(0, 1, 0))

# Arms: skinny with knobbly elbows, big hands with dark claws; the right one holds the club.
for s in (-1, 1):
    shoulder, elbow, hand = Vector((s * 0.2, 0.02, 0.74)), Vector((s * 0.27, 0.05, 0.53)), Vector((s * 0.27, -0.08, 0.38))
    k.ellipsoid(shoulder, (0.055, 0.055, 0.055), SKIN, f"arm_{s}")
    k.loft(f"upper{s}", [(shoulder, 0.042, 0.042), (shoulder.lerp(elbow, 0.5), 0.034, 0.034), (elbow, 0.03, 0.03)], SKIN, f"arm_{s}",
           n=10, power=2.0)
    k.ellipsoid(elbow, (0.036, 0.036, 0.036), SKIN, f"fore_{s}", 8, 5)
    k.loft(f"fore{s}", [(elbow, 0.03, 0.03), (elbow.lerp(hand, 0.6), 0.036, 0.034), (hand, 0.03, 0.028)], SKIN, f"fore_{s}", n=10)
    k.loft(f"bracer{s}", [(elbow.lerp(hand, 0.4), 0.04, 0.038), (elbow.lerp(hand, 0.8), 0.038, 0.035)], LEATHER, f"fore_{s}", n=10)
    palm = hand + Vector((0, -0.02, -0.025))
    k.ellipsoid(palm, (0.04, 0.045, 0.04), SKIN, f"fore_{s}", 10, 6)
    for t in range(4):  # fingers curled forward and down, with claws
        x = palm.x + (t - 1.5) * 0.02
        base = Vector((x, palm.y - 0.035, palm.z))
        tip = base + Vector((0, -0.03, -0.035))
        k.rod(base, tip, 0.011, SKIN, f"fore_{s}", 5)
        k.rod(tip, tip + Vector((0, 0.005, -0.02)), 0.008, NAIL, f"fore_{s}", 4, taper=0.1)
    # Legs: short and bowed, wrapped shins, big flat three-toed feet.
    hip, knee, ankle = Vector((s * 0.09, 0, 0.38)), Vector((s * 0.13, -0.04, 0.2)), Vector((s * 0.11, 0, 0.05))
    k.loft(f"thigh{s}", [(hip, 0.055, 0.055), (hip.lerp(knee, 0.5), 0.05, 0.05), (knee, 0.042, 0.042)], SKIN, f"thigh_{s}", n=10)
    k.ellipsoid(knee, (0.046, 0.046, 0.046), SKIN, f"shin_{s}", 8, 5)
    k.loft(f"shin{s}", [(knee, 0.042, 0.042), (knee.lerp(ankle, 0.5), 0.04, 0.042), (ankle, 0.032, 0.032)], SKIN, f"shin_{s}", n=10)
    for i in range(3):  # the wraps: three bands down the shin
        c = knee.lerp(ankle, 0.35 + i * 0.22)
        k.loft(f"wrap{s}{i}", [(c + Vector((0, 0, 0.018)), 0.047, 0.048), (c - Vector((0, 0, 0.018)), 0.045, 0.046)], CLOTH, f"shin_{s}",
               n=10)
    k.loft(f"foot{s}", [((s * 0.11, 0.05, 0.035), 0.04, 0.035), ((s * 0.11, -0.03, 0.035), 0.055, 0.035),
                        ((s * 0.115, -0.11, 0.025), 0.065, 0.025)], SKIN, f"shin_{s}", n=12, power=2.4, side=(1, 0, 0))
    for t in range(3):  # toes, splayed, with claws
        a = (t - 1) * 0.35
        base = Vector((s * 0.115 + math.sin(a) * 0.03, -0.12, 0.022))
        tip = base + Vector((math.sin(a) * 0.03, -0.045, -0.005))
        k.rod(base, tip, 0.016, SKIN, f"shin_{s}", 6, taper=0.7)
        k.rod(tip, tip + Vector((math.sin(a) * 0.01, -0.022, -0.012)), 0.01, NAIL, f"shin_{s}", 4, taper=0.1)

# The club: a knobbly branch, thicker toward the top, nails sticking out.
grip = Vector((0.27, -0.1, 0.36))
top = grip + Vector((0, -0.33, 0.36))  # held forward and up, ready to swing
k.rod(grip - (top - grip) * 0.2, grip + (top - grip) * 0.4, 0.025, WOOD, "fore_1", 7)
k.rod(grip + (top - grip) * 0.4, top, 0.045, WOOD, "fore_1", 8, taper=1.3)
k.ellipsoid(top, (0.06, 0.06, 0.06), WOOD, "fore_1", 10, 6)
for i in range(10):
    t = 0.55 + 0.45 * (i % 5) / 4
    a = i * 2.4
    c = grip + (top - grip) * t
    d = Vector((math.cos(a), math.sin(a) * 0.5, 0.2)).normalized()
    k.rod(c + d * 0.04, c + d * 0.09, 0.008, IRON, "fore_1", 4, taper=0.3)
k.rod(grip - (top - grip) * 0.08, grip + (top - grip) * 0.08, 0.03, LEATHER, "fore_1", 8)  # grip wrap

bones = [("root", (0, 0, 0), None), ("hips", (0, 0, 0.42), "root"), ("spine", (0, 0, 0.5), "hips"),
         ("head", (0, 0.0, 0.78), "spine")]
for s in (-1, 1):
    bones += [(f"arm_{s}", (s * 0.2, 0.02, 0.74), "spine"), (f"fore_{s}", (s * 0.27, 0.05, 0.53), f"arm_{s}"),
              (f"thigh_{s}", (s * 0.09, 0, 0.38), "hips"), (f"shin_{s}", (s * 0.13, -0.04, 0.2), f"thigh_{s}")]
k.bones(bones)
k.bind("Goblin")
key = k.key
# Bones point up, and in a bone's own space Z is forward: rotating about X tilts what's above a
# pivot forward and swings what hangs below it backward, so an arm or leg comes forward with -X.
HUNCH = 12  # the spine leans forward this much at rest


def pose(f, spine=0.0, head=0.0, turn=0.0):
    key(f, "spine", rot=(HUNCH + spine, turn, 0))
    key(f, "head", rot=(-HUNCH + head, -turn * 0.5, 0))


def idle():
    for f, w in ((1, 0), (20, 1), (40, -1), (60, 0)):
        key(f, "hips", rot=(0, 0, 3 * w), loc=(0, -0.01 * abs(w), 0))
        pose(f, 2 * abs(w), 0, 6 * w)
    # The free hand scratches the head now and then.
    for f, (a, b) in ((1, (0, 0)), (25, (0, 0)), (32, (-140, -60)), (36, (-150, -70)), (40, (-140, -60)), (44, (-150, -70)),
                      (52, (0, 0)), (60, (0, 0))):
        key(f, "arm_-1", rot=(a, 0, -20 if a else 0))
        key(f, "fore_-1", rot=(b, 0, 0))
    for f, a in ((1, -10), (30, -14), (60, -10)):  # the club arm hangs, the club resting forward
        key(f, "arm_1", rot=(a, 0, 5))
        key(f, "fore_1", rot=(-20, 0, 0))


def walk():
    for f, ph in ((1, 1), (7, 0), (13, -1), (19, 0), (25, 1)):
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(-30 * ph * s, 0, s * 6))
            key(f, f"shin_{s}", rot=(30 * max(0, ph * s), 0, 0))
        key(f, "hips", rot=(0, 6 * ph, -4 * ph), loc=(0, -0.03 * (1 - abs(ph)), 0))
        pose(f, 6, -4, -5 * ph)
        key(f, "arm_-1", rot=(25 * ph, 0, -10))
        key(f, "fore_-1", rot=(-25, 0, 0))
        key(f, "arm_1", rot=(-10 - 10 * ph, 0, 8))
        key(f, "fore_1", rot=(-25, 0, 0))


def attack():
    # Club up behind the head, a hop forward, then down hard.
    key(8, "arm_1", rot=(-170, 0, 15))
    key(8, "fore_1", rot=(-60, 0, 0))
    pose(8, -18, 10)
    key(8, "root", loc=(0, 0, -0.03))
    key(13, "arm_1", rot=(-40, 0, 5))
    key(13, "fore_1", rot=(-10, 0, 0))
    pose(13, 25, -12)
    key(13, "root", loc=(0, 0.06, 0.3))
    for f in (8, 13):
        key(f, "thigh_1", rot=(-35 if f == 13 else 0, 0, 0))
        key(f, "shin_1", rot=(30 if f == 13 else 0, 0, 0))
        key(f, "arm_-1", rot=(-30, 0, -40))
    key(18, "arm_1", rot=(-30, 0, 5))
    key(18, "fore_1", rot=(-15, 0, 0))
    pose(18, 20, -8)
    key(18, "root", loc=(0, 0, 0.25))


def hit():
    pose(5, -20, -20, 10)
    key(5, "root", loc=(0, 0, -0.1))
    key(5, "arm_-1", rot=(-40, 0, -30))
    key(5, "arm_1", rot=(-30, 0, 30))


def death():
    pose(8, -20, -25)
    key(8, "arm_-1", rot=(-60, 0, -40))
    key(8, "arm_1", rot=(-70, 0, 40))
    key(8, "root", loc=(0, 0, -0.05))
    for f in (24, 40):  # topples backward onto its back
        key(f, "root", rot=(-85, 0, 0), loc=(0, 0.12, -0.25))
        pose(f, -12, 10)
        key(f, "arm_-1", rot=(-100, 0, -60))
        key(f, "arm_1", rot=(-110, 0, 60))
        key(f, "fore_1", rot=(-10, 0, 0))
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(-35, 0, s * 15))
            key(f, f"shin_{s}", rot=(40, 0, 0))


k.action("Idle", 61, idle)
k.action("Walk", 25, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 40, death)
k.export(args[0])
if len(args) > 1:
    k.preview(args[1], distance=2.6, target_z=0.55)
