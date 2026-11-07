"""Builds the mummy: a bandaged figure with its arms stretched out, rigged and animated.

    blender -b --python tools/models/mummy.py -- OUT.gltf [PREVIEW_DIR]

Runs inside Blender without a window; faces -Y (glTF +Z), about 1.8 m tall. The bandages are
a generated texture (OUT's folder, Mummy_wrap.png) on the shapes' own UVs, so they move with
the limbs. Animations: Idle (sways), Walk (shambles), Attack (raises both arms and slams them
down), HitReact and Death (topples forward).
"""

import random
import sys
from pathlib import Path

import bpy

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
out = Path(args[0])
k = Kit()


def bandage_texture(path, size=256):
    """Overlapping wraps: slanted bands of slightly different shades with dark edges."""
    rnd = random.Random(7)
    band = 22
    shades = [rnd.uniform(0.82, 1.0) for _ in range(64)]
    base = (226 / 255, 212 / 255, 178 / 255)
    px = []
    for y in range(size):
        for x in range(size):
            v = y + 0.18 * x + 3 * ((x // 37) % 2)  # the wraps overlap a little unevenly
            i = int(v // band)
            edge = v % band
            s = shades[i % 64] * (0.55 if edge < 1.6 else 0.85 if edge < 3 else 1)
            s *= rnd.uniform(0.95, 1.03)
            px += [min(1, c * s) for c in base] + [1]
    img = bpy.data.images.new("Mummy_wrap", size, size)
    img.pixels = px
    img.filepath_raw = str(path)
    img.file_format = "PNG"
    img.save()
    return img


out.parent.mkdir(parents=True, exist_ok=True)
WRAP = k.material("Wrap", (255, 255, 255))
tex = WRAP.node_tree.nodes.new("ShaderNodeTexImage")
tex.image = bandage_texture(out.parent / "Mummy_wrap.png")
WRAP.node_tree.links.new(tex.outputs["Color"], WRAP.node_tree.nodes["Principled BSDF"].inputs["Base Color"])
LOOSE = k.material("Loose", (200, 182, 145))
SOCKET = k.material("Socket", (35, 28, 25))
GLOW = k.material("Glow", (140, 255, 210))

# Body
k.ellipsoid((0, 0, 0.98), (0.19, 0.13, 0.15), WRAP, "hips")
k.ellipsoid((0, 0.01, 1.26), (0.22, 0.14, 0.3), WRAP, "spine")
k.ellipsoid((0, -0.01, 1.62), (0.12, 0.13, 0.15), WRAP, "head")
k.rod((0, 0, 1.45), (0, 0, 1.55), 0.06, WRAP, "head")  # neck
for s in (-1, 1):
    k.ellipsoid((s * 0.045, -0.11, 1.64), (0.035, 0.025, 0.025), SOCKET, "head", 10, 6)
    k.ellipsoid((s * 0.045, -0.125, 1.64), (0.016, 0.012, 0.014), GLOW, "head", 8, 5)
    # Arms reach forward; the loose end of a bandage hangs from each forearm.
    shoulder, elbow, hand = (s * 0.25, 0, 1.45), (s * 0.27, -0.3, 1.42), (s * 0.25, -0.58, 1.4)
    k.ellipsoid(shoulder, (0.07, 0.07, 0.07), WRAP, f"arm_{s}")
    k.rod(shoulder, elbow, 0.055, WRAP, f"arm_{s}", 10)
    k.ellipsoid(elbow, (0.05, 0.05, 0.05), WRAP, f"fore_{s}")
    k.rod(elbow, hand, 0.045, WRAP, f"fore_{s}", 10, taper=0.85)
    k.ellipsoid((s * 0.25, -0.63, 1.39), (0.045, 0.06, 0.03), WRAP, f"fore_{s}")
    k.box((s * 0.25, -0.45, 1.3), (0.035, 0.008, 0.2), LOOSE, f"fore_{s}", rot=(10, 0, s * 8))
    # Legs
    hip, knee, ankle = (s * 0.1, 0, 0.92), (s * 0.11, -0.01, 0.5), (s * 0.11, 0.01, 0.09)
    k.rod(hip, knee, 0.075, WRAP, f"thigh_{s}", 10, taper=0.85)
    k.ellipsoid(knee, (0.06, 0.06, 0.06), WRAP, f"shin_{s}")
    k.rod(knee, ankle, 0.06, WRAP, f"shin_{s}", 10, taper=0.8)
    k.ellipsoid((s * 0.11, -0.04, 0.045), (0.06, 0.12, 0.045), WRAP, f"shin_{s}")
k.box((0.08, -0.12, 0.8), (0.04, 0.008, 0.22), LOOSE, "hips", rot=(-6, 0, 10))  # loose end at the waist

bones = [("root", (0, 0, 0), None), ("hips", (0, 0, 0.95), "root"), ("spine", (0, 0, 1.1), "hips"),
         ("head", (0, 0, 1.5), "spine")]
for s in (-1, 1):
    bones += [(f"arm_{s}", (s * 0.25, 0, 1.45), "spine"), (f"fore_{s}", (s * 0.27, -0.3, 1.42), f"arm_{s}"),
              (f"thigh_{s}", (s * 0.1, 0, 0.92), "hips"), (f"shin_{s}", (s * 0.11, -0.01, 0.5), f"thigh_{s}")]
k.bones(bones)
k.bind("Mummy")
key = k.key
# Bones point up, so rotating about X tilts what's above a pivot forward and swings what
# hangs below it backward: legs swing forward with negative X, arms rise with negative X.


def arms(frame, lift, sway=0.0):
    for s in (-1, 1):
        key(frame, f"arm_{s}", rot=(-lift, 0, s * sway))


def idle():
    for f, a in ((1, 0), (30, 4), (60, 0)):
        key(f, "spine", rot=(0, a, 0))
        arms(f, a * 1.5)
        key(f, "head", rot=(0, -a * 2, a))


def walk():
    # A shamble: stiff legs swinging, the body rocking from side to side, arms out front.
    for f, ph in ((1, 1), (9, 0), (17, -1), (25, 0), (33, 1)):
        for s in (-1, 1):
            key(f, f"thigh_{s}", rot=(-22 * ph * s, 0, 0))
            key(f, f"shin_{s}", rot=(12 * max(0, -ph * s), 0, 0))
        key(f, "hips", rot=(0, 0, 5 * ph), loc=(0, -0.03 * abs(ph), 0))
        key(f, "spine", rot=(4, 0, -4 * ph))
        arms(f, 4 * ph, 0)


def attack():
    arms(8, 55)  # raise both arms
    key(8, "spine", rot=(-10, 0, 0))
    arms(14, -25)  # and slam them down
    key(14, "spine", rot=(18, 0, 0))
    key(14, "root", loc=(0, 0, 0.25))
    key(14, "head", rot=(10, 0, 0))


def hit():
    key(5, "spine", rot=(-14, 0, 6))
    key(5, "head", rot=(-15, 0, 0))
    key(5, "root", loc=(0, 0, -0.1))
    arms(5, 15)


def death():
    key(10, "spine", rot=(-12, 0, 0))
    key(10, "hips", loc=(0, -0.08, 0))
    for s in (-1, 1):
        key(10, f"thigh_{s}", rot=(-10, 0, 0))
        key(10, f"shin_{s}", rot=(25, 0, 0))
    key(26, "root", rot=(88, 0, 0), loc=(0, 0.13, 0))  # topples forward onto the ground
    key(40, "root", rot=(90, 0, 0), loc=(0, 0.12, 0))
    arms(26, 60, 20)
    arms(40, 70, 30)
    key(40, "spine")


k.action("Idle", 61, idle)
k.action("Walk", 33, walk)
k.action("Attack", 24, attack)
k.action("HitReact", 18, hit)
k.action("Death", 40, death)
k.export(out)
if len(args) > 1:
    k.preview(args[1], distance=4.2, target_z=0.9, textured=True)
