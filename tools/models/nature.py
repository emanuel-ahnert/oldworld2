"""Builds the map scenery: trees, palms, bushes, plants, flowers, grass, rocks, bones, the
ancient desert's sphinx and pharaoh statue, the North's snowy pines, rocks and shrubs, frosted
ferns, bare twigs and ice crystals, the Mediterranean's olive trees, cypresses, limestone
rocks, lavender, sage, poppies, broken columns and amphorae, and the unholy lands' dead trees,
dark pines and rocks, thorn bushes, gravestones and mushrooms.

    blender -b --python tools/models/nature.py -- OUT_DIR [PREVIEW_DIR] [NAME,NAME...]

Runs inside Blender without a window and writes one glTF per model to OUT_DIR. Models are
made at their size on the map (1 unit = 1 tile), standing on the origin. Thin parts (leaves,
fronds, petals, grass blades) use materials named "Thin...", which get no ink outline in the
game (r3d.foliage). Every model is made from a fixed seed, so a rebuild gives the same shapes.
"""

import math
import random
import sys
from pathlib import Path

import bmesh
import bpy
from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
out_dir = Path(args[0])
preview = Path(args[1]) if len(args) > 1 and args[1] else None

BARK = (110, 75, 50)
BARK_GREY = (120, 105, 90)
LEAF = (80, 128, 40)
LEAF_DARK = (55, 98, 32)
LEAF_LIGHT = (108, 150, 48)
PINE = (45, 100, 60)
PINE_DARK = (30, 80, 50)
FROND = (100, 138, 48)
FROND_DARK = (72, 108, 38)
PALM_BARK = (150, 115, 75)
ROCK = (150, 132, 112)
BONE = (235, 225, 200)
SNOW = (240, 244, 250)
ICE = (170, 215, 240)


# ---- shapes ----

def blob(k, center, radii, mat, rnd, jitter=0.12, subdiv=2):
    """A lumpy ball: an icosphere with its vertices pushed in and out a little."""
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=subdiv, radius=1, location=center)
    o = bpy.context.object
    o.scale = radii
    for v in o.data.vertices:
        v.co *= 1 + rnd.uniform(-jitter, jitter)
    return k.finish(o, mat, None)


def mesh_object(name, bm, mat, k, smooth=True):
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    return k.finish(o, mat, None, smooth=smooth)


def strip(k, points, widths, side, mat, fold=0.25, thickness=0.012):
    """A thin leaf along a center line: widths per point, spread along side, with a crease
    down the middle (fold, as a share of the width). Solidified so both sides render."""
    bm = bmesh.new()
    side = Vector(side).normalized()
    rows = []
    for i, (p, w) in enumerate(zip(points, widths)):
        p = Vector(p)
        tangent = (Vector(points[min(i + 1, len(points) - 1)]) - Vector(points[max(i - 1, 0)])).normalized()
        up = side.cross(tangent).normalized()
        rows.append([bm.verts.new(p - side * w), bm.verts.new(p + up * w * fold), bm.verts.new(p + side * w)])
    for a, b in zip(rows, rows[1:]):
        for j in range(2):
            bm.faces.new((a[j], a[j + 1], b[j + 1], b[j]))
    bmesh.ops.remove_doubles(bm, verts=bm.verts, dist=1e-5)
    bmesh.ops.solidify(bm, geom=bm.faces[:], thickness=thickness)
    return mesh_object("leaf", bm, mat, k)


def arc(start, direction, length, rise, droop, n=8):
    """Points from start going out along a horizontal direction, rising then drooping."""
    d = Vector((direction[0], direction[1], 0)).normalized()
    pts = []
    for i in range(n + 1):
        t = i / n
        pts.append(Vector(start) + d * length * t + Vector((0, 0, rise * t - droop * t * t)))
    return pts


def leaf_widths(n, width, tip=0.1):
    """Widths for a leaf: narrow at the base, widest a third of the way, pointed at the tip."""
    return [width * max(tip, math.sin(math.pi * (i / n) ** 0.8)) for i in range(n + 1)]


def branch(k, pts, r0, r1, mat, verts=7):
    """A tapering limb through points."""
    n = len(pts) - 1
    for i, (a, b) in enumerate(zip(pts, pts[1:])):
        ra = r0 + (r1 - r0) * i / n
        rb = r0 + (r1 - r0) * (i + 1) / n
        k.rod(a, b, ra, mat, None, verts, taper=rb / ra)


def chisel(o, rnd, cuts):
    """Flattens a ball into a faceted stone: vertices beyond random planes move onto them."""
    for _ in range(cuts):
        n = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(-0.3, 1))).normalized()
        d = rnd.uniform(0.6, 0.85)
        for v in o.data.vertices:
            over = v.co.dot(n) - d
            if over > 0:
                v.co -= n * over


def sink(o, depth):
    """Flattens the bottom of a stone so it sits in the ground."""
    lo = min((o.matrix_world @ v.co).z for v in o.data.vertices)
    cut = lo + depth
    inv = o.matrix_world.inverted()
    for v in o.data.vertices:
        w = o.matrix_world @ v.co
        if w.z < cut:
            w.z = cut
            v.co = inv @ w
    o.location.z -= cut


# ---- trees ----

def leaf_cards(k, center, radius, mats, rnd, count, length=0.28, width=0.09):
    """Small leaves sticking out of a clump's surface, so its outline is ragged, not round."""
    for i in range(count):
        d = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(-0.4, 1))).normalized()
        base = Vector(center) + d * radius * 0.85
        side = d.cross(Vector((0, 0, 1)))
        if side.length < 0.1:
            side = Vector((1, 0, 0))
        tip = base + d * length + Vector((0, 0, -length * 0.35))
        pts = [base.lerp(tip, t / 4) for t in range(5)]
        strip(k, pts, leaf_widths(4, width, 0.15), side, mats[i % len(mats)], fold=0.3, thickness=0.008)


def limbs(k, start, direction, length, radius, depth, bark, rnd, ends, spread=0.55, rise=0.35):
    """Branches that fork: each limb splits in two (sometimes three) until depth runs out;
    the ends are where leaf clumps go."""
    d = Vector(direction).normalized()
    mid = start + d * length * 0.5 + Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), 0.3)) * length * 0.12
    end = start + d * length
    branch(k, [start, mid, end], radius, radius * 0.65, bark, 6)
    if depth == 0:
        ends.append((end, d))
        return
    for _ in range(rnd.choice((2, 2, 3))):
        nd = d + Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), 0)) * spread + Vector((0, 0, rise))
        limbs(k, end, nd, length * rnd.uniform(0.6, 0.75), radius * 0.62, depth - 1, bark, rnd, ends, spread, rise)


def round_tree(k, rnd, height=4.6, roots=False):
    bark = k.material("Bark", BARK_GREY if roots else BARK)
    greens = [k.material("Leaves", LEAF), k.material("LeavesDark", LEAF_DARK), k.material("LeavesLight", LEAF_LIGHT)]
    cards = [k.material("ThinLeaf", LEAF), k.material("ThinLeafDark", LEAF_DARK), k.material("ThinLeafLight", LEAF_LIGHT)]
    # A trunk that bends a little on its way up.
    r = 0.26 if roots else 0.19
    fork = height * (0.3 if roots else 0.32)
    lean = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), 0)) * height * 0.04
    trunk = [Vector((0, 0, -0.1))]
    for i in range(1, 5):
        t = i / 4
        trunk.append(Vector((0, 0, fork * t)) + lean * t + Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), 0)) * 0.04)
    branch(k, trunk, r, r * 0.7, bark, 9)
    # Roots flaring into the ground; old trees have long ones crawling over it.
    for i in range(6 if roots else 5):
        a = i * 2 * math.pi / (6 if roots else 5) + rnd.uniform(-0.3, 0.3)
        d = Vector((math.cos(a), math.sin(a), 0))
        reach = rnd.uniform(0.8, 1.05) if roots else rnd.uniform(0.32, 0.42)
        branch(k, [Vector((0, 0, 0.45 if roots else 0.3)) + d * r * 0.4, d * reach * 0.45 + Vector((0, 0, 0.1)), d * reach + Vector((0, 0, -0.06))],
               r * 0.55, r * 0.15, bark, 6)
    # Main limbs fork twice; a clump of leaves sits on every end.
    ends = []
    top = trunk[-1]
    count = rnd.randint(3, 4)
    for i in range(count):
        a = i * 2 * math.pi / count + rnd.uniform(-0.35, 0.35)
        d = Vector((math.cos(a), math.sin(a), rnd.uniform(0.8, 1.2)))
        limbs(k, top - Vector((0, 0, 0.15)), d, height * 0.19, r * 0.6, 2, bark, rnd, ends, rise=0.45)
    limbs(k, top, Vector((0, 0, 1)), height * 0.2, r * 0.55, 1, bark, rnd, ends, rise=0.6)  # the leader, up the middle
    lo = min(e.z for e, _ in ends)
    hi = max(e.z for e, _ in ends)
    clump = height * 0.08
    for end, d in ends:
        t = (end.z - lo) / max(hi - lo, 0.01)  # higher clumps catch more light
        shade = 2 if t > 0.7 else 0 if t > 0.3 else 1
        c = end + Vector((0, 0, clump * 0.3))
        s = clump * rnd.uniform(0.9, 1.25)
        blob(k, c, (s, s, s * 0.85), greens[shade], rnd, jitter=0.18)
        for _ in range(2):  # smaller lumps around it
            o = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(-0.3, 0.6))).normalized() * s * 0.75
            ss = s * rnd.uniform(0.5, 0.7)
            blob(k, c + o, (ss, ss, ss * 0.85), greens[(shade + rnd.randint(0, 1)) % 3], rnd, jitter=0.2, subdiv=1)
        leaf_cards(k, c, s, [cards[shade], cards[(shade + 1) % 3]], rnd, 7)
    # A dense middle so you don't see through the crown.
    core = (sum((e for e, _ in ends), Vector()) / len(ends)) * 0.8 + top * 0.2
    span = max((e - core).length for e, _ in ends)
    blob(k, core, (span * 0.55, span * 0.55, span * 0.55), greens[1], rnd, jitter=0.2)


def needle_tuft(k, center, d, size, mat, cards, rnd, sprays=4, snow=None):
    """A drooping clump of needles on a bough: a lumpy, flattened ball stretched along the
    bough, with jagged sprays hanging from its edges; with snow (a material), a cap of snow
    lying on top."""
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=1, radius=1, location=center)
    o = bpy.context.object
    for v in o.data.vertices:
        v.co *= 1 + rnd.uniform(-0.25, 0.25)
    o.scale = (size * 1.8, size * 0.85, size * 0.42)
    o.rotation_euler = (0, math.radians(rnd.uniform(10, 25)), math.atan2(d.y, d.x))
    k.finish(o, mat, None)
    if snow:
        blob(k, Vector(center) + Vector((0, 0, size * 0.32)) + d * size * 0.2, (size * 1.5, size * 0.75, size * 0.22), snow, rnd, 0.15, 1)
    side = Vector((-d.y, d.x, 0))
    for i in range(sprays):
        a = rnd.uniform(-1, 1)
        base = Vector(center) + side * a * size * 0.6 + d * rnd.uniform(0, 1.4) * size
        out = (d * 0.9 + side * a * 0.6).normalized()
        n = 4
        pts = [base + out * size * 1.3 * t / n + Vector((0, 0, -size * 0.4 * (t / n) ** 1.5)) for t in range(n + 1)]
        w = [size * 0.3 * (1 - t / n) * (0.5 if t % 2 else 1) + 0.006 for t in range(n + 1)]
        strip(k, pts, w, out.cross(Vector((0, 0, 1))), cards[i % len(cards)], fold=0.4, thickness=0.008)


def pine(k, rnd, height=4.6, tufts=3, sprays=2, snow=False, gloomy=False):
    """tufts: needle tufts on a long bough (one fewer on short ones); sprays: per tuft. snow:
    a winter pine, darker and bluer, with snow lying on its boughs. gloomy: the unholy lands'
    pine, nearly black-green, its bark grey."""
    bark = k.material("Bark", BARK_GREY if gloomy else BARK)
    green, dark, light = (PINE, PINE_DARK, (62, 122, 72))
    if snow:
        green, dark, light = (38, 84, 66), (26, 64, 52), (52, 102, 80)
    if gloomy:
        green, dark, light = (40, 52, 46), (26, 34, 32), (58, 70, 62)
    greens = [k.material("Needles", green), k.material("NeedlesDark", dark), k.material("NeedlesLight", light)]
    cards = [k.material("ThinNeedles", green), k.material("ThinNeedlesDark", dark), k.material("ThinNeedlesLight", light)]
    white = k.material("Snow", SNOW) if snow else None
    k.rod((0, 0, -0.1), (0, 0, height * 0.95), 0.16, bark, None, 8, taper=0.2)
    for i in range(5):  # root flare
        a = i * 2 * math.pi / 5 + rnd.uniform(-0.3, 0.3)
        d = Vector((math.cos(a), math.sin(a), 0))
        branch(k, [Vector((0, 0, 0.28)), d * 0.18 + Vector((0, 0, 0.05)), d * 0.34 + Vector((0, 0, -0.05))], 0.08, 0.025, bark, 5)
    tiers = rnd.randint(6, 7)
    for i in range(tiers):
        t = i / tiers
        z = height * (0.17 + 0.7 * t)
        r = height * 0.25 * (1 - 0.8 * t) + 0.18
        # A small dark core, only to close the gaps between the boughs.
        bpy.ops.mesh.primitive_cone_add(vertices=8, radius1=r * 0.45, radius2=0, depth=height * 0.16, location=(0, 0, z + height * 0.05))
        k.finish(bpy.context.object, greens[1], None)
        count = max(5, int(10 * (1 - 0.45 * t)))
        turn = rnd.uniform(0, 2 * math.pi)
        for j in range(count):
            a = turn + j * 2 * math.pi / count + rnd.uniform(-0.2, 0.2)
            d = Vector((math.cos(a), math.sin(a), 0))
            reach = r * rnd.uniform(0.9, 1.1)
            zz = z + rnd.uniform(-0.06, 0.06) * height
            # The bough itself, sagging a little toward its end.
            pts = [Vector((0, 0, zz + 0.1)), d * reach * 0.5 + Vector((0, 0, zz + 0.02)), d * reach + Vector((0, 0, zz - 0.12))]
            branch(k, pts, 0.035, 0.012, bark, 5)
            # Tufts along it: dark near the trunk, lighter at the tip and near the top.
            n = tufts if reach > 0.6 else tufts - 1
            for m in range(n):
                f = (m + 1) / n
                c = d * reach * (0.25 + 0.7 * f) + Vector((0, 0, zz + 0.05 - 0.15 * f * f))
                size = height * 0.034 * (0.8 + 0.5 * (1 - t)) * rnd.uniform(0.85, 1.15)
                shade = 2 if (f > 0.7 or t > 0.75) else 0 if f > 0.3 else 1
                needle_tuft(k, c, d, size * (1 + 0.25 * (4 - tufts)), greens[shade], [cards[shade], cards[(shade + 1) % 3]], rnd, sprays,
                            white if (m == n - 1 or rnd.random() < 0.4) else None)
    # The top: a spire of needles.
    bpy.ops.mesh.primitive_cone_add(vertices=8, radius1=0.16, radius2=0, depth=height * 0.15, location=(0, 0, height * 0.94))
    k.finish(bpy.context.object, greens[2], None)
    for i in range(5):
        a = i * 2 * math.pi / 5
        d = Vector((math.cos(a), math.sin(a), 0))
        needle_tuft(k, Vector((0, 0, height * 0.88)) + d * 0.1, d, height * 0.03, greens[2], [cards[2]], rnd, sprays, white)
    if snow:
        blob(k, (0, 0, height * 1.0), (0.1, 0.1, 0.16), white, rnd, 0.1, 1)  # snow on the spire's tip


def palm(k, rnd, height=4.8, trunks=1):
    bark = k.material("Bark", PALM_BARK)
    fronds = [k.material("ThinFrond", FROND), k.material("ThinFrondDark", FROND_DARK)]
    nut = k.material("Coconut", (95, 70, 45))
    for t in range(trunks):
        a = rnd.uniform(0, 2 * math.pi) if trunks == 1 else t * 2 * math.pi / trunks + rnd.uniform(-0.3, 0.3)
        bend = Vector((math.cos(a), math.sin(a), 0)) * height * rnd.uniform(0.15, 0.3)
        h = height * (1 if t == 0 else rnd.uniform(0.65, 0.8))
        pts = [Vector((0, 0, -0.1)) + bend * 0.15 * t]
        n = 12
        for i in range(1, n + 1):  # the trunk curves away and back up
            s = i / n
            pts.append(Vector((0, 0, h * s)) + bend * (s * s) + bend * 0.15 * t)
        for i, (p, q) in enumerate(zip(pts, pts[1:])):  # stepped rings of old leaf bases
            r = 0.19 - 0.06 * i / n
            k.rod(p, q, r * 1.12, bark, None, 8, taper=0.78)
        crown = pts[-1]
        out = (pts[-1] - pts[-2]).normalized()
        count = rnd.randint(7, 9)
        for i in range(count):
            fa = i * 2 * math.pi / count + rnd.uniform(-0.2, 0.2)
            d = Vector((math.cos(fa), math.sin(fa), 0))
            length = h * rnd.uniform(0.32, 0.4)
            p = arc(crown, d, length, rnd.uniform(0.25, 0.6), rnd.uniform(0.9, 1.4), 10)
            strip(k, p, leaf_widths(10, 0.26, 0.05), (-d.y, d.x, 0), fronds[i % 2], fold=0.45)
        for i in range(3):
            ca = i * 2 * math.pi / 3 + rnd.uniform(0, 1)
            k.ellipsoid(crown - out * 0.12 + Vector((math.cos(ca) * 0.12, math.sin(ca) * 0.12, -0.05)), (0.09, 0.09, 0.1), nut, None, 10, 6)


# ---- bushes and plants ----

def bush(k, rnd, flowers=None):
    greens = [k.material("Leaves", LEAF), k.material("LeavesDark", LEAF_DARK), k.material("LeavesLight", LEAF_LIGHT)]
    blob(k, (0, 0, 0.42), (0.62, 0.62, 0.5), greens[1], rnd)
    spots = []
    for i in range(6):
        a = i * 2 * math.pi / 6 + rnd.uniform(-0.3, 0.3)
        c = Vector((math.cos(a) * 0.42, math.sin(a) * 0.42, 0.32 + rnd.uniform(0, 0.15)))
        s = rnd.uniform(0.32, 0.42)
        blob(k, c, (s, s, s * 0.85), greens[i % 3], rnd)
        spots.append((c, s))
    blob(k, (0, 0, 0.78), (0.4, 0.4, 0.33), greens[2], rnd)
    spots.append((Vector((0, 0, 0.78)), 0.38))
    if flowers:
        petal = k.material("Flower", flowers)
        heart = k.material("FlowerHeart", (250, 215, 80))
        for _ in range(22):
            c, s = rnd.choice(spots)
            d = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(0.1, 1))).normalized()
            p = c + d * s * 0.92
            k.ellipsoid(p, (0.1, 0.1, 0.05), petal, None, 8, 4)
            k.ellipsoid(p + d * 0.04, (0.035, 0.035, 0.025), heart, None, 6, 3)


def plant(k, rnd, leaves=7, length=0.5, rise=0.55, droop=0.45, width=0.11, colors=(LEAF, LEAF_DARK)):
    mats = [k.material("ThinLeaf", colors[0]), k.material("ThinLeafDark", colors[1])]
    for i in range(leaves):
        a = i * 2 * math.pi / leaves + rnd.uniform(-0.25, 0.25)
        d = Vector((math.cos(a), math.sin(a), 0))
        ln = length * rnd.uniform(0.8, 1.15)
        p = arc((0, 0, 0.02), d, ln, rise * ln * rnd.uniform(0.8, 1.2) * 2, droop * ln * 2, 8)
        strip(k, p, leaf_widths(8, width * rnd.uniform(0.85, 1.15)), (-d.y, d.x, 0), mats[i % 2], fold=0.3)


def fern(k, rnd):
    mats = [k.material("ThinFrond", (75, 135, 45)), k.material("ThinFrondDark", (55, 110, 40))]
    for i in range(9):
        a = i * 2 * math.pi / 9 + rnd.uniform(-0.2, 0.2)
        d = Vector((math.cos(a), math.sin(a), 0))
        n = 16
        p = arc((0, 0, 0.02), d, rnd.uniform(0.55, 0.7), 0.7, 0.75, n)
        # Serrated: the width zigzags for the little leaflets along the frond.
        w = [0.12 * math.sin(math.pi * min(1, (j / n) * 1.1)) * (0.55 if j % 2 else 1) + 0.005 for j in range(n + 1)]
        strip(k, p, w, (-d.y, d.x, 0), mats[i % 2], fold=0.15)


def flowers(k, rnd, color, heads=6):
    stem = k.material("Stem", (80, 130, 50))
    petal = k.material("Petal", color)
    heart = k.material("Heart", (240, 190, 60) if color != (250, 205, 50) else (150, 90, 40))
    plant(k, rnd, leaves=5, length=0.28, rise=0.3, droop=0.5, width=0.06)
    for i in range(heads):
        a = rnd.uniform(0, 2 * math.pi)
        r = rnd.uniform(0.05, 0.3)
        top = Vector((math.cos(a) * r, math.sin(a) * r, rnd.uniform(0.6, 1.05)))
        k.rod((top.x * 0.3, top.y * 0.3, 0), top, 0.014, stem, None, 5)
        for j in range(6):
            pa = j * math.pi / 3
            k.ellipsoid(top + Vector((math.cos(pa) * 0.07, math.sin(pa) * 0.07, 0)), (0.06, 0.035, 0.02), petal, None, 8, 4,
                        rot=(0, 0, math.degrees(pa)))
        k.ellipsoid(top + Vector((0, 0, 0.015)), (0.04, 0.04, 0.03), heart, None, 8, 4)


def clover(k, rnd):
    stem = k.material("Stem", (80, 130, 50))
    leaf = k.material("Clover", (85, 150, 55))
    bloom = k.material("Bloom", (240, 235, 245))
    for i in range(7):
        a = rnd.uniform(0, 2 * math.pi)
        r = rnd.uniform(0, 0.25)
        top = Vector((math.cos(a) * r, math.sin(a) * r, rnd.uniform(0.12, 0.22)))
        k.rod((top.x, top.y, 0), top, 0.01, stem, None, 4)
        turn = rnd.uniform(0, 2 * math.pi)
        for j in range(3):
            la = turn + j * 2 * math.pi / 3
            k.ellipsoid(top + Vector((math.cos(la) * 0.045, math.sin(la) * 0.045, 0)), (0.05, 0.04, 0.012), leaf, None, 8, 4,
                        rot=(0, 0, math.degrees(la)))
    for _ in range(2):
        a = rnd.uniform(0, 2 * math.pi)
        top = Vector((math.cos(a) * 0.15, math.sin(a) * 0.15, 0.3))
        k.rod((top.x, top.y, 0), top, 0.01, stem, None, 4)
        k.ellipsoid(top, (0.045, 0.045, 0.05), bloom, None, 8, 6)


def grass(k, rnd):
    mats = [k.material("ThinGrass", (150, 190, 70)), k.material("ThinGrassDark", (110, 160, 55))]
    for i in range(11):
        a = rnd.uniform(0, 2 * math.pi)
        d = Vector((math.cos(a), math.sin(a), 0))
        h = rnd.uniform(0.4, 0.75)
        base = d * rnd.uniform(0, 0.08)
        p = [base + d * h * 0.35 * (t / 6) ** 2 + Vector((0, 0, h * t / 6)) for t in range(7)]
        strip(k, p, [0.035 * (1 - t / 6) + 0.003 for t in range(7)], (-d.y, d.x, 0), mats[i % 2], fold=0.2, thickness=0.006)


def dry_shrub(k, rnd):
    wood = k.material("Twig", (150, 95, 55))
    tips = k.material("DryLeaves", (205, 140, 70))

    def grow(p, d, length, r, depth):
        q = p + d * length
        k.rod(p, q, r, wood, None, 5, taper=0.7)
        if depth == 0:
            k.ellipsoid(q, (0.05, 0.05, 0.04), tips, None, 6, 4)
            return
        for _ in range(2):
            nd = (d + Vector((rnd.uniform(-0.7, 0.7), rnd.uniform(-0.7, 0.7), rnd.uniform(0, 0.4)))).normalized()
            grow(q, nd, length * 0.7, r * 0.7, depth - 1)

    for i in range(5):
        a = i * 2 * math.pi / 5 + rnd.uniform(-0.3, 0.3)
        grow(Vector((0, 0, 0)), Vector((math.cos(a) * 0.6, math.sin(a) * 0.6, 1)).normalized(), 0.22, 0.025, 3)


def agave(k, rnd):
    mats = [k.material("Agave", (150, 155, 90)), k.material("AgaveDark", (120, 130, 75))]
    for ring, (count, tilt, ln) in enumerate(((9, 60, 0.42), (7, 35, 0.36), (5, 12, 0.3))):
        for i in range(count):
            a = i * 2 * math.pi / count + ring * 0.4 + rnd.uniform(-0.1, 0.1)
            t = math.radians(tilt + rnd.uniform(-8, 8))
            d = Vector((math.cos(a) * math.sin(t), math.sin(a) * math.sin(t), math.cos(t)))
            k.rod((0, 0, 0.02), d * ln, 0.05, mats[(i + ring) % 2], None, 5, taper=0.05)


# ---- stones and bones ----

def rock(k, rnd, size=(0.9, 0.8, 0.62), color=ROCK, cuts=9):
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=3, radius=1, location=(0, 0, 0))
    o = bpy.context.object
    chisel(o, rnd, cuts)
    for v in o.data.vertices:
        v.co *= 1 + rnd.uniform(-0.03, 0.03)
    o.scale = size
    o.rotation_euler.z = rnd.uniform(0, math.pi)
    bpy.ops.object.transform_apply(location=False, rotation=True, scale=True)
    sink(o, size[2] * 0.25)
    return k.finish(o, k.material("Rock", color), None, smooth=False)


def skull(k, rnd, horns=True):
    bone = k.material("Bone", BONE)
    dark = k.material("Socket", (60, 45, 35))
    k.ellipsoid((0, 0.05, 0.13), (0.14, 0.15, 0.12), bone, None, 14, 8)  # cranium
    k.ellipsoid((0, -0.13, 0.1), (0.075, 0.17, 0.06), bone, None, 12, 6)  # snout
    for s in (-1, 1):
        k.ellipsoid((s * 0.065, -0.06, 0.16), (0.04, 0.03, 0.035), dark, None, 8, 4)
        k.ellipsoid((s * 0.025, -0.28, 0.11), (0.016, 0.012, 0.014), dark, None, 6, 3)
        if horns:
            branch(k, [Vector((s * 0.11, 0.08, 0.18)), Vector((s * 0.25, 0.05, 0.23)), Vector((s * 0.32, -0.02, 0.36))],
                   0.035, 0.008, bone, 6)


def ribs(k, rnd):
    bone = k.material("Bone", BONE)
    k.rod((0, -0.35, 0.05), (0, 0.35, 0.05), 0.03, bone, None, 6)  # spine, lying on the ground
    for i in range(5):
        y = -0.25 + i * 0.12
        h = 0.22 - 0.025 * i
        for s in (-1, 1):
            pts = [Vector((s * (0.02 + 0.2 * math.sin(t / 6 * math.pi * 0.9)), y + 0.02 * t / 6, 0.05 + h * math.sin(t / 6 * math.pi * 0.6)))
                   for t in range(7)]
            branch(k, pts, 0.018, 0.01, bone, 5)


def single_bone(k, rnd):
    bone = k.material("Bone", BONE)
    k.rod((-0.2, 0, 0.035), (0.2, 0, 0.035), 0.025, bone, None, 7)
    for x in (-0.22, 0.22):
        for y in (-0.025, 0.025):
            k.ellipsoid((x, y, 0.04), (0.035, 0.035, 0.035), bone, None, 8, 6)


def pebble(k, rnd):
    rock(k, rnd, size=(0.22, 0.2, 0.08), cuts=5)


# ---- monuments (ancient desert) ----

STONE = (215, 182, 125)
STONE_DARK = (175, 140, 90)
LAPIS = (40, 70, 150)
GOLD = (225, 180, 70)


def nemes(k, center, scale, stone, lapis, gold, bone=None):
    """The striped royal headdress over a head centered at center: a cap set back so the face
    shows, bands of lapis around it, and striped lappets falling beside the face."""
    c = Vector(center)
    cap = c + Vector((0, 0.03, 0.1)) * scale
    rx, ry, rz = 0.17 * scale, 0.17 * scale, 0.11 * scale
    k.ellipsoid(cap, (rx, ry, rz), gold, bone)
    for i in range(3):  # stripes around the cap, following its curve
        z = (-0.06 + i * 0.045) * scale
        f = math.sqrt(max(0.0, 1 - (z / rz) ** 2))
        bpy.ops.mesh.primitive_cylinder_add(vertices=20, radius=1, depth=0.018 * scale, location=cap + Vector((0, 0, z)))
        o = bpy.context.object
        o.scale = (rx * f + 0.004, ry * f + 0.004, 1)
        k.finish(o, lapis, bone)
    for s in (-1, 1):  # lappets beside the face, striped
        for i in range(5):
            k.box(c + Vector((s * 0.15 * scale, 0.01 * scale, (0.0 - i * 0.06) * scale)), (0.07 * scale, 0.07 * scale, 0.058 * scale),
                  lapis if i % 2 else gold, bone)
    k.ellipsoid(c + Vector((0, 0.12, -0.05)) * scale, (0.15 * scale, 0.08 * scale, 0.18 * scale), gold, bone)  # falling down the back


def stone_face(k, center, scale, stone, dark, gold):
    c = Vector(center)
    k.ellipsoid(c, (0.13 * scale, 0.13 * scale, 0.16 * scale), stone, None)
    k.ellipsoid(c + Vector((0, -0.12, -0.02)) * scale, (0.03 * scale, 0.05 * scale, 0.05 * scale), stone, None)  # nose
    for s in (-1, 1):
        k.ellipsoid(c + Vector((s * 0.05, -0.115, 0.03)) * scale, (0.03 * scale, 0.01 * scale, 0.012 * scale), dark, None)  # eyes
        k.box(c + Vector((s * 0.05, -0.12, 0.06)) * scale, (0.05 * scale, 0.01 * scale, 0.01 * scale), dark, None)  # brows
    k.box(c + Vector((0, -0.12, -0.08)) * scale, (0.06 * scale, 0.01 * scale, 0.012 * scale), dark, None)  # mouth
    k.rod(c + Vector((0, -0.09, -0.13)) * scale, c + Vector((0, -0.08, -0.3)) * scale, 0.035 * scale, gold, None, 6, taper=0.8)  # beard


def sphinx(k, rnd):
    """A lying sphinx facing -Y on a plinth: lion body, paws stretched forward, a pharaoh's head.
    Its origin is the front of its footprint; it reaches back 2.6 m."""
    stone, dark = k.material("Stone", STONE), k.material("StoneDark", STONE_DARK)
    lapis, gold = k.material("Lapis", LAPIS), k.material("Gold", GOLD)
    k.box((0, 1.05, 0.15), (1.5, 2.8, 0.3), dark, None)  # plinth
    k.box((0, 1.05, 0.33), (1.4, 2.7, 0.06), stone, None)
    for i in range(4):  # carved panels along the plinth's sides
        for s in (-1, 1):
            k.box((s * 0.752, 0.0 + i * 0.7, 0.15), (0.01, 0.5, 0.18), stone, None)
    # Body: chest, back and haunches, lying low.
    k.ellipsoid((0, 0.55, 0.85), (0.42, 0.45, 0.45), stone, None)  # chest
    k.ellipsoid((0, 1.25, 0.72), (0.4, 0.6, 0.36), stone, None)  # back
    for s in (-1, 1):
        k.ellipsoid((s * 0.3, 1.75, 0.62), (0.2, 0.45, 0.28), stone, None)  # haunches
        k.ellipsoid((s * 0.36, 1.45, 0.43), (0.12, 0.35, 0.08), stone, None)  # hind legs tucked
        k.box((s * 0.22, -0.05, 0.47), (0.2, 1.0, 0.18), stone, None)  # forelegs, stretched forward
        k.ellipsoid((s * 0.22, -0.55, 0.47), (0.12, 0.1, 0.09), stone, None)  # paws
        for t in range(3):
            k.box((s * 0.22 + (t - 1) * 0.06, -0.62, 0.43), (0.012, 0.06, 0.07), dark, None)  # toes
    branch(k, [Vector((0.38, 2.05, 0.42)), Vector((0.5, 1.6, 0.4)), Vector((0.45, 1.25, 0.42))], 0.06, 0.04, stone)  # tail
    k.ellipsoid((0, 0.35, 1.0), (0.3, 0.2, 0.32), stone, None)  # breast under the head
    stone_face(k, (0, 0.3, 1.4), 1.6, stone, dark, gold)
    nemes(k, (0, 0.3, 1.4), 1.6, stone, lapis, gold)
    k.rod((0, 0.12, 1.62), (0, 0.06, 1.7), 0.035, gold, None, 6, taper=0.5)  # uraeus, the cobra on the brow


def pharaoh_statue(k, rnd):
    """A seated pharaoh on a throne, hands on knees, about 2.6 m tall, facing -Y. Its origin is
    the front of its 2×2 footprint."""
    stone, dark = k.material("Stone", STONE), k.material("StoneDark", STONE_DARK)
    lapis, gold = k.material("Lapis", LAPIS), k.material("Gold", GOLD)
    k.box((0, 0.45, 0.15), (1.5, 1.4, 0.3), dark, None)  # plinth
    k.box((0, 0.6, 0.85), (0.9, 0.8, 1.1), stone, None)  # throne
    k.box((0, 0.98, 1.55), (0.9, 0.14, 0.9), stone, None)  # throne back
    for s in (-1, 1):
        k.box((s * 0.47, 0.6, 0.85), (0.04, 0.7, 0.9), dark, None)  # carved throne sides
        # Legs: thighs on the seat, shins straight down to the plinth, feet together.
        k.box((s * 0.15, 0.2, 1.33), (0.2, 0.62, 0.2), stone, None)
        k.box((s * 0.15, -0.08, 0.82), (0.18, 0.2, 0.9), stone, None)
        k.box((s * 0.15, -0.17, 0.35), (0.18, 0.36, 0.1), stone, None)  # feet
        # Arms: upper arms down the sides, forearms along the thighs, hands flat on the knees.
        k.box((s * 0.3, 0.42, 1.7), (0.13, 0.15, 0.42), stone, None)
        k.box((s * 0.25, 0.12, 1.47), (0.13, 0.5, 0.11), stone, None)
        k.box((s * 0.18, -0.12, 1.47), (0.12, 0.14, 0.06), stone, None)
    k.box((0, 0.42, 1.9), (0.5, 0.3, 0.75), stone, None)  # torso
    k.box((0, 0.27, 1.55), (0.42, 0.04, 0.3), gold, None)  # the kilt's front panel
    for i in range(4):
        k.box((0, 0.25, 1.45 + i * 0.07), (0.44, 0.02, 0.02), lapis, None)
    k.rod((0, 0.42, 2.25), (0, 0.42, 2.38), 0.29, gold, None, 16, taper=0.85)  # broad collar
    k.rod((0, 0.42, 2.25), (0, 0.42, 2.29), 0.3, lapis, None, 16)
    stone_face(k, (0, 0.38, 2.52), 1.0, stone, dark, gold)
    nemes(k, (0, 0.38, 2.52), 1.0, stone, lapis, gold)
    k.rod((0, 0.41, 2.68), (0, 0.41, 2.96), 0.1, k.material("Crown", (235, 230, 215)), None, 10, taper=0.4)  # white crown
    for i in range(3):  # hieroglyph rows on the plinth
        for j in range(5):
            k.box((-0.5 + j * 0.25, -0.252, 0.08 + i * 0.08), (0.12, 0.01, 0.04), stone, None)


def snow_cap(k, rnd, center, radii):
    """A drift of snow lying on something: a flattened lumpy ball."""
    blob(k, center, radii, k.material("Snow", SNOW), rnd, 0.1, 2)


def snow_rock(k, rnd, size=(0.9, 0.8, 0.62)):
    """A grey boulder with snow lying on its top."""
    o = rock(k, rnd, size, color=(132, 132, 138))
    bpy.context.view_layer.update()  # sink moved it
    top = max((o.matrix_world @ v.co).z for v in o.data.vertices)
    snow_cap(k, rnd, (0, 0, top - size[2] * 0.08), (size[0] * 0.45, size[1] * 0.4, size[2] * 0.12))


def snow_bush(k, rnd):
    """A dark evergreen shrub with snow on its top."""
    greens = [k.material("Leaves", (40, 82, 62)), k.material("LeavesDark", (28, 64, 50))]
    blob(k, (0, 0, 0.4), (0.6, 0.6, 0.46), greens[1], rnd)
    for i in range(5):
        a = i * 2 * math.pi / 5 + rnd.uniform(-0.3, 0.3)
        blob(k, (math.cos(a) * 0.38, math.sin(a) * 0.38, 0.32), (0.34, 0.34, 0.3), greens[i % 2], rnd)
    snow_cap(k, rnd, (0, 0, 0.74), (0.5, 0.48, 0.18))
    for i in range(3):
        a = i * 2.1 + 0.4
        snow_cap(k, rnd, (math.cos(a) * 0.4, math.sin(a) * 0.4, 0.55), (0.22, 0.2, 0.08))


def dead_twigs(k, rnd):
    """A few bare grey twigs sticking out of the snow."""
    bark = k.material("Bark", (110, 100, 92))
    for i in range(4):
        a = i * 2 * math.pi / 4 + rnd.uniform(-0.4, 0.4)
        d = Vector((math.cos(a) * 0.3, math.sin(a) * 0.3, 1)).normalized()
        base = Vector((rnd.uniform(-0.15, 0.15), rnd.uniform(-0.15, 0.15), 0))
        ln = rnd.uniform(0.35, 0.6)
        tip = base + d * ln
        branch(k, [base, base + d * ln * 0.5 + Vector((rnd.uniform(-0.05, 0.05), 0, 0)), tip], 0.018, 0.004, bark, 5)
        side = Vector((-d.y + 0.3, d.x, 0.6)).normalized()
        branch(k, [base + d * ln * 0.55, base + d * ln * 0.55 + side * ln * 0.35], 0.008, 0.003, bark, 4)
    snow_cap(k, rnd, (0, 0, 0.0), (0.3, 0.28, 0.06))


def ice_shards(k, rnd):
    """A cluster of pale blue ice crystals jutting from the ground at angles."""
    ice, deep = k.material("Ice", ICE, metal=True), k.material("IceDeep", (120, 175, 215), metal=True)
    for i in range(6):
        a = i * 2 * math.pi / 6 + rnd.uniform(-0.3, 0.3)
        tilt = rnd.uniform(0.15, 0.55) * (0 if i == 0 else 1)
        d = Vector((math.cos(a) * tilt, math.sin(a) * tilt, 1)).normalized()
        base = Vector((math.cos(a) * 0.08 * (i > 0), math.sin(a) * 0.08 * (i > 0), -0.05))
        length = rnd.uniform(0.35, 0.6) * (1.4 if i == 0 else 1)
        k.shard(base, d, rnd.uniform(0.06, 0.09), length, ice if i % 2 else deep, None, flat=(math.cos(a), math.sin(a), 0))
    snow_cap(k, rnd, (0, 0, 0.0), (0.26, 0.24, 0.05))


def frost_fern(k, rnd):
    """A low fern grey-green with frost, its tips white."""
    mats = [k.material("ThinFrond", (110, 150, 140)), k.material("ThinFrondDark", (80, 120, 112))]
    for i in range(8):
        a = i * 2 * math.pi / 8 + rnd.uniform(-0.2, 0.2)
        d = Vector((math.cos(a), math.sin(a), 0))
        n = 14
        p = arc((0, 0, 0.02), d, rnd.uniform(0.4, 0.55), 0.45, 0.6, n)
        w = [0.1 * math.sin(math.pi * min(1, (j / n) * 1.1)) * (0.55 if j % 2 else 1) + 0.005 for j in range(n + 1)]
        strip(k, p, w, (-d.y, d.x, 0), mats[i % 2], fold=0.15)
        k.ellipsoid(p[-2], (0.03, 0.03, 0.02), k.material("Frost", SNOW), None, 6, 3)


OLIVE_BARK = (118, 104, 88)
OLIVE = (128, 142, 92)
OLIVE_DARK = (92, 108, 72)
OLIVE_LIGHT = (164, 176, 128)
CYPRESS = (40, 76, 46)
LIMESTONE = (214, 206, 186)
MARBLE = (236, 232, 222)
CLAY = (186, 104, 62)


def olive_tree(k, rnd, height=3.6):
    """A gnarled old olive: a short, twisted trunk splitting low into crooked limbs, under a
    loose, wide crown of silvery grey-green clumps."""
    bark = k.material("Bark", OLIVE_BARK)
    greens = [k.material("Leaves", OLIVE), k.material("LeavesDark", OLIVE_DARK), k.material("LeavesLight", OLIVE_LIGHT)]
    cards = [k.material("ThinLeaf", OLIVE), k.material("ThinLeafDark", OLIVE_DARK), k.material("ThinLeafLight", OLIVE_LIGHT)]
    r = 0.24
    trunk = [Vector((0, 0, -0.1))]
    twist = rnd.uniform(0, 2 * math.pi)
    for i in range(1, 5):  # leaning and twisting on its way up
        t = i / 4
        trunk.append(Vector((math.cos(twist + t * 2.4) * 0.16 * t, math.sin(twist + t * 2.4) * 0.16 * t, height * 0.28 * t)))
    branch(k, trunk, r, r * 0.75, bark, 9)
    for j in range(3):  # ridges wound round the trunk, like the old wood's grain
        ridge = [p + Vector((math.cos(twist + j * 2.1 + i * 0.9), math.sin(twist + j * 2.1 + i * 0.9), 0)) * r * 0.85
                 for i, p in enumerate(trunk)]
        branch(k, ridge, r * 0.32, r * 0.2, bark, 5)
    ends = []
    top = trunk[-1]
    for i in range(rnd.randint(3, 4)):
        a = i * 2 * math.pi / 4 + rnd.uniform(-0.4, 0.4)
        d = Vector((math.cos(a), math.sin(a), rnd.uniform(0.35, 0.7)))
        limbs(k, top, d, height * 0.22, r * 0.55, 2, bark, rnd, ends, spread=0.75, rise=0.15)
    clump = height * 0.15
    for end, d in ends:
        shade = rnd.choice((0, 0, 1, 2))
        c = end + Vector((0, 0, clump * 0.15))
        s = clump * rnd.uniform(0.9, 1.25)
        blob(k, c, (s * 1.25, s * 1.25, s * 0.62), greens[shade], rnd, jitter=0.25, subdiv=1)  # broad, loose clumps
        for _ in range(2):
            o = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(-0.2, 0.4))).normalized() * s * 0.9
            ss = s * rnd.uniform(0.55, 0.75)
            blob(k, c + o, (ss * 1.2, ss * 1.2, ss * 0.6), greens[(shade + 1) % 3], rnd, jitter=0.25, subdiv=1)
        leaf_cards(k, c, s, [cards[shade], cards[(shade + 2) % 3]], rnd, 11, length=0.22, width=0.05)
    # The middle of the crown, so it reads as one wide canopy with gaps, not separate tufts.
    core = sum((e for e, _ in ends), Vector()) / len(ends)
    span = max(Vector((e.x - core.x, e.y - core.y, 0)).length for e, _ in ends)
    blob(k, core + Vector((0, 0, clump * 0.1)), (span * 0.7, span * 0.7, clump * 0.7), greens[1], rnd, jitter=0.25)


def cypress(k, rnd, height=5.6):
    """A tall, narrow Mediterranean cypress: a dark green flame of layered, lumpy foliage on a
    short trunk."""
    bark = k.material("Bark", BARK)
    greens = [k.material("Needles", CYPRESS), k.material("NeedlesDark", (28, 58, 36)), k.material("NeedlesLight", (58, 98, 58))]
    k.rod((0, 0, -0.1), (0, 0, height * 0.3), 0.14, bark, None, 7, taper=0.7)
    layers = 16
    for i in range(layers):
        t = i / (layers - 1)
        z = height * (0.12 + 0.82 * t)
        w = 0.42 * math.sin(math.pi * (0.12 + 0.82 * (1 - t) ** 0.75)) + 0.07  # widest a third of the way up
        for j in range(5):
            a = rnd.uniform(0, 2 * math.pi)
            o = Vector((math.cos(a), math.sin(a), 0)) * w * 0.45
            shade = 2 if (t > 0.8 or math.cos(a - 2.3) > 0.5) else (1 if math.cos(a - 2.3) < -0.4 else 0)  # lit from the upper left
            blob(k, Vector((0, 0, z)) + o, (w * 0.75, w * 0.75, height * 0.06), greens[shade], rnd, jitter=0.25, subdiv=1)
    blob(k, (0, 0, height * 0.5), (0.3, 0.3, height * 0.4), greens[1], rnd, jitter=0.1)  # the dark core
    blob(k, (0, 0, height * 0.97), (0.09, 0.09, 0.22), greens[2], rnd, jitter=0.1, subdiv=1)


def lavender(k, rnd):
    """A round cushion of grey-green lavender with purple flower spikes standing out of it."""
    leaves = [k.material("Leaves", (124, 140, 112)), k.material("LeavesDark", (96, 112, 88))]
    purple, deep = k.material("Flower", (150, 112, 200)), k.material("FlowerDeep", (112, 80, 168))
    blob(k, (0, 0, 0.26), (0.5, 0.5, 0.3), leaves[1], rnd, jitter=0.15)
    blob(k, (0, 0, 0.36), (0.38, 0.38, 0.22), leaves[0], rnd, jitter=0.15)
    stem = k.material("Stem", (110, 128, 96))
    for i in range(26):
        a = rnd.uniform(0, 2 * math.pi)
        r = rnd.uniform(0.05, 0.42)
        base = Vector((math.cos(a) * r, math.sin(a) * r, 0.3 + 0.15 * (1 - r / 0.42)))
        tip = base + Vector((math.cos(a) * r * 0.4, math.sin(a) * r * 0.4, rnd.uniform(0.22, 0.34)))
        k.rod(base, tip, 0.008, stem, None, 4)
        k.ellipsoid(tip, (0.025, 0.025, 0.065), purple if i % 3 else deep, None, 6, 4)


def sage(k, rnd):
    """A low clump of soft grey-green sage leaves."""
    plant(k, rnd, leaves=9, length=0.32, rise=0.4, droop=0.3, width=0.1, colors=((150, 164, 132), (118, 134, 104)))


def poppies(k, rnd):
    """A few red poppies on thin stems over a tuft of leaves."""
    flowers(k, rnd, (214, 48, 40), heads=5)


def broken_column(k, rnd):
    """A fluted marble column broken off at knee height, a fallen drum beside it, on a square
    base."""
    marble, dark = k.material("Marble", MARBLE), k.material("MarbleShade", (200, 196, 186))
    k.box((0, 0, 0.06), (0.62, 0.62, 0.12), dark, None)
    k.rod((0, 0, 0.12), (0, 0, 0.72), 0.22, marble, None, 16, taper=0.97)
    for i in range(12):  # the flutes: shallow grooves down the shaft
        a = i * math.pi / 6
        k.rod((math.cos(a) * 0.215, math.sin(a) * 0.215, 0.14), (math.cos(a) * 0.21, math.sin(a) * 0.21, 0.7), 0.022, dark, None, 4)
    for i in range(5):  # its jagged top
        a = i * 2 * math.pi / 5 + rnd.uniform(-0.3, 0.3)
        k.shard((math.cos(a) * 0.12, math.sin(a) * 0.12, 0.7), (math.cos(a) * 0.3, math.sin(a) * 0.3, 1), 0.1, rnd.uniform(0.06, 0.16), marble,
                None, flat=(math.cos(a), math.sin(a), 0))
    bpy.ops.mesh.primitive_cylinder_add(vertices=16, radius=0.2, depth=0.36, location=(0.46, 0.18, 0.2), rotation=(0, math.pi / 2, 0.5))
    k.finish(bpy.context.object, marble, None)  # the fallen drum


def amphora(k, rnd):
    """Two clay amphorae: one standing, half sunk in the ground, one lying on its side."""
    clay, dark = k.material("Clay", CLAY), k.material("ClayDark", (140, 76, 46))

    def jar(c, up, scale):
        c, up = Vector(c), Vector(up).normalized()
        rings = [(0.0, 0.04), (0.1, 0.16), (0.3, 0.2), (0.48, 0.15), (0.6, 0.07), (0.7, 0.06), (0.74, 0.075)]
        k.loft("jar", [(c + up * z * scale, r * scale, r * scale) for z, r in rings], clay, None, n=14, power=2.0)
        side = up.cross(Vector((0, 0, 1)))
        side = side.normalized() if side.length > 0.1 else Vector((1, 0, 0))
        for sgn in (-1, 1):  # handles
            a = c + up * 0.68 * scale
            b = c + up * 0.5 * scale + side * sgn * 0.15 * scale
            branch(k, [a + side * sgn * 0.05 * scale, a + side * sgn * 0.14 * scale, b], 0.018 * scale, 0.018 * scale, dark, 4)

    jar((0, 0, -0.12), (0, 0, 1), 1.0)
    jar((0.25, -0.22, 0.12), (1, -0.4, 0.15), 0.85)


DEAD_BARK = (74, 68, 72)
GRAVE = (132, 130, 138)


def dead_tree(k, rnd, height=4.4):
    """A dead tree: a twisted, bare grey-black trunk splitting into crooked limbs that end in
    thin, clawing twigs."""
    bark = k.material("Bark", DEAD_BARK)
    r = 0.22
    lean = Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), 0)) * height * 0.06
    trunk = [Vector((0, 0, -0.1))]
    for i in range(1, 5):
        t = i / 4
        trunk.append(Vector((0, 0, height * 0.42 * t)) + lean * t * t + Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), 0)) * 0.06)
    branch(k, trunk, r, r * 0.6, bark, 8)
    for i in range(5):  # roots clawing at the ground
        a = i * 2 * math.pi / 5 + rnd.uniform(-0.3, 0.3)
        d = Vector((math.cos(a), math.sin(a), 0))
        branch(k, [Vector((0, 0, 0.3)), d * 0.3 + Vector((0, 0, 0.08)), d * 0.6 + Vector((0, 0, -0.05))], r * 0.5, r * 0.12, bark, 5)
    ends = []
    top = trunk[-1]
    for i in range(rnd.randint(3, 4)):
        a = i * 2 * math.pi / 4 + rnd.uniform(-0.5, 0.5)
        d = Vector((math.cos(a), math.sin(a), rnd.uniform(0.6, 1.3)))
        limbs(k, top, d, height * 0.24, r * 0.55, 2, bark, rnd, ends, spread=0.9, rise=0.1)
    for end, d in ends:  # twigs: thin, bent spikes
        for _ in range(2):
            t = (d + Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(-0.3, 0.5)))).normalized()
            branch(k, [end, end + t * 0.25 + Vector((0, 0, -0.05)), end + t * 0.42], 0.02, 0.004, bark, 4)


def thorn_bush(k, rnd):
    """A tangle of dark, thorny branches with a few red berries."""
    bark, berry = k.material("Bark", (58, 48, 50)), k.material("Berry", (170, 30, 36))
    leaves = k.material("Leaves", (60, 62, 56))
    blob(k, (0, 0, 0.22), (0.3, 0.3, 0.2), leaves, rnd, jitter=0.3, subdiv=1)
    for i in range(22):
        a = rnd.uniform(0, 2 * math.pi)
        d = Vector((math.cos(a), math.sin(a), rnd.uniform(0.1, 1.1))).normalized()
        base = Vector((0, 0, 0.1))
        tip = base + d * rnd.uniform(0.55, 0.8)
        bend = Vector((rnd.uniform(-0.15, 0.15), rnd.uniform(-0.15, 0.15), 0.1))
        branch(k, [base, base + d * 0.35 + bend, tip], 0.032, 0.006, bark, 5)
        for j in range(3):  # thorns
            p = base.lerp(tip, 0.35 + 0.2 * j)
            k.shard(p, d.cross(Vector((0, 0, 1))) * (1 if j % 2 else -1) + Vector((0, 0, 0.4)), 0.014, 0.07, bark, None)
        if i % 4 == 0:
            k.ellipsoid(tip, (0.035, 0.035, 0.035), berry, None, 6, 4)


def gravestone(k, rnd, cross=False):
    """A weathered gravestone, leaning a little: a rounded slab with a carved cross, or a stone
    cross; moss at its foot."""
    stone, dark = k.material("Stone", GRAVE), k.material("StoneDark", (92, 90, 98))
    moss = k.material("Moss", (70, 84, 60))
    tilt = rnd.uniform(-10, 10)
    if cross:
        k.box((0, 0, 0.45), (0.12, 0.1, 0.9), stone, None, rot=(tilt, 0, 0))
        k.box((0, 0, 0.62), (0.46, 0.1, 0.12), stone, None, rot=(tilt, 0, 0))
        k.ellipsoid((0, 0, 0.62), (0.09, 0.06, 0.09), dark, None, 10, 5, rot=(90, 0, 0))
    else:
        k.box((0, 0, 0.32), (0.5, 0.12, 0.64), stone, None, rot=(tilt, 0, 0))
        bpy.ops.mesh.primitive_cylinder_add(vertices=16, radius=0.25, depth=0.12, location=(0, 0, 0.64), rotation=(math.pi / 2 + math.radians(tilt), 0, 0))
        k.finish(bpy.context.object, stone, None)
        k.box((0, -0.065, 0.48), (0.05, 0.02, 0.3), dark, None, rot=(tilt, 0, 0))  # the carved cross
        k.box((0, -0.065, 0.54), (0.2, 0.02, 0.05), dark, None, rot=(tilt, 0, 0))
    blob(k, (0, 0, 0.0), (0.32, 0.2, 0.06), moss, rnd, jitter=0.2, subdiv=1)


def mushrooms(k, rnd):
    """A cluster of pale-stemmed mushrooms with violet caps, one big and a few small."""
    stem, cap, spot = k.material("Stem", (214, 206, 200)), k.material("Cap", (150, 82, 200)), k.material("Spot", (226, 196, 250))
    for i in range(5):
        a = rnd.uniform(0, 2 * math.pi)
        r = 0 if i == 0 else rnd.uniform(0.1, 0.25)
        s = 1.0 if i == 0 else rnd.uniform(0.4, 0.65)
        base = Vector((math.cos(a) * r, math.sin(a) * r, 0))
        k.rod(base, base + Vector((0, 0, 0.22 * s)), 0.03 * s, stem, None, 8, taper=0.8)
        k.ellipsoid(base + Vector((0, 0, 0.23 * s)), (0.1 * s, 0.1 * s, 0.055 * s), cap, None, 12, 6)
        for j in range(3):
            pa = a + j * 2.1
            k.ellipsoid(base + Vector((math.cos(pa) * 0.05 * s, math.sin(pa) * 0.05 * s, 0.27 * s)), (0.014 * s, 0.014 * s, 0.008), spot, None, 6, 3)


MODELS = {
    **{f"Tree_{i}": (lambda k, r: round_tree(k, r, height=r.uniform(4.2, 5.0))) for i in range(1, 5)},
    **{f"Tree_Old_{i}": (lambda k, r: round_tree(k, r, height=r.uniform(5.6, 6.4), roots=True)) for i in range(1, 3)},
    **{f"Pine_{i}": (lambda k, r: pine(k, r, height=r.uniform(4.2, 5.2))) for i in range(1, 5)},
    **{f"Palm_{i}": (lambda k, r: palm(k, r, height=r.uniform(4.4, 5.4))) for i in range(1, 4)},
    **{f"Palm_Pair_{i}": (lambda k, r: palm(k, r, height=r.uniform(4.0, 4.8), trunks=2)) for i in range(1, 3)},
    "Bush": lambda k, r: bush(k, r),
    "Bush_Flowers": lambda k, r: bush(k, r, flowers=(180, 140, 230)),
    "Plant": lambda k, r: plant(k, r),
    "Plant_Low": lambda k, r: plant(k, r, leaves=8, length=0.4, rise=0.15, droop=0.3, width=0.1),
    "Plant_Big": lambda k, r: plant(k, r, leaves=6, length=0.9, rise=1.1, droop=0.7, width=0.25),
    "Fern": lambda k, r: fern(k, r),
    "Flowers_Yellow": lambda k, r: flowers(k, r, (250, 205, 50)),
    "Flowers_Orange": lambda k, r: flowers(k, r, (240, 130, 40)),
    "Flowers_Red": lambda k, r: flowers(k, r, (215, 50, 50)),
    "Flowers_White": lambda k, r: flowers(k, r, (245, 245, 235)),
    "Clover": lambda k, r: clover(k, r),
    "Grass": lambda k, r: grass(k, r),
    "Shrub_Dry": lambda k, r: dry_shrub(k, r),
    "Agave": lambda k, r: agave(k, r),
    **{f"Rock_{i}": (lambda k, r: rock(k, r)) for i in range(1, 4)},
    "Skull_Horned": lambda k, r: skull(k, r),
    "Skull": lambda k, r: skull(k, r, horns=False),
    "Ribs": lambda k, r: ribs(k, r),
    "Bone": lambda k, r: single_bone(k, r),
    "Pebble": lambda k, r: pebble(k, r),
    "Sphinx": lambda k, r: sphinx(k, r),
    "Pharaoh_Statue": lambda k, r: pharaoh_statue(k, r),
    # The North (added last: a model's seed is its place in this list).
    **{f"Snow_Pine_{i}": (lambda k, r: pine(k, r, height=r.uniform(4.4, 5.4), snow=True)) for i in range(1, 4)},
    **{f"Snow_Pine_Small_{i}": (lambda k, r: pine(k, r, height=r.uniform(2.8, 3.4), tufts=2, snow=True)) for i in range(1, 3)},
    **{f"Snow_Rock_{i}": (lambda k, r: snow_rock(k, r)) for i in range(1, 3)},
    "Snow_Bush": lambda k, r: snow_bush(k, r),
    "Frost_Fern": lambda k, r: frost_fern(k, r),
    "Dead_Twigs": lambda k, r: dead_twigs(k, r),
    "Ice_Shards": lambda k, r: ice_shards(k, r),
    # The Mediterranean (added last too).
    **{f"Olive_Tree_{i}": (lambda k, r: olive_tree(k, r, height=r.uniform(3.2, 4.0))) for i in range(1, 4)},
    **{f"Cypress_{i}": (lambda k, r: cypress(k, r, height=r.uniform(5.0, 6.2))) for i in range(1, 3)},
    **{f"Lime_Rock_{i}": (lambda k, r: rock(k, r, color=LIMESTONE)) for i in range(1, 3)},
    "Lavender": lambda k, r: lavender(k, r),
    "Sage": lambda k, r: sage(k, r),
    "Poppies": lambda k, r: poppies(k, r),
    "Broken_Column": lambda k, r: broken_column(k, r),
    "Amphora": lambda k, r: amphora(k, r),
    # The unholy lands (added last too).
    **{f"Dead_Tree_{i}": (lambda k, r: dead_tree(k, r, height=r.uniform(4.0, 5.0))) for i in range(1, 4)},
    **{f"Dark_Pine_{i}": (lambda k, r: pine(k, r, height=r.uniform(4.4, 5.6), gloomy=True)) for i in range(1, 3)},
    **{f"Dark_Rock_{i}": (lambda k, r: rock(k, r, color=(92, 88, 98))) for i in range(1, 3)},
    "Thorn_Bush": lambda k, r: thorn_bush(k, r),
    "Gravestone": lambda k, r: gravestone(k, r),
    "Gravestone_Cross": lambda k, r: gravestone(k, r, cross=True),
    "Mushrooms": lambda k, r: mushrooms(k, r),
}

only = set(args[2].split(",")) if len(args) > 2 else None
for seed, (name, build) in enumerate(MODELS.items()):
    if only and name not in only:
        continue
    k = Kit()
    build(k, random.Random(seed * 7919 + 13))
    k.static(name)
    k.export(out_dir / f"{name}.gltf")
    if preview:
        scene = bpy.context.scene
        obj = bpy.data.objects[name]
        corners = [obj.matrix_world @ Vector(c) for c in obj.bound_box]
        size = max(max(c[i] for c in corners) - min(c[i] for c in corners) for i in range(3))
        top = max(c.z for c in corners)
        cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
        scene.collection.objects.link(cam)
        d = size * 1.6 + 0.3
        cam.location = (d * 0.6, -d * 0.75, top * 0.5 + d * 0.5)
        cam.rotation_euler = (Vector((0, 0, top * 0.45)) - cam.location).to_track_quat("-Z", "Y").to_euler()
        scene.camera = cam
        scene.render.engine = "BLENDER_WORKBENCH"
        scene.display.shading.color_type = "MATERIAL"
        scene.render.resolution_x, scene.render.resolution_y = 300, 300
        preview.mkdir(parents=True, exist_ok=True)
        scene.render.filepath = str(preview / f"{name}.png")
        bpy.ops.render.render(write_still=True)
