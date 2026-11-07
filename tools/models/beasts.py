"""Builds the four-legged animals: the wolf and the boar, rigged and animated.

    blender -b --python tools/models/beasts.py -- wolf|snowwolf|direwolf|boar OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; the animal faces -Y (glTF +Z). The wolf is about 1 m
tall at the ears, grey with a pale belly and a shaggy ruff (the snow wolf of the North is the same
wolf, white with blue-grey markings and ice-blue eyes, and the unholy lands' dire wolf is black
with red eyes); the boar about 0.85 m, brown with
a dark bristly mane and curved tusks. Both share one rig: root, body, neck, head, jaw, a
three-bone tail and two bones per leg (upper, and lower with the paw or hoof). Every part is
bound to one bone. Animations: Idle, Walk (a trot), Run (a gallop), Attack (the wolf lunges
and bites, the boar charges and tosses its tusks up), HitReact and Death (collapses onto its
belly, then rolls onto its side).
"""

import math
import sys
from pathlib import Path

import bpy
from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
kind = args[0]
k = Kit()
mesh_part, loft, shard = k.mesh, k.loft, k.shard  # shape helpers from rigkit
rnd_state = [7]


def rand(a, b):
    """A small deterministic random generator, so the fur comes out the same every build."""
    rnd_state[0] = (rnd_state[0] * 1103515245 + 12345) % 2 ** 31
    return a + (b - a) * rnd_state[0] / 2 ** 31


def fur_line(points, mat, bone, count, width, length, lean=(0, 1, 0.4), jitter=0.4):
    """Clumps of fur along a line (a back, a ruff's rim), all leaning roughly one way."""
    pts = [Vector(p) for p in points]
    for i in range(count):
        t = i / max(count - 1, 1) * (len(pts) - 1)
        a, f = int(min(t, len(pts) - 2)), t - int(min(t, len(pts) - 2))
        p = pts[a].lerp(pts[a + 1], f)
        d = Vector(lean) + Vector((rand(-1, 1), rand(-1, 1), rand(-1, 1))) * jitter
        shard(p, d, width * rand(0.8, 1.2), length * rand(0.75, 1.25), mat, bone, flat=(0, 0, 1) if abs(d.normalized().z) < 0.8 else (0, 1, 0))


def fur_ring(center, rx, rz, mat, bone, count, width, length, lean, arc=(0, 2 * math.pi)):
    """Clumps around an ellipse in the XZ plane at center (a ruff, a cheek), leaning out and back."""
    c = Vector(center)
    for i in range(count):
        a = arc[0] + (arc[1] - arc[0]) * (i + rand(-0.3, 0.3)) / count
        p = c + Vector((math.cos(a) * rx, 0, math.sin(a) * rz))
        out = Vector((math.cos(a), 0, math.sin(a)))
        d = out * 0.8 + Vector(lean) + Vector((rand(-1, 1), rand(-1, 1), rand(-1, 1))) * 0.25
        shard(p, d, width * rand(0.8, 1.2), length * rand(0.75, 1.25), mat, bone, flat=(0, 1, 0))


LEGS = []  # (upper bone, lower bone, front?, hip, knee)


def leg(name, profile, split, mat, foot_mat, front, toes, toe_size, hoof=False, feather=None):
    """A leg lofted through profile [(point, half width, half depth)...]; the parts above
    profile[split] move with the upper bone, the rest (and the foot) with the lower one."""
    up, low = f"{name}_up", f"{name}_low"
    loft(f"{name}_upper", profile[:split + 1], mat, up, n=10, power=2.2, side=(1, 0, 0))
    loft(f"{name}_lower", profile[split:], mat, low, n=10, power=2.2, side=(1, 0, 0))
    k.ellipsoid(profile[split][0], (profile[split][1] * 1.05,) * 3, mat, low, 10, 6)  # the joint
    foot = Vector(profile[-1][0])
    if hoof:
        for s in (-1, 1):  # cloven: two toes
            k.ellipsoid(foot + Vector((s * toe_size * 0.55, -toe_size * 0.5, -foot.z + toe_size * 0.7)), (toe_size * 0.55, toe_size, toe_size * 0.8),
                        foot_mat, low, 8, 5)
    else:
        k.ellipsoid(foot + Vector((0, -toe_size * 0.4, -foot.z + toe_size * 0.9)), (toe_size * 1.6, toe_size * 1.9, toe_size), mat, low, 10, 6)
        for t in range(toes):
            x = (t - (toes - 1) / 2) * toe_size * 0.9
            k.ellipsoid(foot + Vector((x, -toe_size * 1.8, -foot.z + toe_size * 0.6)), (toe_size * 0.5, toe_size * 0.7, toe_size * 0.55), mat, low, 8, 5)
            k.rod(foot + Vector((x, -toe_size * 2.4, -foot.z + toe_size * 0.5)), foot + Vector((x, -toe_size * 2.8, -foot.z + 0.004)),
                  toe_size * 0.18, foot_mat, low, 4, taper=0.2)  # claws
    if feather:  # fur feathering down the back of the leg
        a, b, m = feather
        fur_line([a, b], m, up, 5, 0.025, 0.07, lean=(0, 1, -0.6), jitter=0.3)
    i = len(LEGS)
    LEGS.append((up, low, front, Vector(profile[0][0]), Vector(profile[split][0])))


def tail_parts(points, radii, mat, fur=None, fur_len=0.0, fur_width=0.03):
    for i in range(len(points) - 1):
        a, b = Vector(points[i]), Vector(points[i + 1])
        loft(f"tail{i}", [(a, radii[i], radii[i]), ((a + b) / 2, radii[i] * 1.1, radii[i] * 1.1), (b, radii[i + 1], radii[i + 1])],
             mat, f"tail{min(i, 2) + 1}", n=10, power=2.0)  # the tip segment rides on the last bone
        if fur:
            for j in range(9):
                p = a.lerp(b, rand(0, 1))
                ang = rand(0, 2 * math.pi)
                out = Vector((math.cos(ang), 0, math.sin(ang)))
                shard(p + out * radii[i] * 0.7, out + (b - a).normalized() * 1.3, fur_width, fur_len, fur, f"tail{min(i, 2) + 1}", flat=(0, 1, 0))


def wolf():
    fur, dark, pale, eye = {  # the grey wolf, the North's white snow wolf, the unholy lands' black dire wolf
        "wolf": ((118, 118, 126), (70, 70, 80), (205, 200, 190), (230, 170, 40)),
        "snowwolf": ((222, 226, 232), (138, 150, 168), (246, 247, 250), (120, 200, 240)),
        "direwolf": ((52, 48, 56), (30, 28, 34), (96, 90, 100), (230, 40, 40)),
    }[kind]
    fur, dark, pale = k.material("Fur", fur), k.material("DarkFur", dark), k.material("PaleFur", pale)
    nose = k.material("Nose", (25, 22, 22))
    eye = k.material("Eye", eye)
    pupil = k.material("Pupil", (20, 15, 10))
    teeth = k.material("Teeth", (240, 235, 220))
    mouth = k.material("Mouth", (110, 35, 40))
    claw = k.material("Claw", (45, 40, 38))
    # Body: one surface from the chest to the rump. A deep, keeled chest, the waist tucked up,
    # the hips a little higher than the back.
    body = [((0, -0.5, 0.72), 0.08, 0.12), ((0, -0.42, 0.67), 0.15, 0.22), ((0, -0.28, 0.63), 0.17, 0.25),
            ((0, -0.12, 0.645), 0.155, 0.21), ((0, 0.04, 0.67), 0.125, 0.15), ((0, 0.2, 0.675), 0.125, 0.15),
            ((0, 0.36, 0.67), 0.135, 0.16), ((0, 0.48, 0.65), 0.1, 0.12), ((0, 0.53, 0.64), 0.04, 0.05)]
    loft("body", body, fur, "body", n=16, power=2.5, keel=0.55)
    # A pale underside: chest and belly.
    loft("chestfur", [((0, -0.46, 0.6), 0.06, 0.08), ((0, -0.36, 0.5), 0.11, 0.12), ((0, -0.2, 0.44), 0.1, 0.07),
                      ((0, -0.02, 0.53), 0.07, 0.04), ((0, 0.12, 0.56), 0.05, 0.03)], pale, "body", n=12, power=2.2)
    # Dark saddle over the back, fur clumps along the spine and down the sides of the chest.
    # Darker fur along the back, in clumps lying back along the spine.
    for s in (-0.05, 0, 0.05):
        fur_line([(s, -0.36, 0.86), (s * 1.6, -0.1, 0.85), (s * 1.4, 0.15, 0.81), (s, 0.42, 0.79)], dark, "body", 14, 0.035, 0.1,
                 lean=(s * 4, 1.4, 0.35), jitter=0.25)
    for s in (-1, 1):
        fur_line([(s * 0.15, -0.38, 0.52), (s * 0.15, -0.18, 0.46), (s * 0.1, 0.0, 0.53)], pale, "body", 8, 0.03, 0.08,
                 lean=(s * 0.4, 0.5, -1))  # shaggy under the chest
        fur_line([(s * 0.15, -0.3, 0.75), (s * 0.16, -0.1, 0.72), (s * 0.12, 0.25, 0.74)], fur, "body", 9, 0.03, 0.07,
                 lean=(s * 0.6, 1, 0.1))  # along the flanks
    # Neck and ruff: a thick collar of long fur.
    loft("neck", [((0, -0.4, 0.7), 0.12, 0.15), ((0, -0.5, 0.79), 0.12, 0.14), ((0, -0.6, 0.87), 0.09, 0.1)], fur, "neck", n=14, power=2.2)
    fur_ring((0, -0.47, 0.8), 0.14, 0.15, fur, "neck", 14, 0.045, 0.1, lean=(0, 2.2, 0.1), arc=(-0.6, math.pi + 0.6))
    fur_ring((0, -0.5, 0.74), 0.12, 0.13, fur, "neck", 12, 0.04, 0.09, lean=(0, 1.8, -0.2), arc=(-0.3, math.pi + 0.3))
    fur_ring((0, -0.53, 0.68), 0.09, 0.08, pale, "neck", 7, 0.04, 0.09, lean=(0, 0.8, -1.2), arc=(math.pi * 1.15, math.pi * 1.85))
    # Head: skull, a stop where the forehead drops to the muzzle, a long tapering muzzle.
    head = [((0, -0.56, 0.93), 0.08, 0.07), ((0, -0.62, 0.93), 0.12, 0.1), ((0, -0.69, 0.925), 0.11, 0.09),
            ((0, -0.75, 0.9), 0.07, 0.065), ((0, -0.84, 0.885), 0.055, 0.05), ((0, -0.93, 0.88), 0.04, 0.04),
            ((0, -0.965, 0.878), 0.025, 0.027)]
    loft("head", head, fur, "head", n=16, power=2.3, keel=0.8)
    loft("bridge", [((0, -0.72, 0.95), 0.04, 0.02), ((0, -0.85, 0.925), 0.035, 0.02), ((0, -0.93, 0.912), 0.02, 0.012)], dark, "head", n=10)
    k.ellipsoid((0, -0.968, 0.89), (0.03, 0.022, 0.024), nose, "head", 10, 6)
    loft("jaw", [((0, -0.66, 0.84), 0.07, 0.035), ((0, -0.78, 0.83), 0.05, 0.028), ((0, -0.91, 0.84), 0.032, 0.02)], pale, "jaw", n=12)
    k.ellipsoid((0, -0.8, 0.85), (0.035, 0.1, 0.012), mouth, "jaw", 10, 5)
    for s in (-1, 1):
        fur_ring((s * 0.11, -0.64, 0.88), 0.03, 0.06, pale, "head", 6, 0.025, 0.07, lean=(s * 0.8, 0.8, -0.3),
                 arc=(math.pi * 0.5, math.pi * 1.6) if s < 0 else (-math.pi * 0.6, math.pi * 0.5))  # cheek fur
        k.ellipsoid((s * 0.062, -0.73, 0.95), (0.024, 0.016, 0.015), eye, "head", 8, 5, rot=(0, 0, s * 20))
        k.ellipsoid((s * 0.066, -0.744, 0.952), (0.01, 0.007, 0.012), pupil, "head", 6, 4)
        k.ellipsoid((s * 0.06, -0.725, 0.968), (0.035, 0.025, 0.012), dark, "head", 8, 4, rot=(0, s * -20, s * 20))  # brow
        # Ears: flat triangles, dark outside, pale inside, tilted a little forward.
        b0, b1, tip = Vector((s * 0.04, -0.6, 1.0)), Vector((s * 0.11, -0.6, 0.97)), Vector((s * 0.09, -0.63, 1.14))
        back = Vector((0, 0.03, 0))
        mesh_part("ear", [b0, b1, tip, b0 + back, b1 + back, tip + back * 0.5],
                  [(0, 1, 2), (5, 4, 3), (0, 3, 4, 1), (1, 4, 5, 2), (2, 5, 3, 0)], dark, "head", smooth=False)
        mesh_part("earin", [b0 + Vector((s * 0.012, -0.004, 0.01)), b1 - Vector((s * 0.012, 0.004, -0.01)), tip + Vector((0, -0.004, -0.03))],
                  [(0, 1, 2) if s > 0 else (2, 1, 0)], pale, "head", smooth=False)
        k.rod((s * 0.03, -0.92, 0.865), (s * 0.03, -0.925, 0.835), 0.008, teeth, "head", 4, taper=0.2)  # upper fangs
        k.rod((s * 0.025, -0.885, 0.84), (s * 0.025, -0.89, 0.866), 0.007, teeth, "jaw", 4, taper=0.2)  # lower fangs
    fur_line([(0, -0.6, 1.0), (0, -0.53, 0.96)], fur, "head", 4, 0.03, 0.06, lean=(0, 1, 0.3))
    # Legs: front ones nearly straight with a fur feather behind the forearm; hind ones with
    # broad thighs and a backward-bending hock.
    for s, side in ((-1, "L"), (1, "R")):
        x = s * 0.11
        leg(f"front_{side}", [((x, -0.33, 0.68), 0.07, 0.11), ((x, -0.35, 0.5), 0.055, 0.075), ((x, -0.35, 0.33), 0.04, 0.045),
                              ((x, -0.35, 0.18), 0.03, 0.035), ((x, -0.36, 0.06), 0.028, 0.03)], 2, fur, claw, True, 4, 0.02,
            feather=((x, -0.31, 0.45), (x, -0.32, 0.32), fur))
        leg(f"hind_{side}", [((s * 0.1, 0.34, 0.68), 0.075, 0.13), ((s * 0.11, 0.28, 0.5), 0.065, 0.1), ((s * 0.11, 0.24, 0.36), 0.04, 0.05),
                             ((s * 0.11, 0.34, 0.18), 0.03, 0.035), ((s * 0.11, 0.37, 0.06), 0.028, 0.03)], 2, fur, claw, False, 4, 0.02,
            feather=((s * 0.11, 0.38, 0.55), (s * 0.11, 0.33, 0.4), fur))
    # Tail: bushy, hanging back and down, dark at the tip.
    pts = [(0, 0.5, 0.66), (0, 0.63, 0.6), (0, 0.74, 0.5), (0, 0.8, 0.36)]
    tail_parts([Vector(p) for p in pts] + [Vector((0, 0.82, 0.3))], (0.04, 0.06, 0.06, 0.045, 0.02), fur, fur, 0.1, 0.035)
    fur_line([(0, 0.79, 0.37), (0, 0.82, 0.3)], dark, "tail3", 6, 0.035, 0.08, lean=(0, 0.3, -1))
    return pts, (0, -0.42, 0.72), (0, -0.6, 0.86), (0, -0.66, 0.85), 0.6


def boar():
    hide = k.material("Hide", (120, 85, 62))
    dark = k.material("Bristle", (60, 44, 34))
    belly = k.material("Belly", (148, 112, 88))
    snout = k.material("Snout", (165, 110, 100))
    nostril = k.material("Nostril", (50, 30, 30))
    tusk = k.material("Tusk", (240, 232, 210))
    hoof = k.material("Hoof", (45, 35, 30))
    eye = k.material("Eye", (25, 18, 15))
    # Body: one heavy wedge, highest over the shoulders, sloping down to the rump.
    body = [((0, -0.5, 0.55), 0.13, 0.17), ((0, -0.4, 0.57), 0.21, 0.27), ((0, -0.24, 0.58), 0.25, 0.31),
            ((0, -0.05, 0.55), 0.245, 0.28), ((0, 0.15, 0.52), 0.225, 0.25), ((0, 0.33, 0.5), 0.2, 0.22),
            ((0, 0.47, 0.49), 0.15, 0.17), ((0, 0.55, 0.49), 0.06, 0.07)]
    loft("body", body, hide, "body", n=18, power=2.6, keel=0.8)
    loft("belly", [((0, -0.3, 0.34), 0.1, 0.04), ((0, 0.0, 0.3), 0.12, 0.05), ((0, 0.3, 0.32), 0.09, 0.04)], belly, "body", n=12)
    # The mane: a ridge of long dark bristles from between the ears to mid-back, and short
    # bristles all over the hide.
    loft("mane", [((0, -0.5, 0.73), 0.05, 0.03), ((0, -0.3, 0.88), 0.09, 0.04), ((0, -0.05, 0.84), 0.08, 0.035),
                  ((0, 0.2, 0.77), 0.05, 0.025)], dark, "body", n=12, power=2.5)
    for s in (-1, 0, 1):
        fur_line([(s * 0.04, -0.5, 0.75), (s * 0.05, -0.3, 0.9), (s * 0.05, -0.05, 0.86), (s * 0.03, 0.25, 0.77)], dark, "body",
                 16, 0.025, 0.14 if s == 0 else 0.1, lean=(s * 0.5, 0.6, 1), jitter=0.35)
    for i in range(70):
        y = rand(-0.4, 0.4)
        a = rand(-0.2, math.pi + 0.2)
        taper = 1 - max(0, y - 0.15) * 1.2  # the body narrows toward the rump
        rx, rz = 0.23 * taper, 0.26 * taper
        p = Vector((math.cos(a) * rx, y, 0.53 + math.sin(a) * rz * 0.9))
        shard(p, Vector((math.cos(a), 1.2, math.sin(a))), 0.012, rand(0.03, 0.05), dark, "body")
    # Head: a long wedge narrowing to the snout disc, small eyes high up, tusks curving up.
    loft("neck", [((0, -0.4, 0.58), 0.18, 0.22), ((0, -0.5, 0.55), 0.17, 0.2), ((0, -0.56, 0.53), 0.15, 0.17)], hide, "neck", n=14, power=2.4)
    head = [((0, -0.52, 0.56), 0.13, 0.16), ((0, -0.6, 0.53), 0.14, 0.15), ((0, -0.7, 0.48), 0.1, 0.11),
            ((0, -0.79, 0.43), 0.075, 0.08), ((0, -0.86, 0.4), 0.07, 0.068), ((0, -0.88, 0.395), 0.072, 0.07)]
    loft("head", head, hide, "head", n=16, power=2.6, keel=0.7)
    loft("snout", [((0, -0.875, 0.395), 0.074, 0.072), ((0, -0.9, 0.39), 0.07, 0.066)], snout, "head", n=16, power=2.2)
    loft("jaw", [((0, -0.58, 0.42), 0.09, 0.04), ((0, -0.7, 0.37), 0.065, 0.03), ((0, -0.8, 0.35), 0.045, 0.02)], hide, "jaw", n=12)
    for s in (-1, 1):
        k.ellipsoid((s * 0.025, -0.902, 0.395), (0.016, 0.006, 0.022), nostril, "head", 6, 4)
        k.ellipsoid((s * 0.085, -0.67, 0.56), (0.02, 0.014, 0.014), eye, "head", 6, 4)
        k.ellipsoid((s * 0.08, -0.66, 0.58), (0.035, 0.025, 0.01), dark, "head", 6, 4, rot=(0, s * -25, 0))  # brow
        # Ears: pointed, flopped forward and out.
        b0, b1, tip = Vector((s * 0.06, -0.5, 0.68)), Vector((s * 0.13, -0.5, 0.63)), Vector((s * 0.17, -0.6, 0.73))
        back = Vector((0, 0.025, 0))
        mesh_part("ear", [b0, b1, tip, b0 + back, b1 + back, tip + back * 0.5],
                  [(0, 1, 2), (5, 4, 3), (0, 3, 4, 1), (1, 4, 5, 2), (2, 5, 3, 0)], dark, "head", smooth=False)
        # Tusks: from the jaw's corner, curving up and back.
        t0, t1, t2 = Vector((s * 0.055, -0.8, 0.37)), Vector((s * 0.1, -0.84, 0.42)), Vector((s * 0.105, -0.8, 0.5))
        k.rod(t0, t1, 0.017, tusk, "head", 6, taper=0.75)
        k.rod(t1, t2, 0.0128, tusk, "head", 6, taper=0.15)
        fur_line([(s * 0.13, -0.58, 0.48), (s * 0.12, -0.68, 0.42)], dark, "head", 4, 0.02, 0.05, lean=(s * 0.6, 1, -0.3))  # whiskery cheeks
    fur_line([(0, -0.6, 0.66), (0, -0.52, 0.72)], dark, "head", 5, 0.025, 0.08, lean=(0, 0.8, 1))
    # Legs: short and sturdy, big hams behind, cloven hooves.
    for s, side in ((-1, "L"), (1, "R")):
        x = s * 0.14
        leg(f"front_{side}", [((x, -0.3, 0.5), 0.09, 0.14), ((x, -0.31, 0.32), 0.065, 0.09), ((x, -0.31, 0.2), 0.042, 0.05),
                              ((x, -0.32, 0.1), 0.035, 0.04), ((x, -0.32, 0.05), 0.035, 0.04)], 2, hide, hoof, True, 0, 0.032, hoof=True)
        leg(f"hind_{side}", [((s * 0.13, 0.32, 0.52), 0.1, 0.17), ((s * 0.135, 0.28, 0.34), 0.08, 0.12), ((s * 0.135, 0.26, 0.22), 0.042, 0.05),
                             ((s * 0.135, 0.31, 0.1), 0.035, 0.04), ((s * 0.135, 0.32, 0.05), 0.035, 0.04)], 2, hide, hoof, False, 0, 0.032, hoof=True)
    pts = [(0, 0.53, 0.56), (0, 0.58, 0.5), (0, 0.6, 0.43), (0, 0.6, 0.37)]
    tail_parts([Vector(p) for p in pts] + [Vector((0, 0.6, 0.35))], (0.018, 0.015, 0.013, 0.011, 0.01), hide)
    fur_line([(0, 0.6, 0.37), (0, 0.6, 0.33)], dark, "tail3", 5, 0.02, 0.06, lean=(0, 0.2, -1))
    return pts, (0, -0.42, 0.56), (0, -0.55, 0.52), (0, -0.62, 0.42), 0.45


tail, neck_at, head_at, jaw_at, body_z = (boar if kind == "boar" else wolf)()
bones = [("root", (0, 0, 0), None), ("body", (0, 0.1, body_z), "root"), ("neck", neck_at, "body"), ("head", head_at, "neck"),
         ("jaw", jaw_at, "head")]
for i in range(3):
    bones.append((f"tail{i + 1}", tail[i], "body" if i == 0 else f"tail{i}"))
for up, low, _, hip, knee in LEGS:
    bones += [(up, tuple(hip), "body"), (low, tuple(knee), up)]
k.bones(bones)
k.bind({"wolf": "Wolf", "snowwolf": "SnowWolf", "direwolf": "DireWolf"}.get(kind, "Boar"))
key = k.key
# Bones point up, and in a bone's own space Z is forward: rotating about X tilts what's above a
# pivot forward, lowers what's in front of it (a nodding head, an opening jaw), raises what's
# behind it (the tail) and swings what hangs below it backward (legs come forward with -X).
FRONT = [(u, l) for u, l, f, _, _ in LEGS if f]
HIND = [(u, l) for u, l, f, _, _ in LEGS if not f]


def legs(frame, swings):
    """swings: {upper bone: (swing, bend)}; swing forward is negative, bend lifts the foot."""
    for (up, low, front, _, _) in LEGS:
        s, b = swings.get(up, (0, 0))
        key(frame, up, rot=(s, 0, 0))
        key(frame, low, rot=(b if front else -b, 0, 0))


def tail_pose(frame, lift, wag=0.0):
    for i in range(3):
        key(frame, f"tail{i + 1}", rot=(lift * (1 - 0.3 * i), 0, wag * (1 + 0.4 * i)))


def cycle(length, pairs, swing, bend, body_bob=0.0, pitch=0.0):
    """A looping gait: pairs is a list of (upper bones, phase 0..1); each foot swings forward,
    lifting halfway, then pushes back on the ground."""
    steps = 8
    for i in range(steps + 1):
        f = 1 + round((length - 1) * i / steps)
        pose = {}
        for bones_, phase in pairs:
            t = (i / steps + phase) % 1
            s = swing * math.cos(2 * math.pi * t)  # +swing (back) at t=0, -swing (forward) at t=0.5
            b = bend * max(0, math.sin(2 * math.pi * t))  # lifted while swinging forward
            for name in bones_:
                pose[name] = (s, b)
        legs(f, pose)
        t = i / steps
        key(f, "root", loc=(0, body_bob * abs(math.sin(2 * math.pi * t)), 0))
        key(f, "body", rot=(pitch * math.sin(2 * math.pi * t), 0, 0))
        tail_pose(f, 10 + 6 * math.sin(2 * math.pi * t), 6 * math.sin(2 * math.pi * t))
        key(f, "head", rot=(-pitch * 0.6 * math.sin(2 * math.pi * t), 0, 0))


def idle():
    for f, a in ((1, 0), (30, 1), (60, 0)):
        key(f, "body", loc=(0, 0.006 * a, 0))  # breathing
    if kind != "boar":
        for f, w in ((1, 0), (10, 12), (20, -12), (30, 12), (40, -12), (50, 0), (60, 0)):
            tail_pose(f, 5, w)
        for f, y in ((1, 0), (15, 0), (25, 18), (40, 18), (50, 0), (60, 0)):
            key(f, "head", rot=(-5 if y else 0, y, 0))
            key(f, "neck", rot=(0, y * 0.4, 0))
    else:
        for f, n in ((1, 0), (12, 14), (16, 10), (20, 14), (24, 10), (34, 0), (60, 0)):  # snuffling at the ground
            key(f, "neck", rot=(n, 0, 0))
            key(f, "head", rot=(n * 0.6, 0, 0))
            key(f, "jaw", rot=(4 if n == 10 else 0, 0, 0))
        for f, w in ((1, 0), (40, 0), (44, 25), (48, -25), (52, 25), (56, 0), (60, 0)):
            tail_pose(f, 0, w)


def walk():
    fl, fr = [n for n, _ in FRONT if n.endswith("L_up")], [n for n, _ in FRONT if n.endswith("R_up")]
    hl, hr = [n for n, _ in HIND if n.endswith("L_up")], [n for n, _ in HIND if n.endswith("R_up")]
    cycle(25, [(fl + hr, 0), (fr + hl, 0.5)], 22, 40, body_bob=0.02)


def run():
    front = [n for n, _ in FRONT]
    hind = [n for n, _ in HIND]
    cycle(17, [(front, 0), (hind, 0.45)], 38, 60, body_bob=0.07, pitch=7)


def attack():
    if kind != "boar":
        # Crouch, leap forward with the jaws wide open, snap them shut, land.
        legs(7, {u: (12, 30) for u, _ in FRONT} | {u: (-10, 40) for u, _ in HIND})
        key(7, "root", loc=(0, -0.12, -0.05))
        key(7, "body", rot=(-6, 0, 0))
        key(7, "neck", rot=(-10, 0, 0))
        tail_pose(7, 5)
        legs(12, {u: (-45, 20) for u, _ in FRONT} | {u: (35, 0) for u, _ in HIND})
        key(12, "root", loc=(0, 0.15, 0.45))
        key(12, "body", rot=(-10, 0, 0))
        key(12, "neck", rot=(10, 0, 0))
        key(12, "jaw", rot=(40, 0, 0))
        tail_pose(12, 25)
        legs(15, {u: (-15, 10) for u, _ in FRONT} | {u: (15, 10) for u, _ in HIND})
        key(15, "root", loc=(0, 0, 0.5))
        key(15, "neck", rot=(15, 0, 0))
        key(15, "head", rot=(10, 0, 8))
        key(15, "jaw", rot=(0, 0, 0))
        legs(22, {})
        key(22, "root", loc=(0, 0, 0.2))
    else:
        # Head down, charge, then toss the tusks up and back.
        key(7, "neck", rot=(20, 0, 0))
        key(7, "head", rot=(10, 0, 0))
        key(7, "root", loc=(0, -0.05, -0.08))
        legs(7, {u: (10, 0) for u, _ in FRONT} | {u: (15, 20) for u, _ in HIND})
        key(12, "root", loc=(0, 0, 0.55))
        legs(12, {u: (-30, 40) for u, _ in FRONT} | {u: (30, 0) for u, _ in HIND})
        key(12, "neck", rot=(25, 0, 0))
        key(12, "head", rot=(10, 0, 0))
        key(15, "root", loc=(0, 0.05, 0.6))
        key(15, "neck", rot=(-25, 0, 0))
        key(15, "head", rot=(-25, 0, 0))
        key(15, "body", rot=(-8, 0, 0))
        legs(15, {u: (-10, 0) for u, _ in FRONT} | {u: (10, 0) for u, _ in HIND})
        key(22, "root", loc=(0, 0, 0.25))
        key(22, "neck", rot=(5, 0, 0))
    tail_pose(22, 10)


def hit():
    key(5, "root", loc=(0, -0.04, -0.12))
    key(5, "body", rot=(-8, 0, 6))
    key(5, "neck", rot=(-15, 12, 0))
    key(5, "jaw", rot=(18, 0, 0))
    tail_pose(5, -15)
    legs(5, {u: (10, 15) for u, _ in FRONT})


def lying_legs(frame, swings):
    """Like legs(), with a sideways swing too: swings is {upper bone: (swing, bend, across)};
    across swings the leg toward +X (negative: toward -X, the side the animal falls onto, see
    death)."""
    for (up, low, front, _, _) in LEGS:
        sw, b, across = swings.get(up, (0, 0, 0))
        key(frame, up, rot=(sw, 0, across))
        key(frame, low, rot=(b if front else -b, 0, 0))


def death():
    """Collapses instead of tipping over stiffly: staggers with the head dropping and the front
    legs buckling, sinks onto its belly as the hind legs give way, rolls onto its left side
    with the legs on top falling across the ones below and the paws curled, then the head
    settles on the ground and the tail goes limp."""
    upper = {u for u, l, f, h, _ in LEGS if h.x > 0}  # rolling onto its left side, the right legs end up on top
    key(1, "root", rot=(0, 0, 0), loc=(0, 0, 0))
    key(6, "neck", rot=(-25, 0, 0))
    key(6, "head", rot=(10, 0, 0))
    key(6, "jaw", rot=(22, 0, 0))
    key(6, "root", loc=(0, -body_z * 0.08, -0.03))
    lying_legs(6, {u: (15, 45, 0) for u, _ in FRONT} | {u: (-6, 12, 0) for u, _ in HIND})
    tail_pose(6, -12)
    # On its belly: the front legs folded under, the hind legs giving way.
    key(14, "root", rot=(0, 0, 8), loc=(0, -body_z * 0.55, -0.04))
    key(14, "body", rot=(4, 0, 0))
    key(14, "neck", rot=(-30, 0, -6))
    key(14, "head", rot=(14, 0, 0))
    lying_legs(14, {u: (35, 95, 0) for u, _ in FRONT} | {u: (-40, 85, 0) for u, _ in HIND})
    tail_pose(14, -18)
    # Rolling onto its side; the legs on top swing down across the ones below.
    key(24, "root", rot=(0, 0, 84), loc=(0, body_z * 0.3, 0))
    key(24, "body", rot=(0, 0, 0))
    key(24, "neck", rot=(-5, 0, -18))
    key(24, "head", rot=(6, 0, -12))
    key(24, "jaw", rot=(14, 0, 0))
    lying_legs(24, {u: (-15 if f else 18, 30, -34 if u in upper else -4) for u, l, f, h, _ in LEGS})
    tail_pose(24, -20, 8)
    # Settling: the head sinks to the ground, the paws curl a little more, the tail lies limp.
    key(40, "root", rot=(0, 0, 86), loc=(0, body_z * 0.29, 0))
    key(40, "neck", rot=(4, 0, -24))
    key(40, "head", rot=(2, 0, -16))
    key(40, "jaw", rot=(10, 0, 0))
    lying_legs(40, {u: (-10 if f else 14, 38, -42 if u in upper else -6) for u, l, f, h, _ in LEGS})
    tail_pose(40, -22, 10)


k.action("Idle", 61, idle)
k.action("Walk", 25, walk)
k.action("Run", 17, run)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 40, death)
k.export(args[1])
if len(args) > 2:
    k.preview(args[2], distance=3.0, target_z=0.5)
