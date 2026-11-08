"""Builds the shaman's totems, carved and painted wooden poles in a ring of stones:

- wrath: two grim faces stacked on the pole, a thunderbird with spread wings on top and a red
  wrath crystal above (Magic Wrath Totem);
- healing: a calm face with closed eyes and painted waves, a bowl on top holding a glowing orb
  of water, and three streams of drops falling from it into a small pool at the foot (Healing
  Stream Totem).

    blender -b --python tools/models/totem.py -- wrath|healing OUT.gltf [PREVIEW.png]

Runs inside Blender without a window. Static models about 1.3 to 1.4 m tall, standing on their
origin, the faces toward -Y (glTF +Z). The battle scene raises them out of the ground beside the
shaman at the start of a fight (internal/game/scene.go).
"""

import math
import sys
from pathlib import Path

import bpy
from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
kind, out = args[0], args[1]
k = Kit()

STONE = k.material("Stone", (128, 124, 118))
BONE = k.material("Bone", (228, 218, 192))


def stones():
    for x, y, r in ((0.17, -0.06, 0.09), (-0.15, -0.1, 0.08), (-0.05, 0.17, 0.1), (0.12, 0.14, 0.07), (-0.18, 0.07, 0.07)):
        k.ellipsoid((x, y, r * 0.4), (r, r * 0.9, r * 0.6), STONE, None, 8, 6)


def wrath():
    wood = k.material("Wood", (120, 82, 50))
    dark = k.material("Carving", (52, 34, 24))
    teal = k.material("Teal", (55, 150, 160))
    red = k.material("Red", (180, 45, 35))
    glow = k.material("Glow", (255, 170, 60))
    crystal = k.material("Crystal", (235, 70, 45))
    fur = k.material("Fur", (150, 120, 88))

    k.rod((0, 0, 0), (0, 0, 1.08), 0.1, wood, None, 12, taper=0.9)
    stones()

    # The lower face, painted teal: round eyes and a wide mouth with bared teeth.
    k.rod((0, 0, 0.26), (0, 0, 0.52), 0.122, teal, None, 12)
    for s in (-1, 1):
        k.ellipsoid((s * 0.045, -0.11, 0.44), (0.032, 0.02, 0.03), dark, None, 8, 6)
        k.ellipsoid((s * 0.045, -0.122, 0.44), (0.012, 0.008, 0.012), BONE, None, 6, 4)
    k.box((0, -0.115, 0.33), (0.12, 0.02, 0.05), dark, None)
    for x in (-0.04, -0.013, 0.013, 0.04):
        k.box((x, -0.124, 0.345), (0.018, 0.008, 0.02), BONE, None)

    k.rod((0, 0, 0.55), (0, 0, 0.6), 0.118, fur, None, 10)  # a fur wrap between the faces

    # The upper face, painted red: angry brows over glowing eyes, a snarl.
    k.rod((0, 0, 0.62), (0, 0, 0.9), 0.112, red, None, 12)
    for s in (-1, 1):
        k.box((s * 0.045, -0.105, 0.83), (0.06, 0.02, 0.018), dark, None, rot=(0, s * 22, 0))
        k.ellipsoid((s * 0.042, -0.106, 0.79), (0.022, 0.012, 0.016), glow, None, 8, 5)
    k.box((0, -0.106, 0.69), (0.09, 0.02, 0.035), dark, None)
    for s in (-1, 1):
        k.rod((s * 0.03, -0.112, 0.705), (s * 0.03, -0.112, 0.67), 0.008, BONE, None, 4, taper=0.2)  # fangs

    # The thunderbird on top: a head with a hooked beak, wings spread wide, feathers hanging.
    k.ellipsoid((0, -0.02, 1.0), (0.1, 0.1, 0.09), teal, None, 12, 8)
    k.rod((0, -0.09, 1.0), (0, -0.2, 0.95), 0.03, BONE, None, 6, taper=0.2)  # the beak
    for s in (-1, 1):
        k.ellipsoid((s * 0.04, -0.085, 1.03), (0.015, 0.008, 0.012), glow, None, 6, 4)
        wing = Vector((s * 0.1, 0, 0.98))
        k.box(wing + Vector((s * 0.14, 0, 0.06)), (0.3, 0.035, 0.09), teal, None, rot=(0, -s * 20, 0))
        k.box(wing + Vector((s * 0.29, 0, 0.12)), (0.08, 0.036, 0.1), red, None, rot=(0, -s * 20, 0))  # painted tips
        for i in range(3):
            x = s * (0.15 + 0.07 * i)
            top = Vector((x, 0, 1.0 + 0.04 * i))
            k.rod(top, top + Vector((s * 0.01, 0, -0.18)), 0.008, BONE, None, 4)
            k.ellipsoid(top + Vector((s * 0.012, 0, -0.21)), (0.014, 0.006, 0.04), red, None, 6, 4)

    # The wrath crystal above the bird's head, on a wooden spike.
    k.rod((0, -0.02, 1.06), (0, -0.02, 1.17), 0.022, wood, None, 6, taper=0.6)
    k.ellipsoid((0, -0.02, 1.25), (0.055, 0.055, 0.11), crystal, None, 6, 4)
    k.ellipsoid((0, -0.02, 1.25), (0.03, 0.03, 0.06), glow, None, 6, 4)


def healing():
    wood = k.material("Wood", (176, 140, 98))  # paler wood than the wrath totem
    dark = k.material("Carving", (78, 58, 40))
    blue = k.material("Blue", (50, 110, 190))
    foam = k.material("Foam", (235, 245, 250))
    water = k.material("Water", (120, 200, 240))
    glow = k.material("Glow", (200, 245, 255))
    rope = k.material("Rope", (190, 165, 110))

    k.rod((0, 0, 0), (0, 0, 1.0), 0.1, wood, None, 12, taper=0.9)
    stones()
    k.ellipsoid((0, -0.18, 0.012), (0.2, 0.13, 0.012), water, None, 16, 4)  # a small pool at the foot

    # The face: closed eyes, a calm mouth, on a blue band with white waves below.
    k.rod((0, 0, 0.3), (0, 0, 0.62), 0.118, blue, None, 12)
    for s in (-1, 1):
        k.box((s * 0.045, -0.115, 0.52), (0.05, 0.016, 0.012), dark, None, rot=(0, s * -12, 0))  # closed eyes
        k.box((s * 0.045, -0.118, 0.555), (0.055, 0.014, 0.012), foam, None, rot=(0, s * 8, 0))  # white brows
    k.box((0, -0.115, 0.42), (0.07, 0.016, 0.014), dark, None)  # a calm mouth
    for i in range(6):  # a ring of white wave crests
        a = i * math.tau / 6
        c = Vector((math.sin(a) * 0.12, -math.cos(a) * 0.12, 0.34))
        k.ellipsoid(c, (0.04, 0.02, 0.022), foam, None, 8, 5, rot=(0, 25, math.degrees(a)))

    k.rod((0, 0, 0.65), (0, 0, 0.7), 0.114, rope, None, 10)  # a rope wrap

    # A carved bowl on top holding a glowing orb of water.
    k.rod((0, 0, 0.88), (0, 0, 0.98), 0.16, wood, None, 14, taper=1.35)
    k.rod((0, 0, 0.97), (0, 0, 1.0), 0.218, blue, None, 14)  # the painted rim
    k.ellipsoid((0, 0, 1.08), (0.11, 0.11, 0.1), water, None, 12, 8)
    k.ellipsoid((0, 0, 1.1), (0.06, 0.06, 0.055), glow, None, 10, 6)

    # Three streams spilling over the rim and arcing down into the pool, thinning as they
    # fall, each ending in a splash.
    for a in (-60, 0, 60):
        r = math.radians(a)
        out_x, out_y = math.sin(r), -math.cos(r)

        def at(t):
            reach = 0.21 + 0.08 * math.sin(t * math.pi * 0.8)  # out over the rim, then down
            return Vector((out_x * reach, out_y * reach, 0.99 - 0.9 * t * t))
        n = 8
        for i in range(n):
            t0, t1 = i / n, (i + 1) / n
            k.rod(at(t0), at(t1), 0.02 - 0.01 * t0, water, None, 6, taper=0.85)
        end = at(1)
        k.ellipsoid((end.x, end.y, 0.02), (0.045, 0.045, 0.018), foam, None, 8, 4)  # a splash

    # Blue feathers hanging from the bowl.
    for s in (-1, 1):
        top = Vector((s * 0.15, 0.04, 0.9))
        k.rod(top, top + Vector((0, 0, -0.14)), 0.007, BONE, None, 4)
        k.ellipsoid(top + Vector((0, 0, -0.17)), (0.012, 0.006, 0.04), blue, None, 6, 4)


{"wrath": wrath, "healing": healing}[kind]()
k.static({"wrath": "Totem_Wrath", "healing": "Totem_Healing"}[kind])
k.export(out)
if len(args) > 2 and args[2]:
    scene = bpy.context.scene
    cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
    scene.collection.objects.link(cam)
    cam.location = (1.4, -2.4, 1.2)
    cam.rotation_euler = (Vector((0, 0, 0.65)) - cam.location).to_track_quat("-Z", "Y").to_euler()
    scene.camera = cam
    scene.render.engine = "BLENDER_WORKBENCH"
    scene.display.shading.color_type = "MATERIAL"
    scene.render.resolution_x, scene.render.resolution_y = 360, 480
    scene.render.filepath = args[2]
    bpy.ops.render.render(write_still=True)
