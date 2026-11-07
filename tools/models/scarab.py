"""Builds the scarab: a rigged, animated beetle made of simple shapes (no pack has one).

    blender -b --python tools/models/scarab.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window. The beetle is about 1.2 m long and 0.6 m high, head
toward -Y (glTF +Z, the direction game models face). Every part is bound to one bone, so the
animations are rigid: Idle, Walk (tripod gait), Attack (lunge and head slam), HitReact and
Death (flips onto its back). With PREVIEW_DIR, also renders a few frames of each to PNG.
"""

import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()
BODY_Z = 0.3  # center of the body above the ground

SHELL = k.material("Shell", (40, 95, 105))
DARK = k.material("Dark", (30, 32, 38))
GOLD = k.material("Gold", (215, 165, 60))
EYE = k.material("Eye", (240, 190, 70))

# Body: two wing cases meeting in a seam, the thorax in front, then the head.
for side in (-1, 1):
    k.ellipsoid((side * 0.12, 0.12, BODY_Z), (0.25, 0.56, 0.27), SHELL, "body")
k.ellipsoid((0, -0.42, BODY_Z - 0.01), (0.31, 0.2, 0.19), SHELL, "body")
k.ellipsoid((0, 0.08, BODY_Z - 0.07), (0.33, 0.56, 0.21), DARK, "body")  # belly, shows when it lies on its back
k.ellipsoid((0, -0.64, BODY_Z - 0.05), (0.19, 0.15, 0.12), DARK, "head")
k.rod((0, -0.72, BODY_Z), (0, -0.86, BODY_Z + 0.2), 0.05, GOLD, "head", taper=0.15)  # horn
for side in (-1, 1):
    k.ellipsoid((side * 0.13, -0.72, BODY_Z - 0.01), (0.04, 0.04, 0.04), EYE, "head", 10, 6)
    k.rod((side * 0.08, -0.76, BODY_Z - 0.12), (side * 0.03, -0.9, BODY_Z - 0.14), 0.03, DARK, "head", taper=0.2)
    k.rod((side * 0.12, -0.74, BODY_Z + 0.03), (side * 0.25, -0.9, BODY_Z + 0.12), 0.012, DARK, "head")
    k.ellipsoid((side * 0.26, -0.91, BODY_Z + 0.13), (0.03, 0.04, 0.025), DARK, "head", 8, 5)

# Legs: hip on the body side, knee up and out, foot on the ground. Front legs reach forward,
# back legs backward.
LEGS = {}
for i, (y, reach) in enumerate(((-0.42, -0.25), (-0.18, 0.0), (0.12, 0.3))):
    for side, name in ((-1, "L"), (1, "R")):
        bone = f"leg_{name}{i + 1}"
        hip = Vector((side * 0.24, y, BODY_Z - 0.1))
        knee = Vector((side * 0.5, y + reach * 0.5, BODY_Z + 0.06))
        foot = Vector((side * 0.68, y + reach, 0.0))
        k.rod(hip, knee, 0.04, DARK, bone)
        k.rod(knee, foot, 0.032, DARK, bone, taper=0.4)
        k.ellipsoid(knee, (0.045, 0.045, 0.045), GOLD, bone, 8, 5)
        LEGS[bone] = (hip, side)

k.bones([("body", (0, 0, BODY_Z), None), ("head", (0, -0.55, BODY_Z), "body")] +
        [(name, hip, "body") for name, (hip, side) in LEGS.items()])
k.bind("Scarab")
key = k.key


def leg(frame, name, swing=0.0, lift=0.0):
    """swing: degrees around the vertical (positive = foot forward); lift: degrees up."""
    side = LEGS[name][1]
    key(frame, name, rot=(0, -side * swing, side * lift))


def idle():
    for f, z in ((1, 0), (30, -0.012), (60, 0)):
        key(f, "body", loc=(0, z, 0))
    for f, a in ((1, 0), (20, 6), (40, -6), (60, 0)):
        key(f, "head", rot=(0, a, 0))


def walk():
    tripods = (("leg_L1", "leg_R2", "leg_L3"), ("leg_R1", "leg_L2", "leg_R3"))
    for t, group in enumerate(tripods):
        for name in group:
            for f, s, lift in ((1, 18, 0), (6, 0, 0), (11, -18, 0), (16, 0, 22), (21, 18, 0)):
                ff = (f - 1 + t * 10) % 20 + 1
                leg(ff, name, s, lift)
                if ff == 1:
                    leg(21, name, s, lift)
    for f, z in ((1, 0), (6, 0.015), (11, 0), (16, 0.015), (21, 0)):
        key(f, "body", loc=(0, z, 0))


def attack():
    key(8, "body", rot=(-8, 0, 0), loc=(0, 0.03, -0.1))  # rear back
    key(14, "body", rot=(6, 0, 0), loc=(0, 0, 0.3))  # lunge forward
    key(8, "head", rot=(-25, 0, 0))
    key(14, "head", rot=(30, 0, 0))


def hit():
    key(5, "body", rot=(-10, 0, 6), loc=(0, 0.02, -0.15))
    key(5, "head", rot=(-20, 0, 0))


def death():
    key(10, "body", rot=(0, 0, 40), loc=(0, 0.15, 0))
    key(24, "body", rot=(0, 0, 180), loc=(0, 0.05, 0))
    key(36, "body", rot=(0, 0, 180), loc=(0, 0.0, 0))
    # On its back, negative lift (toward the belly) points the legs up in the air.
    for name in LEGS:
        leg(24, name, 0, -20)
        leg(30, name, 12, -40)
        leg(36, name, -6, -32)


k.action("Idle", 61, idle)
k.action("Walk", 21, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 36, death)
k.export(args[0])
if len(args) > 1:
    k.preview(args[1], distance=3.0, target_z=0.3)
