"""Builds a snake: a banded body coiled on the ground, rising into a neck and head, rigged and
animated. Two variants: the cobra (raised high, with a hood) and the sand snake (lower, sandy,
no hood).

    blender -b --python tools/models/snake.py -- cobra|sandsnake OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; head toward -Y (glTF +Z). The body is one tube that
bends smoothly: the coil follows the root bone, the neck a chain of four bones. Animations:
Idle (sways), Walk (slithers), Attack (rears back and strikes), HitReact and Death (the neck
sinks to the ground).
"""

import math
import sys
from pathlib import Path

import bmesh
import bpy
from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
variant = args[0]
cobra = variant == "cobra"
k = Kit()

if cobra:
    SKIN = k.material("Skin", (80, 92, 48))
    BAND = k.material("Band", (40, 44, 30))
    PALE = k.material("Belly", (205, 185, 120))
    NECK_TOP = Vector((0, -0.24, 0.72))
else:
    SKIN = k.material("Skin", (205, 170, 112))
    BAND = k.material("Band", (125, 88, 55))
    PALE = k.material("Belly", (232, 212, 165))
    NECK_TOP = Vector((0, -0.32, 0.4))
DARK = k.material("Dark", (25, 22, 20))
EYE = k.material("Eye", (230, 180, 50))
TONGUE = k.material("Tongue", (190, 40, 50))
RADIUS = 0.08
CENTER = Vector((0, 0.2, 0))  # middle of the coil

# ---- the body path, tail to head ----
path = []  # (point, part): part "coil" or a neck parameter 0..1
turns, steps = 1.5, 90
for i in range(steps):
    f = i / (steps - 1)
    a = math.radians(-90) + (1 - f) * turns * 2 * math.pi  # a spiral from the tail inside to the front (-Y)
    rho = 0.1 + 0.26 * f
    r = 0.01 + (RADIUS - 0.01) * f ** 0.7  # thickens all the way to the neck
    path.append((CENTER + Vector((rho * math.cos(a), rho * math.sin(a), r)), "coil", r))
p0 = path[-1][0]
p1 = p0 + Vector((-0.02, -0.14, 0.05))
p3 = NECK_TOP
p2 = p3 + Vector((0, 0.12, -0.3 if cobra else -0.15))
neck = []
for i in range(1, 25):
    t = i / 24
    q = (1 - t) ** 3 * p0 + 3 * (1 - t) ** 2 * t * p1 + 3 * (1 - t) * t ** 2 * p2 + t ** 3 * p3
    neck.append(q)
    path.append((q, t, RADIUS - 0.015 * t))

# ---- the tube ----
bm = bmesh.new()
SIDES = 10
rings = []
normal = Vector((0, 0, 1))
for i, (p, part, r) in enumerate(path):
    tangent = (path[min(i + 1, len(path) - 1)][0] - path[max(i - 1, 0)][0]).normalized()
    normal = (normal - tangent * normal.dot(tangent)).normalized()  # parallel transport
    side = tangent.cross(normal)
    ring = []
    for j in range(SIDES):
        a = 2 * math.pi * j / SIDES
        ring.append(bm.verts.new(p + r * (math.cos(a) * normal + math.sin(a) * side)))
    rings.append(ring)
bm.verts.ensure_lookup_table()
for i in range(len(rings) - 1):
    band = (i // 5) % 2 == 1  # stripes every 5 rings
    for j in range(SIDES):
        f = bm.faces.new((rings[i][j], rings[i][(j + 1) % SIDES], rings[i + 1][(j + 1) % SIDES], rings[i + 1][j]))
        f.material_index = 1 if band else 0
        f.smooth = True
bm.faces.new(list(reversed(rings[0])))  # tail cap
mesh = bpy.data.meshes.new("Body")
bm.to_mesh(mesh)
body = bpy.data.objects.new("Body", mesh)
bpy.context.scene.collection.objects.link(body)
mesh.materials.append(SKIN)
mesh.materials.append(BAND)

# Weights: the coil follows the root, the neck blends along its four bones.
NECK_BONES = ["neck1", "neck2", "neck3", "neck4"]
groups = {n: body.vertex_groups.new(name=n) for n in ["root"] + NECK_BONES}
for i, (p, part, r) in enumerate(path):
    idx = [v.index for v in rings[i]]
    if part == "coil":
        groups["root"].add(idx, 1.0, "REPLACE")
        continue
    b = part * len(NECK_BONES)
    for n, name in enumerate(NECK_BONES):
        w = max(0.0, 1 - abs(b - (n + 0.5)))
        if n == 0 and b < 0.5:
            w = 0.5 + b
            groups["root"].add(idx, 1 - w, "REPLACE")
        if n == len(NECK_BONES) - 1 and b > len(NECK_BONES) - 0.5:
            w = 1.0
        if w > 0:
            groups[name].add(idx, w, "REPLACE")
k.parts.append((body, None))

# ---- head (and hood) ----
fwd = Vector((0, -1, 0))
hc = NECK_TOP + fwd * 0.08 + Vector((0, 0, 0.01))
k.ellipsoid(hc, (0.075, 0.12, 0.05), SKIN, "head")
k.ellipsoid(hc + Vector((0, 0.01, -0.02)), (0.065, 0.11, 0.035), PALE, "head")  # jaw
for s in (-1, 1):
    k.ellipsoid(hc + Vector((s * 0.055, -0.04, 0.025)), (0.018, 0.02, 0.018), EYE, "head", 8, 5)
    k.ellipsoid(hc + Vector((s * 0.06, -0.045, 0.027)), (0.009, 0.012, 0.012), DARK, "head", 6, 4)
    k.rod(hc + Vector((0, -0.16, -0.01)), hc + Vector((s * 0.025, -0.22, -0.015)), 0.005, TONGUE, "head", 4)
k.rod(hc + Vector((0, -0.1, -0.01)), hc + Vector((0, -0.16, -0.01)), 0.007, TONGUE, "head", 4)
if cobra:
    # The hood flares out to both sides of the neck, just below the head.
    hood = NECK_TOP + Vector((0, 0.03, -0.12))
    k.ellipsoid(hood, (0.21, 0.04, 0.2), SKIN, "neck4", rot=(-8, 0, 0))
    k.ellipsoid(hood + Vector((0, -0.02, -0.02)), (0.14, 0.035, 0.15), PALE, "neck4", rot=(-8, 0, 0))
    for s in (-1, 1):  # the spectacle marks on the back
        k.ellipsoid(hood + Vector((s * 0.08, 0.035, 0.03)), (0.045, 0.015, 0.04), DARK, "neck4", 10, 6)
        k.ellipsoid(hood + Vector((s * 0.08, 0.042, 0.03)), (0.022, 0.012, 0.02), PALE, "neck4", 8, 5)

# ---- rig ----
cuts = [p0] + [neck[int(len(neck) * (n + 1) / 4) - 1] for n in range(4)]
k.bones([("root", CENTER, None)] +
        [(name, cuts[n], "root" if n == 0 else NECK_BONES[n - 1], cuts[n + 1]) for n, name in enumerate(NECK_BONES)] +
        [("head", NECK_TOP, "neck4", NECK_TOP + fwd * 0.2)])
k.bind("Cobra" if cobra else "SandSnake")
key = k.key


def neck(frame, bends, sway=0.0, head=0.0):
    """bends: forward bend per neck bone in degrees (negative rears back); sway: sideways;
    head: the head's pitch against the horizon in degrees (positive = nose up), whatever the
    neck does."""
    for name, b in zip(NECK_BONES, bends):
        key(frame, name, rot=(b, 0, sway))
    # Where would the head point if it just followed the neck? Its rest direction turned by
    # how far the top neck bone moved from its rest pose.
    bpy.context.scene.frame_set(frame)
    top = k.arm.pose.bones["neck4"]
    turn = top.matrix.to_3x3() @ top.bone.matrix_local.to_3x3().inverted()
    follows = turn @ fwd
    lift = math.degrees(math.asin(max(-1, min(1, follows.z))))
    # The head bone's X axis is world -X, so a positive turn about it lifts the nose.
    key(frame, "head", rot=(head - lift, 0, 0))


def idle():
    for f, s in ((1, 0), (16, 5), (31, 0), (46, -5), (61, 0)):
        neck(f, (0, 0, 0, 0), s)


def walk():
    for f, s in ((1, 0), (7, 9), (13, 0), (19, -9), (25, 0)):
        key(f, "root", rot=(0, s * 0.8, 0))
        neck(f, (0, 0, 0, 0), -s)


def attack():
    neck(8, (-12, -14, -10, -6), 0, 15)  # rear back, nose up
    neck(13, (14, 16, 12, 6), 0, -10)  # strike
    key(13, "root", loc=(0, 0, 0.15))
    neck(20, (6, 6, 4, 0))


def hit():
    neck(5, (-10, -12, -8, 0), 6, 12)


def death():
    neck(12, (10, 10, 6, 2), 6, 0)
    neck(24, (20, 18, 12, 0), 10, -5)
    neck(36, (21, 19, 13, 0), 10, -8)  # the head lies flat
    key(36, "root", rot=(0, 10, 0))


k.action("Idle", 61, idle)
k.action("Walk", 25, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 16, hit)
k.action("Death", 36, death)
k.export(args[1])
if len(args) > 2:
    k.preview(args[2], distance=2.4, target_z=0.35)
