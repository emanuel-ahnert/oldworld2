"""Builds the giant crab of the Mediterranean shores: a rigged, animated crab made of simple
shapes.

    blender -b --python tools/models/crab.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window. About 1.2 m across the legs and 0.6 m tall at the eye
stalks (the game scales it up); its front toward -Y (glTF +Z). A wide, knobbly shell in
red-orange with a pale underside, eyes on stalks, two big claws (the right one bigger), each
with a moving finger, and four jointed legs on each side. Every part is bound to one bone.
Animations: Idle (claws snap, eyes twitch), Walk (scuttles sideways), Attack (lunges with
both claws and pinches), HitReact and Death (sinks and flips its claws up).
"""

import math
import sys
from pathlib import Path

from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()
BODY_Z = 0.22

SHELL = k.material("Shell", (206, 84, 48))
SHELL_DARK = k.material("ShellDark", (150, 54, 34))
BELLY = k.material("Belly", (236, 204, 168))
TIP = k.material("Tip", (60, 36, 30))
EYE = k.material("Eye", (24, 20, 18))
SPOT = k.material("Spot", (232, 144, 90))

# The shell: wide and flat, a ridge across the front edge, knobs over the top, a pale belly.
k.ellipsoid((0, 0, BODY_Z), (0.34, 0.25, 0.11), SHELL, "body", 20, 10)
k.ellipsoid((0, 0.01, BODY_Z - 0.05), (0.3, 0.22, 0.07), BELLY, "body", 18, 8)
k.loft("rim", [((-0.3, -0.12, BODY_Z + 0.02), 0.03, 0.025), ((0, -0.22, BODY_Z + 0.04), 0.035, 0.03),
               ((0.3, -0.12, BODY_Z + 0.02), 0.03, 0.025)], SHELL_DARK, "body", n=8, side=(0, 1, 0))
for x, y, r in ((-0.15, -0.05, 0.05), (0.15, -0.05, 0.05), (0, 0.06, 0.06), (-0.22, 0.06, 0.04), (0.22, 0.06, 0.04), (0, -0.12, 0.035)):
    k.ellipsoid((x, y, BODY_Z + 0.085), (r, r, r * 0.6), SPOT if r < 0.05 else SHELL_DARK, "body", 10, 6)
for s in (-1, 1):  # eyes on stalks, and little mouth parts under the front edge
    k.rod((s * 0.07, -0.19, BODY_Z + 0.06), (s * 0.08, -0.22, BODY_Z + 0.18), 0.016, SHELL, f"eye_{s}", 6)
    k.ellipsoid((s * 0.08, -0.22, BODY_Z + 0.2), (0.03, 0.03, 0.03), EYE, f"eye_{s}", 8, 5)
    k.rod((s * 0.04, -0.22, BODY_Z - 0.01), (s * 0.03, -0.27, BODY_Z - 0.06), 0.018, SHELL_DARK, "body", 5, taper=0.4)

# Claws: an arm forward and out, a big claw with a fixed outer finger and a moving inner one.
for s in (-1, 1):
    big = 1.25 if s > 0 else 1.0
    shoulder, elbow, wrist = (s * 0.25, -0.16, BODY_Z), (s * 0.38, -0.32, BODY_Z + 0.05), (s * 0.3, -0.45, BODY_Z + 0.06)
    k.rod(shoulder, elbow, 0.04, SHELL, f"arm_{s}", 8)
    k.ellipsoid(elbow, (0.045, 0.045, 0.045), SHELL_DARK, f"fore_{s}", 8, 5)
    k.rod(elbow, wrist, 0.045, SHELL, f"fore_{s}", 8)
    palm = Vector((s * 0.28, -0.53, BODY_Z + 0.06))
    k.ellipsoid(palm, (0.075 * big, 0.11 * big, 0.065 * big), SHELL, f"fore_{s}", 12, 7)
    k.rod(palm + Vector((s * 0.03, -0.08 * big, 0.01)), palm + Vector((s * 0.0, -0.2 * big, 0.02)), 0.035 * big, SHELL_DARK, f"fore_{s}", 7,
          taper=0.2)
    k.rod(palm + Vector((-s * 0.04, -0.08 * big, 0.0)), palm + Vector((-s * 0.01, -0.19 * big, -0.01)), 0.03 * big, TIP,
          f"finger_{s}", 7, taper=0.2)

# Legs: four a side, from under the shell up to a knee and down to a pointed foot.
LEGS = {}
for i, (y, reach) in enumerate(((-0.12, -0.12), (-0.03, -0.04), (0.06, 0.05), (0.15, 0.14))):
    for side, name in ((-1, "L"), (1, "R")):
        bone = f"leg_{name}{i + 1}"
        hip = Vector((side * 0.27, y, BODY_Z - 0.02))
        knee = Vector((side * 0.46, y + reach * 0.5, BODY_Z + 0.1))
        foot = Vector((side * 0.6, y + reach, 0.0))
        k.rod(hip, knee, 0.026, SHELL, bone, 6)
        k.ellipsoid(knee, (0.028, 0.028, 0.028), SHELL_DARK, bone, 6, 4)
        k.rod(knee, foot, 0.022, SHELL, bone, 6, taper=0.3)
        k.rod(foot + Vector((0, 0, 0.03)), foot, 0.01, TIP, bone, 4, taper=0.2)
        LEGS[bone] = side

bones = [("body", (0, 0, BODY_Z), None)]
for s in (-1, 1):
    bones += [(f"arm_{s}", (s * 0.25, -0.16, BODY_Z), "body"), (f"fore_{s}", (s * 0.38, -0.32, BODY_Z + 0.05), f"arm_{s}"),
              (f"finger_{s}", (s * 0.27, -0.6, BODY_Z + 0.06), f"fore_{s}"), (f"eye_{s}", (s * 0.07, -0.19, BODY_Z + 0.06), "body")]
bones += [(name, (side * 0.27, 0, BODY_Z - 0.02), "body") for name, side in LEGS.items()]
k.bones(bones)
k.bind("GiantCrab")
key = k.key


def leg(frame, name, swing=0.0, lift=0.0):
    """swing: degrees around the vertical (positive = foot forward); lift: degrees up."""
    side = LEGS[name]
    key(frame, name, rot=(0, -side * swing, side * lift))


def claws(frame, raise_=0.0, open_=0.0, spread=0.0):
    for s in (-1, 1):
        key(frame, f"arm_{s}", rot=(-raise_, -s * spread, 0))
        key(frame, f"finger_{s}", rot=(0, s * open_, 0))


def idle():
    for f, o in ((1, 0), (12, 30), (18, 0), (40, 0), (46, 30), (52, 0), (60, 0)):
        claws(f, 5 if o else 0, o)
    for f, a in ((1, 0), (20, 12), (24, 0), (50, -12), (54, 0), (60, 0)):
        for s in (-1, 1):
            key(f, f"eye_{s}", rot=(0, 0, a * s))
    for f, z in ((1, 0), (30, -0.01), (60, 0)):
        key(f, "body", loc=(0, z, 0))


def walk():
    # Scuttling sideways: the legs on each side lift and reach in turn, the body sways.
    groups = (("leg_L1", "leg_R2", "leg_L3", "leg_R4"), ("leg_R1", "leg_L2", "leg_R3", "leg_L4"))
    for g, group in enumerate(groups):
        for name in group:
            for f, s, lift in ((1, 0, 0), (5, 0, 25), (9, 0, 0), (17, 0, -8), (17, 0, 0)):
                ff = (f - 1 + g * 8) % 16 + 1
                leg(ff, name, s, lift)
                if ff == 1:
                    leg(17, name, s, lift)
    for f, z, r in ((1, 0, 0), (5, 0.015, 4), (9, 0, 0), (13, 0.015, -4), (17, 0, 0)):
        key(f, "body", rot=(0, 0, r), loc=(0, z, 0))
        claws(f, 8, 0, 0)


def attack():
    claws(7, 35, 40, -15)  # both claws up and wide open
    key(7, "body", loc=(0, 0.03, 0.05))
    claws(12, -10, 0, 10)  # snap shut on the target
    key(12, "body", rot=(8, 0, 0), loc=(0, 0, 0.22))
    claws(18, 5, 15, 0)


def hit():
    key(5, "body", rot=(-10, 0, 6), loc=(0, 0.02, -0.1))
    claws(5, 30, 30, 15)
    for s in (-1, 1):
        key(5, f"eye_{s}", rot=(-30, 0, 0))


def death():
    claws(10, 30, 40, 20)
    key(10, "body", rot=(0, 0, 6), loc=(0, 0.02, 0))
    for name in LEGS:  # the legs give way and curl in under it
        leg(12, name, 0, 10)
        leg(30, name, 0, 35)
    for s in (-1, 1):
        key(26, f"eye_{s}", rot=(-70, 0, 0))
    key(26, "body", rot=(0, 0, 3), loc=(0, -0.14, 0))
    key(36, "body", rot=(0, 0, 3), loc=(0, -0.15, 0))
    claws(28, 45, 50, 40)  # the claws fall open
    claws(36, 50, 50, 45)


k.action("Idle", 61, idle)
k.action("Walk", 17, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 36, death)
k.export(args[0])
if len(args) > 1 and args[1]:
    k.preview(args[1], distance=3.0, target_z=0.35)
