"""Builds the scorpion: a rigged, animated desert scorpion made of simple shapes.

    blender -b --python tools/models/scorpion.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window. About 1.3 m long from the pincers to the back, the tail
curled up over the back to about 0.8 m; head toward -Y (glTF +Z). Every part is bound to one
bone: the tail is a chain of five segments and the stinger, each pincer has a moving finger.
Animations: Idle (tail sways, pincers snap), Walk, Attack (pincers grab, the tail stings over
the head), HitReact and Death (collapses, the tail falls flat behind it).
"""

import math
import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()
BODY_Z = 0.2

SHELL = k.material("Shell", (195, 125, 50))
DARK = k.material("Dark", (110, 62, 28))
STING = k.material("Sting", (55, 30, 22))
EYE = k.material("Eye", (20, 18, 16))


def segment(a, b, radius, mat, bone, flat=1.0):
    """An ellipsoid stretched from a to b (both in the YZ plane), radius wide and flat * radius high."""
    a, b = Vector(a), Vector(b)
    d = b - a
    tilt = math.degrees(math.atan2(-d.y, d.z))  # turns the ellipsoid's long (Z) axis along a→b
    k.ellipsoid((a + b) / 2, (radius, radius * flat, d.length / 2 * 1.15), mat, bone, rot=(tilt, 0, 0))


# Body: the head plate in front, then the segmented back, each segment a little narrower.
k.ellipsoid((0, -0.2, BODY_Z), (0.2, 0.22, 0.09), SHELL, "body")
for i, y in enumerate((0.02, 0.17, 0.31, 0.44)):
    k.ellipsoid((0, y, BODY_Z + 0.01), (0.21 - i * 0.025, 0.09, 0.085 - i * 0.008), SHELL if i % 2 else DARK, "body")
k.ellipsoid((0, 0.1, BODY_Z - 0.04), (0.17, 0.4, 0.05), DARK, "body")  # belly
for s in (-1, 1):
    k.ellipsoid((s * 0.04, -0.33, BODY_Z + 0.07), (0.022, 0.022, 0.022), EYE, "body", 8, 5)
    k.ellipsoid((s * 0.13, -0.38, BODY_Z + 0.04), (0.014, 0.014, 0.014), EYE, "body", 6, 4)
    k.rod((s * 0.05, -0.4, BODY_Z - 0.02), (s * 0.03, -0.46, BODY_Z - 0.05), 0.022, DARK, "body", taper=0.3)  # mouth parts

# Tail: five segments curling up and forward over the back, then the stinger hooked down.
TAIL = [(0, 0.5, 0.21), (0, 0.66, 0.3), (0, 0.76, 0.46), (0, 0.76, 0.63), (0, 0.67, 0.77), (0, 0.53, 0.83)]
for i in range(5):
    segment(TAIL[i], TAIL[i + 1], 0.075 - i * 0.008, DARK if i % 2 else SHELL, f"tail{i + 1}", flat=0.9)
tip = Vector(TAIL[5])
k.ellipsoid(tip + Vector((0, -0.05, -0.01)), (0.05, 0.07, 0.05), SHELL, "sting")
k.rod(tip + Vector((0, -0.1, -0.02)), tip + Vector((0, -0.16, -0.1)), 0.025, STING, "sting", taper=0.1)

# Pincers: an arm out to the side and forward, a big claw with a fixed outer finger and a
# moving inner one.
for s in (-1, 1):
    shoulder, elbow, wrist = (s * 0.14, -0.33, BODY_Z), (s * 0.34, -0.45, BODY_Z + 0.03), (s * 0.3, -0.62, BODY_Z + 0.02)
    k.rod(shoulder, elbow, 0.035, SHELL, f"arm_{s}")
    k.ellipsoid(elbow, (0.04, 0.04, 0.04), DARK, f"fore_{s}", 8, 5)
    k.rod(elbow, wrist, 0.04, SHELL, f"fore_{s}")
    k.ellipsoid((s * 0.29, -0.7, BODY_Z + 0.02), (0.085, 0.11, 0.06), SHELL, f"fore_{s}")
    k.rod((s * 0.32, -0.78, BODY_Z + 0.02), (s * 0.27, -0.94, BODY_Z + 0.02), 0.035, DARK, f"fore_{s}", taper=0.2)
    k.rod((s * 0.25, -0.78, BODY_Z + 0.02), (s * 0.28, -0.92, BODY_Z + 0.02), 0.028, DARK, f"finger_{s}", taper=0.2)

# Legs: four on each side, hip under the body, knee up and out, foot on the ground.
LEGS = {}
for i, (y, reach) in enumerate(((-0.24, -0.16), (-0.12, -0.04), (0.0, 0.08), (0.12, 0.2))):
    for side, name in ((-1, "L"), (1, "R")):
        bone = f"leg_{name}{i + 1}"
        hip = Vector((side * 0.16, y, BODY_Z - 0.03))
        knee = Vector((side * 0.36, y + reach * 0.5, BODY_Z + 0.08))
        foot = Vector((side * 0.5, y + reach, 0.0))
        k.rod(hip, knee, 0.024, DARK, bone)
        k.rod(knee, foot, 0.02, DARK, bone, taper=0.4)
        LEGS[bone] = side

bones = [("body", (0, 0, BODY_Z), None)]
for i in range(5):
    bones.append((f"tail{i + 1}", TAIL[i], "body" if i == 0 else f"tail{i}"))
bones.append(("sting", TAIL[5], "tail5"))
for s in (-1, 1):
    bones += [(f"arm_{s}", (s * 0.14, -0.33, BODY_Z), "body"), (f"fore_{s}", (s * 0.34, -0.45, BODY_Z + 0.03), f"arm_{s}"),
              (f"finger_{s}", (s * 0.25, -0.78, BODY_Z + 0.02), f"fore_{s}")]
bones += [(name, (side * 0.16, 0, 0), "body") for name, side in LEGS.items()]
k.bones(bones)
k.bind("Scorpion")
key = k.key
# Bones point up: rotating about X tilts what's above a pivot forward, so positive X curls the
# tail further over the back and negative X raises a pincer (it reaches forward from its pivot).


def leg(frame, name, swing=0.0, lift=0.0):
    """swing: degrees around the vertical (positive = foot forward); lift: degrees up."""
    side = LEGS[name]
    key(frame, name, rot=(0, -side * swing, side * lift))


def tail(frame, curl, sway=0.0, sting=0.0):
    """curl: degrees per segment, or a list of five (positive curls forward); sway: sideways."""
    for i, c in enumerate(curl if isinstance(curl, (list, tuple)) else [curl] * 5):
        key(frame, f"tail{i + 1}", rot=(c, 0, sway))
    key(frame, "sting", rot=(sting, 0, 0))


def claws(frame, raise_=0.0, open_=0.0, spread=0.0):
    for s in (-1, 1):
        key(frame, f"arm_{s}", rot=(-raise_, -s * spread, 0))
        key(frame, f"finger_{s}", rot=(0, s * open_, 0))


def idle():
    for f, a in ((1, 0), (20, 4), (40, -4), (60, 0)):
        tail(f, abs(a) * 0.5, a)
    for f, o in ((1, 0), (10, 25), (16, 0), (36, 25), (42, 0), (60, 0)):
        claws(f, 0, o)
    for f, z in ((1, 0), (30, -0.008), (60, 0)):
        key(f, "body", loc=(0, z, 0))


def walk():
    groups = (("leg_L1", "leg_R2", "leg_L3", "leg_R4"), ("leg_R1", "leg_L2", "leg_R3", "leg_L4"))
    for g, group in enumerate(groups):
        for name in group:
            for f, s, lift in ((1, 15, 0), (6, 0, 0), (11, -15, 0), (16, 0, 20), (21, 15, 0)):
                ff = (f - 1 + g * 10) % 20 + 1
                leg(ff, name, s, lift)
                if ff == 1:
                    leg(21, name, s, lift)
    for f, z, a in ((1, 0, 0), (6, 0.01, 3), (11, 0, 0), (16, 0.01, -3), (21, 0, 0)):
        key(f, "body", loc=(0, z, 0))
        tail(f, 0, a)


def attack():
    claws(6, 20, 35, -10)  # pincers up and open
    tail(6, -8, 0, -15)  # the tail draws back
    key(6, "body", loc=(0, 0.02, -0.06))
    claws(11, -5, 0, 5)  # grab
    tail(11, (40, 20, -5, -12, -10), 0, 25)  # and sting over the head
    key(11, "body", rot=(6, 0, 0), loc=(0, 0, 0.18))
    tail(17, 10, 0, 10)


def hit():
    key(5, "body", rot=(-8, 0, 5), loc=(0, 0.02, -0.12))
    claws(5, 25, 20, 10)
    tail(5, -6, 8)


def death():
    claws(10, 15, 30, 15)
    tail(10, -6, 6, 10)
    key(10, "body", rot=(0, 0, 8), loc=(0, 0.02, 0))
    for name in LEGS:
        leg(12, name, 0, 15)
        leg(30, name, 0, 25)
    claws(28, -5, 30, 25)
    tail(26, (-36, -28, -30, -31, -28), 4, -20)  # the tail falls flat behind it
    tail(36, (-38, -30, -32, -33, -30), 4, -20)
    key(26, "body", rot=(0, 0, 4), loc=(0, -0.12, 0))
    key(36, "body", rot=(0, 0, 4), loc=(0, -0.13, 0))


k.action("Idle", 61, idle)
k.action("Walk", 21, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 36, death)
k.export(args[0])
if len(args) > 1:
    k.preview(args[1], distance=3.0, target_z=0.35)
