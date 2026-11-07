"""Builds the weapons: a steel sword, a scimitar, three shields (the knight's heater shield,
the paladin's round one and the hoplite's bronze one), a spear, two axes, a dagger, a bow, a
war hammer and a staff for each caster.

    blender -b --python tools/models/weapons.py -- OUT_DIR [PREVIEW_DIR]

Runs inside Blender without a window and writes one glTF per weapon to OUT_DIR (Sword_Steel,
Scimitar, Shield_Heater, Shield_Paladin, Shield_Hoplite, Spear, Axe, Axe_Dane, Dagger, Bow,
Hammer, Staff_Mage, Staff_Warlock, Staff_Priest, Staff_Shaman, Staff_Druid) and the shields'
wood texture (Wood.png). Each weapon is held at its origin and points up its +Z axis (glTF
+Y), so the same grip offset works for all of them. The shields lie in the XZ plane with their
face toward -Y (glTF +Z) and their back at the origin, for strapping to the forearm.
"""

import math
import sys
from pathlib import Path

import bpy
from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit, write_png  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
out_dir = Path(args[0])
preview = Path(args[1]) if len(args) > 1 and args[1] else None

STEEL = (205, 212, 222)
DARK_STEEL = (120, 122, 130)
LEATHER = (110, 68, 40)
WOOD = (125, 85, 50)
GOLD = (225, 180, 70)


METALS = {"Steel", "Fuller", "Fitting", "Iron", "Edge", "Gold", "Crystal", "Buckle"}  # shaded as metal in the game


def mats(k, **colors):
    return {name: k.material(name, rgb, metal=name in METALS) for name, rgb in colors.items()}


def mesh(k, name, verts, faces, mat, smooth=False):
    me = bpy.data.meshes.new(name)
    me.from_pydata(verts, [], faces)
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    return k.finish(o, mat, None, smooth=smooth)


def blade(k, z0, z1, width, thick, point, mat):
    """A double-edged blade up the Z axis: a diamond cross-section (sharp edges along X, a
    ridge along Y) that narrows to a point over its last stretch."""
    verts, faces = [], []
    n = 6
    for i in range(n + 1):
        z = z0 + (z1 - point - z0) * i / n
        w = width * (1 - 0.12 * i / n)
        verts += [(-w, 0, z), (0, -thick, z), (w, 0, z), (0, thick, z)]
    verts.append((0, 0, z1))
    for i in range(n):
        a, b = 4 * i, 4 * (i + 1)
        for j in range(4):
            faces.append((a + j, a + (j + 1) % 4, b + (j + 1) % 4, b + j))
    tip = len(verts) - 1
    last = 4 * n
    for j in range(4):
        faces.append((last + j, last + (j + 1) % 4, tip))
    faces.append((3, 2, 1, 0))
    return mesh(k, "Blade", verts, faces, mat)


def sword(k):
    m = mats(k, Steel=STEEL, Fuller=(150, 158, 170), Fitting=DARK_STEEL, Leather=LEATHER, Wrap=(80, 48, 28))
    blade(k, 0.09, 0.92, 0.042, 0.009, 0.14, m["Steel"])
    k.box((0, 0, 0.42), (0.016, 0.0185, 0.6), m["Fuller"], None)  # the groove down the middle
    # Guard: a bar with ends that bend toward the blade.
    k.box((0, 0, 0.085), (0.2, 0.035, 0.03), m["Fitting"], None)
    for s in (-1, 1):
        k.box((s * 0.115, 0, 0.1), (0.04, 0.03, 0.025), m["Fitting"], None, rot=(0, s * -35, 0))
    k.rod((0, 0, -0.115), (0, 0, 0.07), 0.021, m["Leather"], None, 8)
    for i in range(6):  # leather wraps
        z = -0.1 + i * 0.03
        k.rod((0, 0, z), (0, 0, z + 0.008), 0.024, m["Wrap"], None, 8)
    k.ellipsoid((0, 0, -0.14), (0.034, 0.03, 0.034), m["Fitting"], None, 10, 6)


def scimitar(k):
    """A curved desert sword: a broad blade widening toward the tip and sweeping back, sharp on
    the outside of the curve, a short brass guard with ends bent toward the blade, a wrapped grip."""
    m = mats(k, Steel=STEEL, Fitting=(196, 152, 78), Leather=(120, 40, 34), Wrap=(70, 40, 28))
    n = 10
    spine, edge = [], []
    for i in range(n + 1):
        t = i / n
        z = 0.09 + 0.74 * t
        bend = 0.12 * t * t  # sweeping back toward -X
        w = 0.03 + 0.03 * math.sin(t * math.pi * 0.85)  # widest toward the tip
        spine.append((-bend - 0.012, z))
        edge.append((-bend + w, z - 0.03 * t * t))
    tip = (-0.13, 0.86)
    outline = spine + [tip] + list(reversed(edge))
    verts = [(x, -0.006, z) for x, z in outline] + [(x, 0.006, z) for x, z in outline]
    count = len(outline)
    faces = [tuple(range(count)), tuple(range(2 * count - 1, count - 1, -1))]
    faces += [(j, (j + 1) % count, count + (j + 1) % count, count + j) for j in range(count)]
    mesh(k, "Blade", verts, faces, m["Steel"])
    k.box((0, 0, 0.08), (0.14, 0.03, 0.022), m["Fitting"], None)
    for s_ in (-1, 1):
        k.box((s_ * 0.075, 0, 0.06), (0.018, 0.026, 0.045), m["Fitting"], None, rot=(0, s_ * 20, 0))
    k.rod((0, 0, -0.1), (0, 0, 0.07), 0.019, m["Leather"], None, 8)
    for i in range(5):
        z = -0.085 + i * 0.032
        k.rod((0, 0, z), (0, 0, z + 0.008), 0.022, m["Wrap"], None, 8)
    k.ellipsoid((0, 0, -0.12), (0.026, 0.024, 0.034), m["Fitting"], None, 10, 6)


def spear(k):
    """A hoplite's spear: a long ash shaft, a leaf-shaped bronze head, a bronze butt spike."""
    m = mats(k, Wood=(150, 118, 78), Gold=(200, 150, 74), Leather=(110, 72, 44))
    k.rod((0, 0, -0.7), (0, 0, 1.5), 0.02, m["Wood"], None, 8)
    k.rod((0, 0, -0.1), (0, 0, 0.12), 0.024, m["Leather"], None, 8)
    k.rod((0, 0, 1.45), (0, 0, 1.56), 0.024, m["Gold"], None, 8)  # the socket
    outline = [(0, 1.55), (0.045, 1.62), (0.05, 1.7), (0.03, 1.8), (0, 1.9), (-0.03, 1.8), (-0.05, 1.7), (-0.045, 1.62)]
    n = len(outline)
    verts = [(x, 0.008, z) for x, z in outline] + [(x, -0.008, z) for x, z in outline]
    faces = [tuple(range(n)), tuple(range(2 * n - 1, n - 1, -1))] + [(j, (j + 1) % n, n + (j + 1) % n, n + j) for j in range(n)]
    mesh(k, "Head", verts, faces, m["Gold"])
    k.rod((0, 0, -0.84), (0, 0, -0.68), 0.022, m["Gold"], None, 6, taper=0.2)


def hoplite_shield(k):
    """A hoplite's round bronze shield (aspis), dished, with a raised rim and a red emblem;
    the back is planked wood with an arm band."""
    m = mats(k, Gold=(200, 150, 74), Edge=(150, 104, 48), Field=(178, 36, 36), Leather=(92, 58, 34))
    m["Back"] = k.textured("Planks", wood_png(), 0.36)
    r, n = 0.36, 40
    bend = lambda x, z: 0.05 * (x * x + z * z) / (r * r)

    def disc(radius, y0, y1, mat):
        pts = [(radius * math.cos(2 * math.pi * i / n), radius * math.sin(2 * math.pi * i / n)) for i in range(n)]
        verts = [(x, y0 + bend(x, z), z) for x, z in pts] + [(x, y1 + bend(x, z), z) for x, z in pts]
        faces = [tuple(range(n)), tuple(range(2 * n - 1, n - 1, -1))] + [(j, (j + 1) % n, n + (j + 1) % n, n + j) for j in range(n)]
        mesh(k, "Disc", verts, faces, mat)

    disc(r, -0.02, -0.05, m["Back"])
    disc(r * 0.96, -0.05, -0.062, m["Gold"])
    bpy.ops.mesh.primitive_torus_add(major_radius=r - 0.01, minor_radius=0.024, major_segments=40, minor_segments=8,
                                     location=(0, 0.0, 0), rotation=(math.pi / 2, 0, 0))
    k.finish(bpy.context.object, m["Edge"], None)
    # The emblem: a red disc with a gold wheel of rays, like a sun.
    disc(r * 0.45, -0.06, -0.066, m["Field"])
    for i in range(8):
        a = i * math.pi / 4
        k.box((math.cos(a) * 0.09, -0.068, math.sin(a) * 0.09), (0.12, 0.006, 0.02), m["Gold"], None, rot=(0, -math.degrees(a), 0))
    k.ellipsoid((0, -0.07, 0), (0.035, 0.012, 0.035), m["Gold"], None, 10, 5)
    k.box((0, -0.008, 0.1), (0.36, 0.008, 0.045), m["Leather"], None)


def wood_png():
    """A tileable plank texture written next to the props: four boards across, each its own
    shade, with wavy grain, a knot or two and dark seams between them."""
    path = out_dir / "Wood.png"
    size, boards = 64, 4
    shades = [(118, 80, 46), (132, 92, 54), (110, 74, 42), (126, 88, 50)]
    knots = [(0, 22, 40), (2, 40, 12)]  # (board, x, y)
    rows = []
    for y in range(size):
        row = []
        for x in range(size):
            bw = size // boards
            i, bx = x // bw, x % bw
            # Grain: stripes along the board, bent by waves that repeat over the tile's height
            # and around the knots.
            u = bx + 1.6 * math.sin(2 * math.pi * (y / size * 2 + i * 0.37)) + 0.8 * math.sin(2 * math.pi * y / size * 5 + i)
            for kb, kx, ky in knots:
                if kb == i:
                    dy = (y - ky + size / 2) % size - size / 2
                    d = math.hypot(x - kx, dy * 0.6)
                    u += 5 * math.exp(-d * d / 18) * (1 if x > kx else -1)
            grain = 0.5 + 0.5 * math.sin(u * 1.9 + i * 2.1)
            v = 0.86 + 0.18 * grain ** 3 - 0.1 * (math.sin(u * 0.7 + i) > 0.85)
            for kb, kx, ky in knots:
                dy = (y - ky + size / 2) % size - size / 2
                if kb == i and math.hypot(x - kx, dy * 0.75) < 2.2:
                    v = 0.55
            if bx == 0:
                v = 0.42  # the seam
            elif bx == 1 or bx == bw - 1:
                v *= 0.88
            row.append(tuple(min(255, int(c * v)) for c in shades[i]))
        rows.append(row)
    return write_png(path, rows)


def heater_shield(k):
    """A knight's heater shield: flat-topped, curving to a point, a steel rim around a red field
    with a gold cross. Face toward -Y, back at the origin.
    The back is planked wood inside a steel lip, with leather straps for the arm."""
    m = mats(k, Field=(178, 36, 36), Steel=(138, 156, 182), Edge=(78, 90, 112), Gold=(215, 170, 70),
             Leather=(92, 58, 34), Buckle=(150, 152, 160))
    m["Back"] = k.textured("Planks", wood_png(), 0.36)
    w, top, bottom = 0.27, 0.34, -0.42
    corner = top - 0.02
    upper = [(-w + 2 * w * i / 8, top - 0.02 * ((-w + 2 * w * i / 8) / w) ** 2) for i in range(9)]  # slightly curved top
    right = [(w * math.cos(i / 12 * math.pi / 2) ** 1.3, corner - (corner - bottom) * (i / 12) ** 1.1) for i in range(1, 13)]
    left = [(-x, z) for x, z in reversed(right[:-1])]
    outline = upper + right + left  # along the top, down the right side to the point, up the left
    bend = lambda x: 0.035 * (x / w) ** 2  # curved toward the arm
    mid = -0.04  # the outline's middle height, which insets shrink toward

    def slab(pts, y0, y1, mat, scale=1.0):
        n = len(pts)
        verts = [(x * scale, y0 + bend(x), z * scale) for x, z in pts] + [(x * scale, y1 + bend(x), z * scale) for x, z in pts]
        faces = [tuple(range(n)), tuple(range(2 * n - 1, n - 1, -1))] + [(j, (j + 1) % n, n + (j + 1) % n, n + j) for j in range(n)]
        mesh(k, "Slab", verts, faces, mat)

    def lip(pts, inset, y0, y1, mat):
        """A flat frame between the outline and the outline shrunk by inset (meters)."""
        n = len(pts)
        inner = [(x * (1 - inset / w), mid + (z - mid) * (1 - inset / 0.38)) for x, z in pts]
        verts = []
        for y in (y0, y1):
            verts += [(x, y + bend(x), z) for x, z in pts] + [(x, y + bend(x), z) for x, z in inner]
        faces = []
        for j in range(n):
            a, b = j, (j + 1) % n
            faces += [(a, b, n + b, n + a), (2 * n + a, 2 * n + n + a, 2 * n + n + b, 2 * n + b),  # the two faces
                      (a, 2 * n + a, 2 * n + b, b), (n + a, n + b, 3 * n + b, 3 * n + a)]  # outer and inner walls
        mesh(k, "Lip", verts, faces, mat)

    slab(outline, -0.02, -0.045, m["Back"])
    lip(outline, 0.022, -0.012, -0.03, m["Edge"])
    slab(outline, -0.045, -0.055, m["Edge"])
    slab(outline, -0.05, -0.06, m["Field"], 0.9)
    k.box((0, -0.063, -0.02), (0.06, 0.008, 0.5), m["Gold"], None)  # the cross
    k.box((0, -0.063, 0.1), (0.36, 0.008, 0.06), m["Gold"], None)
    k.ellipsoid((0, -0.066, 0.1), (0.045, 0.012, 0.045), m["Steel"], None, 12, 6)  # boss
    # The back: two straps across for the forearm, each with a buckle, and a rivet at each end.
    for z, length in ((0.12, 0.3), (-0.12, 0.24)):
        k.box((0, -0.014, z), (length, 0.008, 0.045), m["Leather"], None)
        k.box((0.05, -0.009, z), (0.03, 0.006, 0.052), m["Buckle"], None)
        k.box((0.05, -0.006, z), (0.018, 0.004, 0.034), m["Leather"], None)
        for s in (-1, 1):
            k.ellipsoid((s * length / 2, -0.012 + bend(length / 2), z), (0.012, 0.008, 0.012), m["Buckle"], None, 6, 4)


def axe(k):
    m = mats(k, Wood=WOOD, Leather=LEATHER, Iron=(150, 152, 160), Edge=(215, 220, 228))
    k.rod((0, 0, -0.3), (0, 0, 0.52), 0.022, m["Wood"], None, 8, taper=0.85)
    k.rod((0, 0, -0.12), (0, 0, 0.08), 0.026, m["Leather"], None, 8)
    k.ellipsoid((0, 0, -0.3), (0.028, 0.028, 0.02), m["Wood"], None, 8, 4)
    # Head: a socket around the haft, and a bearded blade flaring out to +X, which is down
    # in the sword grip (the blade's edge faces the ground when held forward).
    k.box((0, 0, 0.44), (0.065, 0.05, 0.1), m["Iron"], None)
    k.box((-0.045, 0, 0.44), (0.04, 0.035, 0.05), m["Iron"], None)  # the poll, behind the haft
    outline = [(0.03, 0.48), (0.12, 0.5), (0.2, 0.56), (0.235, 0.47), (0.235, 0.38), (0.2, 0.27),
               (0.15, 0.32), (0.1, 0.38), (0.03, 0.4)]
    n = len(outline)
    verts = [(x, 0.016 * (1 - x / 0.3), z) for x, z in outline] + [(x, -0.016 * (1 - x / 0.3), z) for x, z in outline]
    faces = [tuple(range(n)), tuple(range(2 * n - 1, n - 1, -1))] + [(j, (j + 1) % n, n + (j + 1) % n, n + j) for j in range(n)]
    mesh(k, "AxeBlade", verts, faces, m["Iron"])
    # A bright, honed strip along the cutting edge.
    k.box((0.232, 0, 0.415), (0.012, 0.012, 0.29), m["Edge"], None)


def dane_axe(k):
    """The berserker's axe: a long haft bound with iron bands, a broad blade with a deep beard
    sweeping down, a honed edge."""
    m = mats(k, Wood=(98, 70, 46), Leather=(70, 44, 30), Iron=(118, 122, 130), Edge=(215, 220, 228))
    k.rod((0, 0, -0.32), (0, 0, 0.74), 0.024, m["Wood"], None, 8, taper=0.85)
    k.rod((0, 0, -0.14), (0, 0, 0.08), 0.028, m["Leather"], None, 8)
    for z in (0.2, 0.32, 0.5):
        k.rod((0, 0, z), (0, 0, z + 0.02), 0.029, m["Iron"], None, 8)
    k.box((0, 0, 0.64), (0.07, 0.055, 0.12), m["Iron"], None)
    outline = [(0.03, 0.69), (0.14, 0.72), (0.25, 0.8), (0.29, 0.68), (0.29, 0.52), (0.25, 0.38), (0.19, 0.44),
               (0.12, 0.54), (0.03, 0.59)]
    n = len(outline)
    verts = [(x, 0.017 * (1 - x / 0.35), z) for x, z in outline] + [(x, -0.017 * (1 - x / 0.35), z) for x, z in outline]
    faces = [tuple(range(n)), tuple(range(2 * n - 1, n - 1, -1))] + [(j, (j + 1) % n, n + (j + 1) % n, n + j) for j in range(n)]
    mesh(k, "AxeBlade", verts, faces, m["Iron"])
    k.box((0.287, 0, 0.59), (0.012, 0.012, 0.4), m["Edge"], None)


def dagger(k):
    m = mats(k, Steel=STEEL, Fitting=DARK_STEEL, Leather=LEATHER)
    k.box((0, 0, 0.2), (0.045, 0.01, 0.26), m["Steel"], None)
    k.rod((0, 0, 0.33), (0, 0, 0.4), 0.03, m["Steel"], None, 4, taper=0.05)  # the point
    k.box((0, 0, 0.065), (0.13, 0.025, 0.02), m["Fitting"], None)  # guard
    k.rod((0, 0, -0.07), (0, 0, 0.055), 0.018, m["Leather"], None, 8)
    k.ellipsoid((0, 0, -0.085), (0.026, 0.026, 0.026), m["Fitting"], None, 10, 6)


def bow(k):
    m = mats(k, Wood=WOOD, Leather=LEATHER, String=(230, 225, 210))
    # Limbs curve back away from the archer (+Y) toward the tips, joined by the string.
    pts = []
    for i in range(-8, 9):
        t = i / 8
        pts.append(Vector((0, 0.12 * t * t, 0.62 * t)))
    for a, b in zip(pts, pts[1:]):
        r = 0.02 * (1 - 0.5 * abs((a.z + b.z) / 2) / 0.62)
        k.rod(a, b, r, m["Wood"], None, 6)
    k.rod((0, 0, -0.08), (0, 0, 0.08), 0.026, m["Leather"], None, 8)  # grip
    k.rod(pts[0], pts[-1], 0.004, m["String"], None, 4)


def hammer(k):
    m = mats(k, Wood=WOOD, Leather=LEATHER, Steel=STEEL, Gold=GOLD)
    k.rod((0, 0, -0.18), (0, 0, 0.6), 0.025, m["Wood"], None, 8)
    k.rod((0, 0, -0.12), (0, 0, 0.1), 0.03, m["Leather"], None, 8)
    k.box((0, 0, 0.66), (0.26, 0.13, 0.13), m["Steel"], None)
    for s in (-1, 1):
        k.box((s * 0.135, 0, 0.66), (0.015, 0.15, 0.15), m["Gold"], None)
    k.ellipsoid((0, 0, -0.2), (0.035, 0.035, 0.035), m["Gold"], None, 10, 6)


def shaft(k, mat, top=0.85, bottom=-0.85, r=0.022):
    k.rod((0, 0, bottom), (0, 0, top), r, mat, None, 8, taper=0.85)


def staff_mage(k):
    m = mats(k, Wood=(70, 50, 40), Crystal=(110, 190, 255), Fitting=DARK_STEEL)
    shaft(k, m["Wood"])
    for a in range(3):  # three prongs hold the crystal
        ang = a * 2 * math.pi / 3
        k.rod((0, 0, 0.82), (0.06 * math.cos(ang), 0.06 * math.sin(ang), 1.0), 0.012, m["Fitting"], None, 4)
    k.ellipsoid((0, 0, 0.95), (0.045, 0.045, 0.08), m["Crystal"], None, 8, 6)


def staff_warlock(k):
    m = mats(k, Wood=(40, 32, 38), Orb=(170, 70, 230), Bone=(225, 215, 190), Dark=(30, 25, 25))
    shaft(k, m["Wood"])
    k.ellipsoid((0, 0, 0.93), (0.06, 0.06, 0.06), m["Orb"], None, 12, 8)
    for s in (-1, 1):  # curved horns around the orb
        k.rod((0, 0, 0.84), (s * 0.08, 0, 0.93), 0.016, m["Bone"], None, 6, taper=0.6)
        k.rod((s * 0.08, 0, 0.93), (s * 0.05, 0, 1.06), 0.012, m["Bone"], None, 6, taper=0.2)
    k.ellipsoid((0, 0, 0.8), (0.04, 0.04, 0.035), m["Bone"], None, 10, 6)  # a little skull
    for s in (-1, 1):
        k.ellipsoid((s * 0.015, -0.032, 0.805), (0.01, 0.006, 0.01), m["Dark"], None, 6, 4)


def staff_priest(k):
    m = mats(k, Ivory=(240, 235, 220), Gold=GOLD, Glow=(255, 245, 190))
    shaft(k, m["Ivory"])
    for z in (-0.3, 0.3, 0.8):
        k.rod((0, 0, z - 0.02), (0, 0, z + 0.02), 0.028, m["Gold"], None, 8)
    # A golden ring around a light, facing the sides.
    for i in range(16):
        a, b = i * 2 * math.pi / 16, (i + 1) * 2 * math.pi / 16
        k.rod((0.1 * math.cos(a), 0, 0.96 + 0.1 * math.sin(a)), (0.1 * math.cos(b), 0, 0.96 + 0.1 * math.sin(b)),
              0.014, m["Gold"], None, 4)
    k.ellipsoid((0, 0, 0.96), (0.05, 0.02, 0.05), m["Glow"], None, 10, 6)


def staff_shaman(k):
    m = mats(k, Wood=WOOD, Fur=(160, 130, 95), Red=(180, 50, 40), Bone=(225, 215, 190), Teal=(60, 150, 160))
    shaft(k, m["Wood"])
    k.rod((0, 0, 0.62), (0, 0, 0.74), 0.034, m["Fur"], None, 8)  # fur wrap
    k.box((0, 0, 0.9), (0.08, 0.06, 0.18), m["Teal"], None)  # a carved totem head
    k.box((0, -0.032, 0.93), (0.05, 0.01, 0.025), m["Red"], None)
    for s in (-1, 1):
        k.box((s * 0.06, 0, 0.98), (0.06, 0.03, 0.02), m["Red"], None, rot=(0, s * -20, 0))  # wings
        k.rod((s * 0.03, 0, 0.66), (s * 0.06, -0.01, 0.48), 0.01, m["Bone"], None, 4, taper=0.4)  # feathers
        k.ellipsoid((s * 0.06, -0.01, 0.45), (0.015, 0.006, 0.04), m["Red"], None, 6, 4)


def staff_druid(k):
    m = mats(k, Wood=(105, 75, 45), Leaf=(90, 160, 70), Moss=(120, 150, 60))
    # A crooked shaft, curling over at the top around a cluster of leaves.
    pts = [Vector((0.015 * math.sin(z * 7), 0.012 * math.cos(z * 5), z)) for z in [i * 0.1 - 0.85 for i in range(17)]]
    pts += [Vector((0.06, 0, 0.88)), Vector((0.1, 0, 0.98)), Vector((0.06, 0, 1.06)), Vector((-0.01, 0, 1.04))]
    for a, b in zip(pts, pts[1:]):
        k.rod(a, b, 0.024, m["Wood"], None, 7)
    for i, (x, z) in enumerate(((0.02, 0.98), (-0.04, 0.95), (0.08, 0.9), (0.0, 1.08), (-0.05, 1.03))):
        k.ellipsoid((x, (-1) ** i * 0.02, z), (0.035, 0.012, 0.055), m["Leaf"], None, 8, 5, rot=(0, i * 35, 0))
    k.ellipsoid((0, 0, 0.3), (0.03, 0.03, 0.05), m["Moss"], None, 8, 5)


def paladin_shield(k):
    """A round shield: a white field with a gold sun, a steel rim, a gold boss; the back is
    planked wood with arm straps."""
    m = mats(k, Field=(236, 232, 220), Steel=(198, 204, 214), Edge=(150, 158, 172), Gold=(225, 180, 70),
             Leather=(92, 58, 34), Buckle=(150, 152, 160))
    m["Back"] = k.textured("Planks", wood_png(), 0.36)
    r, n = 0.3, 40
    bend = lambda x, z: 0.03 * (x * x + z * z) / (r * r)  # dished toward the arm

    def disc(radius, y0, y1, mat):
        pts = [(radius * math.cos(2 * math.pi * i / n), radius * math.sin(2 * math.pi * i / n)) for i in range(n)]
        verts = [(x, y0 + bend(x, z), z) for x, z in pts] + [(x, y1 + bend(x, z), z) for x, z in pts]
        faces = [tuple(range(n)), tuple(range(2 * n - 1, n - 1, -1))] + [(j, (j + 1) % n, n + (j + 1) % n, n + j) for j in range(n)]
        mesh(k, "Disc", verts, faces, mat)

    disc(r, -0.02, -0.045, m["Back"])
    disc(r * 0.93, -0.045, -0.055, m["Field"])
    bpy.ops.mesh.primitive_torus_add(major_radius=r - 0.006, minor_radius=0.02, major_segments=40, minor_segments=8,
                                     location=(0, -0.038 + 0.03, 0), rotation=(math.pi / 2, 0, 0))
    k.finish(bpy.context.object, m["Steel"], None)
    bpy.ops.mesh.primitive_torus_add(major_radius=r * 0.62, minor_radius=0.008, major_segments=40, minor_segments=6,
                                     location=(0, -0.056 + 0.012, 0), rotation=(math.pi / 2, 0, 0))
    k.finish(bpy.context.object, m["Gold"], None)  # an inner gold ring
    # The sun: rays around a boss, alternating long and short.
    for i in range(12):
        a = i * math.pi / 6
        length = 0.15 if i % 2 == 0 else 0.11
        d = Vector((math.cos(a), 0, math.sin(a)))
        verts = [tuple(d * 0.07 + Vector((-d.z, 0, d.x)) * 0.022 + Vector((0, -0.058, 0))),
                 tuple(d * 0.07 - Vector((-d.z, 0, d.x)) * 0.022 + Vector((0, -0.058, 0))),
                 tuple(d * length + Vector((0, -0.057 + bend(d.x * length, d.z * length), 0)))]
        verts += [(x, y + 0.006, z) for x, y, z in verts]
        mesh(k, "Ray", verts, [(0, 1, 2), (5, 4, 3), (0, 3, 4, 1), (1, 4, 5, 2), (2, 5, 3, 0)], m["Gold"])
    k.ellipsoid((0, -0.06, 0), (0.075, 0.04, 0.075), m["Gold"], None, 16, 8)  # boss
    k.ellipsoid((0, -0.095, 0), (0.025, 0.012, 0.025), m["Steel"], None, 10, 5)
    for i in range(8):  # rivets on the rim
        a = i * math.pi / 4 + math.pi / 8
        k.ellipsoid((math.cos(a) * (r - 0.006), -0.03 + 0.03, math.sin(a) * (r - 0.006)), (0.012, 0.03, 0.012), m["Edge"], None, 6, 4)
    for z, length in ((0.1, 0.32), (-0.1, 0.32)):  # arm straps on the back
        k.box((0, -0.014 + 0.004, z), (length, 0.008, 0.045), m["Leather"], None)
        k.box((0.05, -0.006, z), (0.03, 0.006, 0.052), m["Buckle"], None)


WEAPONS = {
    "Sword_Steel": sword,
    "Scimitar": scimitar,
    "Shield_Heater": heater_shield,
    "Shield_Paladin": paladin_shield,
    "Shield_Hoplite": hoplite_shield,
    "Spear": spear,
    "Axe": axe,
    "Axe_Dane": dane_axe,
    "Dagger": dagger,
    "Bow": bow,
    "Hammer": hammer,
    "Staff_Mage": staff_mage,
    "Staff_Warlock": staff_warlock,
    "Staff_Priest": staff_priest,
    "Staff_Shaman": staff_shaman,
    "Staff_Druid": staff_druid,
}

for name, build in WEAPONS.items():
    k = Kit()
    build(k)
    k.static(name)
    k.export(out_dir / f"{name}.gltf")
    if preview:
        scene = bpy.context.scene
        cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
        scene.collection.objects.link(cam)
        cam.location = (1.6, -2.0, 0.6)
        cam.rotation_euler = (Vector((0, 0, 0.2)) - cam.location).to_track_quat("-Z", "Y").to_euler()
        scene.camera = cam
        scene.render.engine = "BLENDER_WORKBENCH"
        scene.display.shading.color_type = "MATERIAL"
        scene.render.resolution_x, scene.render.resolution_y = 240, 400
        preview.mkdir(parents=True, exist_ok=True)
        scene.render.filepath = str(preview / f"{name}.png")
        bpy.ops.render.render(write_still=True)
