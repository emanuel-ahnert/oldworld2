"""Builds the tomb guardian: a golden jackal-headed temple statue with a sun staff, rigged and
animated.

    blender -b --python tools/models/guardian.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; faces -Y (glTF +Z), about 2.3 m tall. Gold with lapis
stripes on the headdress, collar and kilt; the staff in its right hand carries a sun disc.
Every part is bound to one bone. Animations: Idle, Walk (a heavy stride), Attack (swings the
staff overhead and brings it down), HitReact and Death (falls to its knees and slumps forward).
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
k = Kit()

GOLD = k.material("Gold", (225, 175, 65))
DEEP = k.material("DeepGold", (170, 120, 40))
LAPIS = k.material("Lapis", (35, 65, 150))
BLACK = k.material("Black", (30, 26, 24))
GLOW = k.material("Glow", (255, 245, 170))

# Torso: chest, waist, the kilt flaring below, a broad collar over the shoulders.
k.ellipsoid((0, 0, 1.5), (0.3, 0.18, 0.24), GOLD, "spine")
k.ellipsoid((0, 0, 1.22), (0.2, 0.14, 0.14), GOLD, "spine")
k.rod((0, 0, 1.12), (0, 0, 0.82), 0.2, GOLD, "hips", 16, taper=1.35)  # kilt
k.box((0, -0.25, 0.92), (0.14, 0.03, 0.26), LAPIS, "hips")  # the kilt's front panel
for z in (0.86, 0.95, 1.04):
    k.box((0, -0.27, z), (0.15, 0.01, 0.025), GOLD, "hips")
k.rod((0, 0, 1.15), (0, 0, 1.1), 0.21, LAPIS, "hips", 16)  # belt
k.rod((0, 0, 1.62), (0, 0, 1.69), 0.34, LAPIS, "spine", 20, taper=0.65)  # collar, in rings
k.rod((0, 0, 1.69), (0, 0, 1.73), 0.23, GOLD, "spine", 20, taper=0.75)

# Head: the jackal's skull and muzzle, tall ears, the striped headdress falling behind the
# ears to the shoulders, glowing eyes.
k.rod((0, 0, 1.7), (0, -0.02, 1.86), 0.08, GOLD, "head", 10)  # neck
k.ellipsoid((0, 0.0, 1.95), (0.1, 0.12, 0.1), BLACK, "head")
k.ellipsoid((0, -0.17, 1.91), (0.045, 0.13, 0.045), BLACK, "head")  # muzzle
k.ellipsoid((0, -0.29, 1.92), (0.022, 0.022, 0.02), BLACK, "head", 8, 5)  # nose tip
k.ellipsoid((0, 0.06, 1.94), (0.15, 0.11, 0.13), LAPIS, "head")  # headdress, behind the ears
for s in (-1, 1):
    k.ellipsoid((s * 0.075, 0.0, 2.13), (0.045, 0.02, 0.12), BLACK, "head", 12, 8, rot=(0, s * 12, 0))  # ears
    k.ellipsoid((s * 0.073, -0.012, 2.11), (0.025, 0.01, 0.08), GOLD, "head", 10, 6, rot=(0, s * 12, 0))
    k.ellipsoid((s * 0.05, -0.1, 1.98), (0.024, 0.016, 0.013), GLOW, "head", 8, 5)
    for i in range(5):  # the headdress falls to the chest in gold and lapis stripes
        k.box((s * 0.14, -0.02, 1.92 - i * 0.055), (0.08, 0.06, 0.056), LAPIS if i % 2 else GOLD, "head")

# Arms: bracers of lapis; the right hand holds the staff upright, the left is a fist.
for s in (-1, 1):
    shoulder, elbow, hand = (s * 0.37, 0, 1.6), (s * 0.43, 0.03, 1.28), (s * 0.43, -0.25, 1.22)
    k.ellipsoid(shoulder, (0.11, 0.11, 0.1), GOLD, f"arm_{s}")
    k.rod(shoulder, elbow, 0.075, GOLD, f"arm_{s}", 10)
    k.ellipsoid(elbow, (0.065, 0.065, 0.065), GOLD, f"fore_{s}")
    k.rod(elbow, hand, 0.07, LAPIS, f"fore_{s}", 10, taper=0.85)
    k.ellipsoid((s * 0.43, -0.3, 1.22), (0.065, 0.07, 0.07), GOLD, f"fore_{s}")
    # Legs: shins with lapis greaves, sandals.
    hip, knee, ankle = (s * 0.12, 0, 1.0), (s * 0.13, -0.01, 0.56), (s * 0.13, 0.01, 0.1)
    k.rod(hip, knee, 0.095, GOLD, f"thigh_{s}", 10, taper=0.85)
    k.ellipsoid(knee, (0.08, 0.08, 0.08), GOLD, f"shin_{s}")
    k.rod(knee, ankle, 0.075, GOLD, f"shin_{s}", 10, taper=0.8)
    k.rod((s * 0.13, -0.005, 0.42), (s * 0.13, 0.005, 0.18), 0.082, LAPIS, f"shin_{s}", 10, taper=0.85)
    k.ellipsoid((s * 0.13, -0.06, 0.05), (0.075, 0.15, 0.05), DEEP, f"shin_{s}")
k.rod((0.43, -0.3, 0.05), (0.43, -0.3, 2.08), 0.035, DEEP, "fore_1", 8)  # staff
k.rod((0.43, -0.3, 2.08), (0.43, -0.3, 2.12), 0.06, GOLD, "fore_1", 8)
k.ellipsoid((0.43, -0.3, 2.24), (0.13, 0.03, 0.13), GOLD, "fore_1", 20, 8)  # sun disc
k.ellipsoid((0.43, -0.33, 2.24), (0.08, 0.015, 0.08), GLOW, "fore_1", 16, 6)

bones = [("root", (0, 0, 0), None), ("hips", (0, 0, 1.05), "root"), ("spine", (0, 0, 1.15), "hips"),
         ("head", (0, 0, 1.72), "spine")]
for s in (-1, 1):
    bones += [(f"arm_{s}", (s * 0.37, 0, 1.6), "spine"), (f"fore_{s}", (s * 0.43, 0.03, 1.28), f"arm_{s}"),
              (f"thigh_{s}", (s * 0.12, 0, 1.0), "hips"), (f"shin_{s}", (s * 0.13, -0.01, 0.56), f"thigh_{s}")]
k.bones(bones)
k.bind("TombGuardian")
key = k.key
# Bones point up: rotating about X tilts what's above a pivot forward and swings what hangs
# below it backward, so an arm or leg comes forward with negative X.


def idle():
    for f, a in ((1, 0), (30, 3), (60, 0)):
        key(f, "spine", rot=(0, a, 0))
        key(f, "head", rot=(0, -a * 3, 0))
        key(f, "arm_-1", rot=(0, 0, -a))


def walk():
    for f, ph in ((1, 1), (9, 0), (17, -1), (25, 0), (33, 1)):
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(-20 * ph * s, 0, 0))
            key(f, f"shin_{s}", rot=(18 * max(0, ph * s), 0, 0))
        key(f, "hips", rot=(0, 3 * ph, 0), loc=(0, -0.025 * (1 - abs(ph)), 0))
        key(f, "arm_-1", rot=(-15 * ph, 0, 0))  # the free arm swings
        key(f, "arm_1", rot=(6 * ph, 0, 0))  # the staff arm barely
        key(f, "spine", rot=(3, -3 * ph, 0))


def attack():
    # The staff stays upright while the arm raises it high (forearm turns back as far as the
    # arm turns up), then its sun disc swings down in front.
    key(9, "arm_1", rot=(-150, 0, 10))
    key(9, "fore_1", rot=(150, 0, 0))
    key(9, "spine", rot=(-10, 10, 0))
    key(9, "head", rot=(-8, 0, 0))
    key(15, "arm_1", rot=(-70, 0, 5))
    key(15, "fore_1", rot=(130, 0, 0))
    key(15, "spine", rot=(18, -8, 0))
    key(15, "root", loc=(0, 0, 0.3))
    for f in (9, 15):
        key(f, "thigh_1", rot=(-25 if f == 15 else 0, 0, 0))
        key(f, "shin_1", rot=(20 if f == 15 else 0, 0, 0))
    key(20, "arm_1", rot=(-60, 0, 5))
    key(20, "fore_1", rot=(110, 0, 0))
    key(20, "spine", rot=(14, -6, 0))
    key(20, "root", loc=(0, 0, 0.25))


def hit():
    key(5, "spine", rot=(-12, 0, -5))
    key(5, "head", rot=(-15, 0, 0))
    key(5, "root", loc=(0, 0, -0.12))
    key(5, "arm_-1", rot=(-20, 0, -15))


def death():
    key(8, "spine", rot=(-10, 0, 0))
    key(8, "head", rot=(-15, 0, 0))
    for f in (16, 40):  # down onto the knees
        key(f, "root", loc=(0, -0.44, 0))
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(-8, 0, s * 4))
            key(f, f"shin_{s}", rot=(92, 0, 0))
    key(16, "spine", rot=(8, 0, 0))
    key(16, "head", rot=(20, 0, 0))
    key(16, "arm_1", rot=(-20, 0, 25))
    key(32, "spine", rot=(100, 0, 0))  # and slumps forward
    key(40, "spine", rot=(104, 0, 0))
    for f in (32, 40):
        key(f, "head", rot=(-30, 0, 15))
        key(f, "arm_1", rot=(-150, 0, 40))
        key(f, "fore_1", rot=(-45, 0, 0))  # the staff falls flat
        key(f, "arm_-1", rot=(-150, 0, -25))


k.action("Idle", 61, idle)
k.action("Walk", 33, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 40, death)
k.export(args[0])
if len(args) > 1:
    k.preview(args[1], distance=5.5, target_z=1.1)
