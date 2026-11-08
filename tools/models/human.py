"""Builds the human body, heads, hair and outfit pieces on the animation library's skeleton.

    blender -b --python tools/models/human.py -- SKELETON.glb OUT_DIR [PREVIEW_DIR] [NAME,NAME...]

SKELETON.glb is the animation file (Universal Animation Library, UAL1_Standard.glb): only its
armature is used, so every piece has the same 65 bones in the same order and the library's
animations play on it. Runs inside Blender without a window and writes one glTF per piece to
OUT_DIR. The skeleton stands in a T-pose facing -Y (glTF +Z), about 1.8 m tall.

Shapes are lofted around the bones (read from the armature, so they follow its proportions).
Skin weights come from each vertex's distance to the bones its part may follow, so a piece of
clothing bends exactly like the body under it. Pieces (PIECES at the end lists them all, with
their colors): the male body (skin, shorts and the face), hair and beards, and outfit pieces
per equipment slot: the heroes' sets (the knight's and paladin's plate, the mage's and
warlock's robes, cloaks and hoods, the rogue's leathers, the shaman's hide and fur, each made
after art/concept) and the enemies' clothes (bandit, nomad). Each outfit color is its own
piece, since a tint would also color the straps and buckles. Textures are written next to the
pieces: Chainmail.png and the embroidered trims (Trim_*.png).
"""

import json
import math
import struct
import sys
from pathlib import Path

import bmesh
import bpy
from mathutils import Vector

sys.path.insert(0, str(Path(__file__).resolve().parent))
from rigkit import Kit, box_uv, textured_material, write_png  # noqa: E402

args = sys.argv[sys.argv.index("--") + 1:]
skeleton, out_dir = args[0], Path(args[1])
preview = Path(args[2]) if len(args) > 2 and args[2] else None
only = set(args[3].split(",")) if len(args) > 3 else None

SKIN = (225, 175, 140)
SKIN_DARK = (195, 140, 110)
LEATHER = (115, 75, 45)
LEATHER_DARK = (75, 48, 30)
LINEN = (215, 200, 165)
STEEL = (200, 205, 212)
HAIR = (90, 60, 40)


def skeleton_joints(path):
    """The joint names of the skeleton file's skin, in order."""
    data = Path(path).read_bytes()
    n = struct.unpack("<I", data[12:16])[0]
    doc = json.loads(data[20:20 + n])
    return [doc["nodes"][j]["name"] for j in doc["skins"][0]["joints"]]


def fix_joint_order(path, order):
    """Blender's exporter lists the joints in its own order, but raylib applies an animation's
    bones by index, so they must be in the skeleton file's order. Reorders the skin's joints
    and rewrites the inverse bind matrices and every vertex's joint indices to match."""
    doc = json.loads(path.read_text())
    blob = bytearray((path.parent / doc["buffers"][0]["uri"]).read_bytes())
    skin = doc["skins"][0]
    names = [doc["nodes"][j]["name"] for j in skin["joints"]]
    if names == order:
        return
    old = {name: i for i, name in enumerate(names)}
    remap = {old[name]: i for i, name in enumerate(order)}

    def view(acc):
        a = doc["accessors"][acc]
        v = doc["bufferViews"][a["bufferView"]]
        return a, v.get("byteOffset", 0) + a.get("byteOffset", 0), v.get("byteStride")

    a, off, _ = view(skin["inverseBindMatrices"])
    mats = [blob[off + 64 * i:off + 64 * (i + 1)] for i in range(a["count"])]
    for i, name in enumerate(order):
        blob[off + 64 * i:off + 64 * (i + 1)] = mats[old[name]]
    skin["joints"] = [skin["joints"][old[name]] for name in order]
    for m in doc["meshes"]:
        for prim in m["primitives"]:
            a, off, stride = view(prim["attributes"]["JOINTS_0"])
            fmt = {5121: "B", 5123: "H"}[a["componentType"]]
            size = struct.calcsize(fmt) * 4
            for i in range(a["count"]):
                at = off + i * (stride or size)
                js = struct.unpack_from("<4" + fmt, blob, at)
                struct.pack_into("<4" + fmt, blob, at, *[remap[j] for j in js])
    (path.parent / doc["buffers"][0]["uri"]).write_bytes(bytes(blob))
    path.write_text(json.dumps(doc, indent=1))


JOINTS = skeleton_joints(skeleton)


class Body:
    """The skeleton and the shape helpers. Each piece gets a fresh one."""

    def __init__(self):
        self.k = Kit()
        bpy.ops.import_scene.gltf(filepath=skeleton)
        for o in list(bpy.data.objects):
            if o.type != "ARMATURE":
                bpy.data.objects.remove(o)
        self.arm = next(o for o in bpy.data.objects if o.type == "ARMATURE")
        self.arm.animation_data_clear()
        for a in list(bpy.data.actions):
            bpy.data.actions.remove(a)
        for p in self.arm.pose.bones:
            p.matrix_basis.identity()
        self.mats = {}
        self.parts = []  # (object, bones it may follow, weight power)
        self.power = 6.0  # how sharply parts added now switch between bones (see weigh); lower is softer

    def head(self, bone):
        return self.arm.matrix_world @ self.arm.data.bones[bone].head_local

    def tail(self, bone):
        return self.arm.matrix_world @ self.arm.data.bones[bone].tail_local

    def mat(self, name, rgb, metal=False):
        if name not in self.mats:
            self.mats[name] = self.k.material(name, rgb)
            if metal:
                self.mats[name]["metal"] = True
        return self.mats[name]

    def textured(self, name, png, scale):
        """A material showing an image (tiled), for parts that get box-mapped UVs in add();
        scale is the texture's size in meters."""
        if name not in self.mats:
            self.mats[name] = textured_material(name, png, scale)
        return self.mats[name]

    def add(self, obj, mat, bones, smooth=True):
        if "tile" in mat:  # the texture tiles at its size whatever the shape
            box_uv(obj, mat["tile"])
        obj.data.materials.append(mat)
        for p in obj.data.polygons:
            p.use_smooth = smooth
        # Every part gets a vertex color, white; metal's alpha is 0.5, which the game's toon
        # shader shades as steel (a highlight, harder bands). Parts must all have it, or
        # joining fills it in black.
        col = obj.data.color_attributes.new("Color", "FLOAT_COLOR", "POINT")
        alpha = 0.5 if mat.get("metal") else 1.0
        for d in col.data:
            d.color = (1.0, 1.0, 1.0, alpha)
        self.parts.append((obj, bones, self.power))
        return obj

    def mesh(self, verts, faces, mat, bones, smooth=True):
        me = bpy.data.meshes.new("part")
        me.from_pydata([tuple(v) for v in verts], [], faces)
        me.validate()
        # Faces outward, whichever way a loft or sheet was built (a loft running toward -Y
        # comes out inside out, which the game's backface culling would hide).
        bm = bmesh.new()
        bm.from_mesh(me)
        bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
        bm.to_mesh(me)
        bm.free()
        o = bpy.data.objects.new("part", me)
        bpy.context.scene.collection.objects.link(o)
        return self.add(o, mat, bones, smooth)

    def loft(self, rings, mat, bones, n=16, power=2.0, side=(1, 0, 0), caps=True):
        """A tube through cross-sections [(center, half width, half depth)]; power > 2 makes
        the sections boxier. side is the direction the width lies along."""
        side = Vector(side)
        pts = [Vector(c) for c, _, _ in rings]
        verts, faces = [], []
        for i, (_, rx, rz) in enumerate(rings):
            t = (pts[min(i + 1, len(pts) - 1)] - pts[max(i - 1, 0)]).normalized()
            up = side.cross(t).normalized()
            sd = t.cross(up).normalized()
            for j in range(n):
                a = 2 * math.pi * j / n
                ca, sa = math.cos(a), math.sin(a)
                x = math.copysign(abs(ca) ** (2 / power), ca) * rx
                z = math.copysign(abs(sa) ** (2 / power), sa) * rz
                verts.append(pts[i] + sd * x + up * z)
        m = len(rings)
        for i in range(m - 1):
            for j in range(n):
                a, b = i * n + j, i * n + (j + 1) % n
                faces.append((a, b, b + n, a + n))
        if caps:
            verts += [pts[0], pts[-1]]
            s, e = len(verts) - 2, len(verts) - 1
            for j in range(n):
                faces.append((s, (j + 1) % n, j))
                faces.append((e, (m - 1) * n + j, (m - 1) * n + (j + 1) % n))
        return self.mesh(verts, faces, mat, bones)

    def ellipsoid(self, center, radii, mat, bones, segments=14, rings=8, rot=(0, 0, 0)):
        bpy.ops.mesh.primitive_uv_sphere_add(segments=segments, ring_count=rings, radius=1, location=center,
                                             rotation=[math.radians(a) for a in rot])
        o = bpy.context.object
        o.scale = radii
        bpy.ops.object.transform_apply(location=False, rotation=True, scale=True)
        return self.add(o, mat, bones)

    def box(self, center, size, mat, bones, rot=(0, 0, 0)):
        bpy.ops.mesh.primitive_cube_add(size=1, location=center, rotation=[math.radians(a) for a in rot])
        o = bpy.context.object
        o.scale = size
        bpy.ops.object.transform_apply(location=False, rotation=True, scale=True)
        return self.add(o, mat, bones, smooth=False)

    def rod(self, a, b, r, mat, bones, verts=8, taper=1.0):
        a, b = Vector(a), Vector(b)
        d = b - a
        bpy.ops.mesh.primitive_cone_add(vertices=verts, radius1=r, radius2=r * taper, depth=d.length, location=(a + b) / 2)
        o = bpy.context.object
        o.rotation_mode = "QUATERNION"
        o.rotation_quaternion = Vector((0, 0, 1)).rotation_difference(d.normalized())
        bpy.ops.object.transform_apply(location=False, rotation=True, scale=True)
        return self.add(o, mat, bones, smooth=False)

    def tube(self, rows, mat, bones, smooth=True):
        """An open tube through rings of points (rows top to bottom, each ring the same count),
        e.g. a skirt with folds."""
        n = len(rows[0])
        verts = [p for row in rows for p in row]
        faces = [(i * n + j, (i + 1) * n + j, (i + 1) * n + (j + 1) % n, i * n + (j + 1) % n) for i in range(len(rows) - 1) for j in range(n)]
        return self.mesh(verts, faces, mat, bones, smooth)

    def sheet(self, grid, mat, inner, bones, thickness=0.012):
        """A thick sheet (a cloak, a tabard) through a grid of points, rows top to bottom: the
        outside in mat, the inside in inner, closed along its edges. The inside is offset
        toward each point's normal's back side."""
        rows, cols = len(grid), len(grid[0])
        g = [[Vector(p) for p in row] for row in grid]
        normals = []
        for i in range(rows):
            line = []
            for j in range(cols):
                du = g[i][min(j + 1, cols - 1)] - g[i][max(j - 1, 0)]
                dv = g[min(i + 1, rows - 1)][j] - g[max(i - 1, 0)][j]
                line.append(du.cross(dv).normalized())
            normals.append(line)
        outer = [p for row in g for p in row]
        back = [g[i][j] - normals[i][j] * thickness for i in range(rows) for j in range(cols)]
        idx = lambda i, j, layer=0: layer * rows * cols + i * cols + j
        faces_out, faces_in = [], []
        for i in range(rows - 1):
            for j in range(cols - 1):
                faces_out.append((idx(i, j), idx(i + 1, j), idx(i + 1, j + 1), idx(i, j + 1)))
                faces_in.append((idx(i, j, 1), idx(i, j + 1, 1), idx(i + 1, j + 1, 1), idx(i + 1, j, 1)))
        rim = [(i, 0) for i in range(rows)] + [(rows - 1, j) for j in range(cols)] + \
              [(i, cols - 1) for i in reversed(range(rows))] + [(0, j) for j in reversed(range(cols))]
        for (a, b), (c, d) in zip(rim, rim[1:]):
            if (a, b) != (c, d):
                faces_out.append((idx(a, b), idx(a, b, 1), idx(c, d, 1), idx(c, d)))
        verts = outer + back
        self.mesh(verts, faces_out, mat, bones)
        self.mesh(verts, faces_in, inner, bones)

    # ---- skinning and output ----

    def weigh(self, obj, bones, power=6.0):
        """Each vertex follows its nearest bones, weighted by inverse distance to each bone's
        segment, keeping the four strongest (what glTF and raylib allow). power may also be a
        function of the vertex's position, for parts that are soft in one place and firm in
        another (a robe's skirt)."""
        segs = [(b, self.head(b), self.tail(b)) for b in bones]
        groups = {b: obj.vertex_groups.get(b) or obj.vertex_groups.new(name=b) for b in bones}
        mw = obj.matrix_world
        for v in obj.data.vertices:
            p = mw @ v.co
            pw = power(p) if callable(power) else power
            ws = []
            for b, h, t in segs:
                d = t - h
                f = max(0.0, min(1.0, (p - h).dot(d) / max(d.length_squared, 1e-9)))
                dist = max((p - (h + d * f)).length, 0.004)
                ws.append((dist ** -pw, b))
            ws.sort(reverse=True)
            ws = ws[:4]
            total = sum(w for w, _ in ws)
            for w, b in ws:
                if w / total > 0.01:
                    groups[b].add([v.index], w / total, "REPLACE")

    def export(self, name):
        for obj, bones, power in self.parts:
            bpy.context.view_layer.update()
            self.weigh(obj, bones, power)
        bpy.ops.object.select_all(action="DESELECT")
        for obj, _, _ in self.parts:
            obj.select_set(True)
        bpy.context.view_layer.objects.active = self.parts[0][0]
        bpy.ops.object.join()
        mesh = bpy.context.object
        mesh.name = name
        mesh.parent = self.arm
        mesh.modifiers.new("Armature", "ARMATURE").object = self.arm
        out = out_dir / f"{name}.gltf"
        out.parent.mkdir(parents=True, exist_ok=True)
        bpy.ops.object.select_all(action="SELECT")
        bpy.ops.export_scene.gltf(filepath=str(out), export_format="GLTF_SEPARATE", export_animations=False, export_skins=True,
                                  export_yup=True, export_def_bones=False, export_vertex_color="ACTIVE")
        fix_joint_order(out, JOINTS)
        print("wrote", out)
        if preview:
            self.render(name)

    def render(self, name):
        scene = bpy.context.scene
        cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
        scene.collection.objects.link(cam)
        cam.location = (1.6, -3.4, 1.5)
        cam.rotation_euler = (Vector((0, 0, 1.05)) - cam.location).to_track_quat("-Z", "Y").to_euler()
        scene.camera = cam
        scene.render.engine = "BLENDER_WORKBENCH"
        scene.display.shading.color_type = "MATERIAL"
        scene.render.resolution_x, scene.render.resolution_y = 400, 400
        preview.mkdir(parents=True, exist_ok=True)
        scene.render.filepath = str(preview / f"{name}.png")
        bpy.ops.render.render(write_still=True)


# Which bones each region may follow.
TORSO = ["pelvis", "spine_01", "spine_02", "spine_03", "neck_01", "clavicle_l", "clavicle_r", "thigh_l", "thigh_r"]  # not the
# upper arms: the torso's sides would stretch down with them; the shoulder balls cover the joint
NECK = ["spine_03", "neck_01", "Head"]
SKIRT = TORSO + ["calf_l", "calf_r"]  # long robes
HEAD = ["Head", "neck_01"]


def arm_bones(s):
    return [f"clavicle_{s}", f"upperarm_{s}", f"lowerarm_{s}", f"hand_{s}", "spine_03"]


def leg_bones(s):
    return [f"thigh_{s}", f"calf_{s}", f"foot_{s}", f"ball_{s}", "pelvis"]


def sides(b):
    """(side letter, sign of X) for both sides."""
    return (("l", 1), ("r", -1))


def torso_rings(b, grow=0.0, top=1.5, bottom=0.86):
    """The torso's cross-sections from the hips up to the base of the neck, grown outward for
    clothing. (height, half width, half depth, forward offset)"""
    table = [(0.86, 0.15, 0.1, 0.0), (0.95, 0.17, 0.115, 0.0), (1.05, 0.155, 0.1, -0.005), (1.15, 0.16, 0.105, -0.01),
             (1.27, 0.18, 0.115, -0.02), (1.37, 0.195, 0.12, -0.02), (1.44, 0.19, 0.1, -0.005), (1.49, 0.1, 0.075, 0.0)]
    return [((0, 0.01 + f, z), rx + grow, rz + grow) for z, rx, rz, f in table if bottom - 1e-6 <= z <= top + 1e-6]


def limb(b, a_bone, b_bone, radii, grow=0.0, a_at=0.0, b_at=1.0, side=(0, 0, 1)):
    """Rings along a limb from a point on one bone to a point on another, radii as (half width,
    half depth) pairs spread evenly along it."""
    a = b.head(a_bone).lerp(b.tail(a_bone), a_at)
    c = b.head(b_bone).lerp(b.tail(b_bone), b_at)
    n = len(radii)
    return [(a.lerp(c, i / (n - 1)), rx + grow, rz + grow) for i, (rx, rz) in enumerate(radii)]


UPPER_ARM = [(0.062, 0.062), (0.058, 0.056), (0.052, 0.05), (0.047, 0.045)]
FOREARM = [(0.046, 0.044), (0.047, 0.043), (0.04, 0.036), (0.032, 0.028)]
THIGH = [(0.092, 0.095), (0.082, 0.085), (0.07, 0.072), (0.058, 0.06)]
CALF = [(0.057, 0.058), (0.056, 0.066), (0.046, 0.05), (0.036, 0.04), (0.034, 0.036)]


def body_shapes(b, skin, cloth, grow=0.0, legs=True):
    """The bare body: torso, neck, arms, hands, and unless legs is False, legs and feet."""
    b.loft(torso_rings(b, grow), skin, TORSO, n=20, power=2.3)
    b.loft([((0, 0.01, 1.46), 0.065, 0.06), ((0, 0.0, 1.53), 0.055, 0.052), ((0, -0.005, 1.6), 0.052, 0.05)], skin, NECK, n=12)
    for s, x in sides(b):
        # Shoulder, upper arm, forearm, hand.
        sh = b.head(f"upperarm_{s}")
        b.ellipsoid(sh + Vector((-x * 0.01, 0, 0.0)), (0.075, 0.07, 0.07), skin, arm_bones(s))
        b.loft(limb(b, f"upperarm_{s}", f"upperarm_{s}", UPPER_ARM, grow, 0.05, 1.0), skin, arm_bones(s), n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", FOREARM, grow, 0.0, 1.0), skin, arm_bones(s), n=12, side=(0, 0, 1))
        hand(b, s, x, skin)
        if not legs:
            continue
        # Thigh, knee, calf, foot.
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", THIGH, grow, 0.0, 1.0), skin, leg_bones(s), n=14, side=(1, 0, 0))
        b.ellipsoid(b.head(f"calf_{s}") + Vector((0, -0.01, 0)), (0.058, 0.06, 0.06), skin, leg_bones(s))
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", CALF, grow, 0.0, 0.92), skin, leg_bones(s), n=14, side=(1, 0, 0))
        foot(b, s, x, skin, grow)


def hand(b, s, x, skin):
    """A palm and four fingers and a thumb along their bones, so fists and grips work."""
    bones = [f"hand_{s}", f"lowerarm_{s}"]
    wrist = b.head(f"hand_{s}")
    knuckles = b.head(f"middle_01_{s}")
    b.loft([(wrist, 0.03, 0.022), (wrist.lerp(knuckles, 0.5), 0.043, 0.02), (knuckles + Vector((-x * 0.005, 0, 0)), 0.045, 0.017)],
           skin, bones, n=12, power=2.6, side=(0, 1, 0))
    for f in ("index", "middle", "ring", "pinky"):
        chain = [f"{f}_01_{s}", f"{f}_02_{s}", f"{f}_03_{s}"]
        pts = [b.head(c) for c in chain] + [b.tail(chain[-1])]
        r = 0.009 if f == "pinky" else 0.0105
        b.loft([(p, r * (1 - 0.1 * i), r * (1 - 0.1 * i)) for i, p in enumerate(pts)], skin, chain + [f"hand_{s}"], n=8)
    chain = [f"thumb_01_{s}", f"thumb_02_{s}", f"thumb_03_{s}"]
    pts = [b.head(c) for c in chain] + [b.tail(chain[-1])]
    b.loft([(p, 0.013 - 0.001 * i, 0.012 - 0.001 * i) for i, p in enumerate(pts)], skin, chain + [f"hand_{s}"], n=8)


def foot(b, s, x, mat, grow=0.0, height=0.0):
    ankle = b.head(f"foot_{s}")
    ball = b.head(f"ball_{s}")
    toe = b.tail(f"ball_{s}")
    rings = [(Vector((ankle.x, ankle.y + 0.045, 0.05 + height * 0.3)), 0.038 + grow, 0.045 + grow + height * 0.3),
             (Vector((ankle.x, ankle.y, 0.06 + height * 0.5)), 0.045 + grow, 0.06 + grow + height * 0.5),
             (Vector((ball.x, (ankle.y + ball.y) / 2, 0.045 + height * 0.3)), 0.048 + grow, 0.045 + grow + height * 0.3),
             (Vector((ball.x, ball.y, 0.03)), 0.05 + grow, 0.03 + grow),
             (Vector((toe.x, toe.y + 0.01, 0.022)), 0.038 + grow, 0.022 + grow)]
    b.loft(rings, mat, leg_bones(s), n=12, power=2.6, side=(1, 0, 0))


def face(b, skin):
    """Head and face: a chin, a brow ridge with arched brows, big readable eyes
    with lids and a highlight, a nose with a bridge and nostril wings, lips and ears."""
    eye_w = b.mat("EyeWhite", (245, 245, 240))
    iris = b.mat("Iris", (80, 60, 40))
    pupil = b.mat("Pupil", (20, 15, 12))
    shine = b.mat("Shine", (255, 255, 255))
    brow = b.mat("Brow", (80, 55, 38))
    lips = b.mat("Lips", (185, 115, 100))
    # The skull: wide at the cheekbones, narrowing to a squarish jaw.
    rings = [(1.535, 0.035, 0.03, -0.08), (1.55, 0.062, 0.066, -0.055), (1.575, 0.08, 0.088, -0.034), (1.615, 0.088, 0.098, -0.022),
             (1.66, 0.093, 0.106, -0.016), (1.71, 0.092, 0.106, -0.012), (1.76, 0.082, 0.096, -0.01), (1.8, 0.055, 0.066, -0.008),
             (1.818, 0.016, 0.02, -0.008)]
    b.loft([((0, y, z), rx, rz) for z, rx, rz, y in rings], skin, HEAD, n=22, power=2.25, side=(1, 0, 0))
    b.ellipsoid((0, -0.079, 1.558), (0.03, 0.02, 0.02), skin, HEAD)  # chin
    for s, x in sides(b):
        # Ears: nearly flush with the head. Ears that stick out are mostly ink outline at game
        # size and read as dark pads.
        b.ellipsoid((x * 0.084, -0.005, 1.655), (0.012, 0.022, 0.032), skin, HEAD, rot=(0, 0, x * -10))
        # Eyes: white, iris, pupil and a highlight, under a lid; bigger than life, so they read.
        c = Vector((x * 0.037, -0.103, 1.682))
        b.ellipsoid(c, (0.023, 0.011, 0.016), eye_w, HEAD, 12, 6)
        b.ellipsoid(c + Vector((0, -0.007, -0.001)), (0.011, 0.006, 0.012), iris, HEAD, 10, 6)
        b.ellipsoid(c + Vector((0, -0.01, -0.001)), (0.0058, 0.004, 0.0064), pupil, HEAD, 8, 4)
        b.ellipsoid(c + Vector((x * 0.003 - 0.002, -0.012, 0.004)), (0.0022, 0.0015, 0.0022), shine, HEAD, 6, 4)
        b.ellipsoid(c + Vector((0, 0.001, 0.011)), (0.026, 0.013, 0.007), skin, HEAD, 12, 6)  # upper lid
        # Brows: arched, thick at the inner end.
        b.loft([(Vector((x * 0.014, -0.112, 1.703)), 0.007, 0.006), (Vector((x * 0.036, -0.112, 1.709)), 0.007, 0.006),
                (Vector((x * 0.057, -0.104, 1.704)), 0.004, 0.004)], brow, HEAD, n=8, side=(0, 1, 0))
        b.ellipsoid((x * 0.012, -0.12, 1.632), (0.008, 0.008, 0.007), skin, HEAD)  # nostril wings
    b.loft([((0, -0.104, 1.7), 0.075, 0.01), ((0, -0.108, 1.706), 0.07, 0.008)], skin, HEAD, n=10, power=3, side=(1, 0, 0))  # brow ridge
    b.loft([((0, -0.108, 1.69), 0.007, 0.006), ((0, -0.12, 1.662), 0.009, 0.009), ((0, -0.128, 1.64), 0.012, 0.011),
            ((0, -0.124, 1.63), 0.009, 0.008)], skin, HEAD, n=10)  # nose
    b.loft([((-0.024, -0.112, 1.603), 0.005, 0.004), ((0, -0.118, 1.606), 0.008, 0.006), ((0.024, -0.112, 1.603), 0.005, 0.004)],
           lips, HEAD, n=8, side=(0, 1, 0))  # upper lip
    b.loft([((-0.02, -0.109, 1.594), 0.005, 0.004), ((0, -0.114, 1.592), 0.009, 0.007), ((0.02, -0.109, 1.594), 0.005, 0.004)],
           lips, HEAD, n=8, side=(0, 1, 0))  # lower lip


def male_body(name, legs=True, skin=SKIN):
    """The male body in linen shorts; without legs, for long robes, which would otherwise show
    the legs pushing through in a long stride (the shoes are their own piece). skin is the
    skin's color (a vampire's is pale)."""
    b = Body()
    skin = b.mat("Skin", skin)
    shorts = b.mat("Shorts", LINEN)
    body_shapes(b, skin, shorts, legs=legs)
    b.loft(torso_rings(b, 0.006, top=0.97), shorts, TORSO, n=20, power=2.3)
    for s, x in sides(b) if legs else ():
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", THIGH[:2], 0.006, 0.0, 0.3), shorts, leg_bones(s), n=14, side=(1, 0, 0))
    face(b, skin)
    b.export(name)


def scalp(b, hair, grow=0.0):
    rings = [(1.67, 0.096, 0.11, -0.005), (1.72, 0.096, 0.112, -0.008), (1.77, 0.084, 0.1, -0.008), (1.81, 0.056, 0.07, -0.008),
             (1.83, 0.015, 0.02, -0.008)]
    b.loft([((0, y + 0.006, z), rx + grow, rz + grow) for z, rx, rz, y in rings], hair, HEAD, n=20, power=2.2, side=(1, 0, 0))


def hair_long(name, color):
    b = Body()
    hair = b.mat("Hair", color)
    scalp(b, hair, 0.002)
    # Falling behind the shoulders.
    b.loft([((0, 0.03, 1.72), 0.1, 0.08), ((0, 0.06, 1.6), 0.1, 0.06), ((0, 0.08, 1.48), 0.11, 0.04), ((0, 0.09, 1.38), 0.09, 0.025)],
           hair, NECK + ["spine_02"], n=16, power=2.4)
    for s, x in sides(b):  # locks framing the face
        b.loft([((x * 0.09, -0.03, 1.72), 0.02, 0.025), ((x * 0.1, -0.03, 1.62), 0.02, 0.025), ((x * 0.095, -0.02, 1.52), 0.012, 0.015)],
               hair, HEAD + ["spine_03"], n=8, side=(0, 1, 0))
    b.export(name)


def beard_shapes(b, hair, full=False):
    if full:  # long and full, to the chest
        b.loft([((0, -0.06, 1.65), 0.09, 0.045), ((0, -0.09, 1.58), 0.08, 0.045), ((0, -0.1, 1.5), 0.06, 0.035),
                ((0, -0.1, 1.43), 0.03, 0.02)], hair, HEAD, n=16, power=2.2, side=(1, 0, 0))
    else:
        b.loft([((0, -0.06, 1.64), 0.085, 0.04), ((0, -0.085, 1.59), 0.07, 0.04), ((0, -0.095, 1.54), 0.045, 0.03),
                ((0, -0.09, 1.51), 0.02, 0.015)], hair, HEAD, n=16, power=2.2, side=(1, 0, 0))
    b.box((0, -0.118, 1.615), (0.05, 0.012, 0.012), hair, HEAD)  # moustache


def beard(name, color, full=False):
    b = Body()
    beard_shapes(b, b.mat("Hair", color), full)
    b.export(name)


def wizard_beard(b, color):
    """A great flowing beard to the belly in locks, a drooping moustache, and long hair falling
    beside the face."""
    hair, dark = b.mat("Hair", color), b.mat("HairDark", tuple(int(c * 0.8) for c in color))
    b.loft([((0, -0.06, 1.66), 0.092, 0.048), ((0, -0.095, 1.58), 0.088, 0.05), ((0, -0.11, 1.48), 0.075, 0.04),
            ((0, -0.115, 1.38), 0.05, 0.03), ((0, -0.11, 1.3), 0.012, 0.01)], hair, HEAD, n=16, power=2.2, side=(1, 0, 0))
    for x, l in ((-0.045, 1.36), (0.0, 1.31), (0.045, 1.37)):  # locks
        b.loft([(Vector((x, -0.13, 1.56)), 0.018, 0.012), (Vector((x * 1.1, -0.135, 1.45)), 0.02, 0.012), (Vector((x * 0.9, -0.125, l)), 0.004, 0.004)],
               dark, HEAD, n=8, side=(1, 0, 0))
    for s, x in ((1, 1), (-1, -1)):
        b.loft([(Vector((x * 0.012, -0.125, 1.62)), 0.012, 0.01), (Vector((x * 0.05, -0.125, 1.6)), 0.012, 0.01), (Vector((x * 0.065, -0.11, 1.54)), 0.004, 0.004)],
               hair, HEAD, n=8, side=(0, 0, 1))  # moustache
        b.loft([(Vector((x * 0.09, -0.04, 1.74)), 0.022, 0.03), (Vector((x * 0.1, -0.05, 1.62)), 0.024, 0.03), (Vector((x * 0.1, -0.04, 1.5)), 0.008, 0.012)],
               hair, HEAD + ["spine_03"], n=8, side=(0, 1, 0))  # hair beside the face


def boots(name, leather=LEATHER, dark=LEATHER_DARK, fur=None):
    b = Body()
    leather, dark, buckle = b.mat("Boot", leather), b.mat("BootDark", dark), b.mat("Buckle", STEEL, metal=True)
    for s, x in sides(b):
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", [(0.068, 0.074), (0.058, 0.062), (0.05, 0.054), (0.048, 0.05)], 0.008, 0.45, 0.96),
               leather, leg_bones(s), n=14, side=(1, 0, 0))
        cuff = b.mat("Fur", fur) if fur else dark
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", [(0.084, 0.09), (0.078, 0.084)] if fur else [(0.08, 0.086), (0.076, 0.082)], 0.0, 0.42, 0.5),
               cuff, leg_bones(s), n=14, side=(1, 0, 0))  # folded cuff, or a fur trim
        foot(b, s, x, leather, 0.014, 0.03)
        b.loft([((x * 0.089, b.head(f"foot_{s}").y + 0.05, 0.012), 0.05, 0.012), ((x * 0.089, b.tail(f"ball_{s}").y + 0.005, 0.012), 0.05, 0.012)],
               dark, leg_bones(s), n=12, power=3, side=(1, 0, 0))  # sole
        if not fur:
            c = b.head(f"calf_{s}").lerp(b.tail(f"calf_{s}"), 0.72)
            b.box(c + Vector((x * 0.055, 0, 0)), (0.01, 0.03, 0.022), buckle, leg_bones(s))
    b.export(name)


def shoes(name, color):
    """Soft, low shoes, for robes."""
    b = Body()
    leather = b.mat("Shoe", color)
    sole = b.mat("Sole", LEATHER_DARK)
    for s, x in sides(b):
        foot(b, s, x, leather, 0.01, 0.0)
        b.loft([((x * 0.089, b.head(f"foot_{s}").y + 0.05, 0.01), 0.048, 0.01), ((x * 0.089, b.tail(f"ball_{s}").y + 0.005, 0.01), 0.046, 0.01)],
               sole, leg_bones(s), n=12, power=3, side=(1, 0, 0))
    b.export(name)


# ---- the knight (art/concept/knight_s51_*.png) ----

def embroidery_png(name, thread, cloth):
    """A tileable embroidered pattern for trims, written next to the pieces as Trim_<name>.png:
    a lattice of diamonds in thread color with a dot in each, on cloth color, so it reads the
    same along a hem or down a front."""
    path = out_dir / f"Trim_{name}.png"
    size = 32
    rows = []
    for y in range(size):
        row = []
        for x in range(size):
            u, v = (x + 0.5) % 16 - 8, (y + 0.5) % 16 - 8
            d = abs(u) + abs(v)
            k = max(0.0, 1 - abs(d - 6.5) / 1.2)  # the diamond's outline
            k = max(k, 1.0 if d < 2.2 else 0.0)  # a dot in the middle
            c, e = (x + 8.5) % 16 - 8, (y + 8.5) % 16 - 8
            k = max(k, 0.8 if abs(c) + abs(e) < 1.6 else 0.0)  # and one between
            shade = 0.85 + 0.15 * (v < 0)  # thread lit from above
            row.append(tuple(int(cl + (th * shade - cl) * k) for th, cl in zip(thread, cloth)))
        rows.append(row)
    return write_png(path, rows)


def folded(levels, folds, phase=0.0):
    """A point on a skirt with folds: levels are (z, half width, half depth, fold depth,
    y offset) from the top down; point(t, a) is at height t (0 at the first level, len - 1 at
    the last, fractions between) and angle a around (0 at +X, -pi/2 at the front)."""
    def point(t, a, out=0.0):
        i = min(int(t), len(levels) - 2)
        f = t - i
        z, rx, ry, depth, y = [p + (q - p) * f for p, q in zip(levels[i], levels[i + 1])]
        r = depth * math.sin(folds * a + phase) + out
        return Vector((math.cos(a) * (rx + r), y + math.sin(a) * (ry + r), z))
    return point


def folded_skirt(b, levels, folds, mat, bones, hem=None, phase=0.0, steps=3, n=None, rags=0.0):
    """A long skirt through folded() levels, steps rings between levels; with hem (a material),
    a band around the bottom over the folds; with rags (meters), a ragged hem, its tatters
    hanging up to that much lower. Returns its point function, for trims."""
    point = folded(levels, folds, phase)
    n = n or folds * 6
    last = len(levels) - 1
    ts = [i / steps for i in range(last * steps + 1)]
    rows = [[point(t, 2 * math.pi * j / n) for j in range(n)] for t in ts]
    if rags:
        tatter = lambda j: rags * (j % 2) * (0.6 + 0.4 * math.sin(j * 2.7))
        rows[-1] = [q - Vector((0, 0, tatter(j))) for j, q in enumerate(rows[-1])]
    b.tube(rows, mat, bones)
    if hem:
        b.tube([[point(t, 2 * math.pi * j / n, 0.005) for j in range(n)] for t in (last - 0.22, last)], hem, bones)
    return point


def front_strip(b, point, t0, t1, width, mat, bones, widen=1.0, out=0.004, a=-math.pi / 2):
    """A strip down a folded skirt (a trim down the front): width meters across at the top,
    times widen at the bottom."""
    rows = []
    for k in range(9):
        t = t0 + (t1 - t0) * k / 8
        w = width * (1 + (widen - 1) * k / 8)
        rx = point(t, 0).x
        rows.append([point(t, a + w / max(rx, 0.05) * c, out) for c in (-0.5, -0.25, 0, 0.25, 0.5)])
    b.sheet(rows, mat, mat, bones, 0.004)


class Plate:
    """The colors of a plate armor set (the knight_* pieces): plate and its rolled edges (metal),
    the cloth of tabard, sash, scarf and cloak, and the gold of buckles and hems."""

    def __init__(self, plate, edge, cloth, cloth_dark, gold=(205, 160, 70), gem=(60, 110, 200), sash=None, cloak=None, lining=None):
        self.plate, self.edge, self.cloth, self.cloth_dark = plate, edge, cloth, cloth_dark
        self.gold, self.gem = gold, gem
        self.sash = sash or cloth
        self.cloak, self.lining = cloak or cloth, lining or cloth_dark

    def mats(self, b):
        """(plate, edge) materials."""
        return b.mat("Steel", self.plate, metal=True), b.mat("Edge", self.edge, metal=True)


KNIGHT = Plate((138, 156, 182), (78, 90, 112), (178, 36, 36), (118, 22, 26))  # art/concept/knight_s51_*.png
BRONZE = Plate((200, 150, 74), (142, 98, 46), (178, 36, 36), (118, 22, 26), gold=(225, 180, 70))  # art/concept/hoplite_s51.png
PALADIN = Plate((198, 204, 214), (200, 158, 72), (240, 236, 222), (196, 188, 168), gold=(225, 180, 70), gem=(200, 50, 50),
                sash=(60, 90, 170), cloak=(60, 90, 170), lining=(225, 220, 205))  # bright steel trimmed in gold, white cloth, blue


def ring_band_along(b, bone, at, r, mat, bones, width=0.02):
    """A thin band around a limb at a point along its bone."""
    h, t = b.head(bone), b.tail(bone)
    d = (t - h).normalized()
    c = h.lerp(t, at)
    b.loft([(c - d * width / 2, r, r), (c + d * width / 2, r, r)], mat, bones, n=14, power=2.2, side=(0, 0, 1), caps=False)


def ring_band(b, center, rx, ry, height, mat, bones, n=20, power=2.4):
    """A short open band around something (a rim, a strap)."""
    c = Vector(center)
    b.loft([(c + Vector((0, 0, height / 2)), rx, ry), (c - Vector((0, 0, height / 2)), rx, ry)], mat, bones, n=n, power=power,
           side=(1, 0, 0), caps=False)


def chainmail_png():
    """A tileable chainmail texture (rows of rings, each row offset by half), written once next
    to the pieces: dark steel rings with a light upper rim on a near-black ground."""
    path = out_dir / "Chainmail.png"
    if path.exists():
        return path
    import math as m
    size, ring = 32, 8  # four rings across, rows half a ring apart
    px = []
    for y in range(size):
        row = []
        for x in range(size):
            best = (0.0, 0.0)
            for cy in range(-1, size // (ring // 2) + 2):
                oy = cy * ring / 2
                ox = (ring / 2) * (cy % 2)
                for cx in range(-1, size // ring + 2):
                    dx, dy = x + 0.5 - (cx * ring + ox), y + 0.5 - oy
                    for wx in (-size, 0, size):
                        for wy in (-size, 0, size):
                            d = m.hypot(dx + wx, dy + wy)
                            k = max(0.0, 1 - abs(d - ring * 0.42) / 1.3)  # the ring's wire
                            if k > best[0]:
                                best = (k, -(dy + wy) / max(d, 1e-6))
            k, up = best
            v = 0.12 + k * (0.45 + 0.35 * up)
            row.append((int(v * 150), int(v * 158), int(v * 175)))
        px.append(row)
    return write_png(path, px)


def knight_mail(b):
    return b.textured("Mail", chainmail_png(), 0.05)


def knight_helm(name):
    """A close helm: a rounded skull rising to a point, a low crest over the top, a visor plate
    standing out over a dark slit, a raised brow band, breathing holes and rivets."""
    import math as m
    b = Body()
    steel, edge = KNIGHT.mats(b)
    dark = b.mat("Slit", (20, 20, 26))
    rings = [(1.52, 0.11, 0.124), (1.58, 0.108, 0.124), (1.66, 0.106, 0.123), (1.72, 0.103, 0.119), (1.77, 0.094, 0.108),
             (1.81, 0.077, 0.089), (1.84, 0.053, 0.062), (1.86, 0.027, 0.032), (1.87, 0.006, 0.007)]
    b.loft([((0, -0.008, z), rx, ry) for z, rx, ry in rings], steel, HEAD, n=24, power=2.25, side=(1, 0, 0))
    ring_band(b, (0, -0.008, 1.523), 0.114, 0.128, 0.014, edge, HEAD + ["spine_03"], n=24, power=2.25)  # bottom rim
    ring_band(b, (0, -0.008, 1.735), 0.106, 0.122, 0.018, edge, HEAD, n=24, power=2.25)  # brow band
    # The crest: a low fin from the brow over the top to the back, peaking in a point.
    crest = [(0, -0.12, 1.75), (0, -0.09, 1.83), (0, -0.04, 1.885), (0, 0.02, 1.88), (0, 0.08, 1.83), (0, 0.11, 1.74)]
    b.loft([(Vector(p), 0.007, 0.012 if i in (2, 3) else 0.008) for i, p in enumerate(crest)], steel, HEAD, n=8, power=2.0, side=(1, 0, 0))
    b.rod((0, -0.04, 1.875), (0, -0.065, 1.95), 0.016, steel, HEAD, 6, taper=0.1)
    # The slit: a dark band around the front, under the brow band.
    b.loft([((0, -0.008, 1.718), 0.107, 0.124), ((0, -0.008, 1.706), 0.108, 0.125)], dark, HEAD, n=24, power=2.25, side=(1, 0, 0),
           caps=False)
    # The visor: a plate standing out over the face, from the slit down to the chin, pointed
    # a little in front like a snout.
    rows = []
    for i, z in enumerate((1.70, 1.665, 1.625, 1.585, 1.555)):
        out = 0.004 + 0.008 * m.sin(i / 4 * m.pi * 0.8)  # bulging a little over the mouth
        row = []
        for j in range(11):
            a = m.pi * (-0.4 + 0.8 * j / 10)
            rx, ry = 0.109 + out, 0.126 + out + 0.006 * m.cos(a * 2.5)
            row.append((m.sin(a) * rx, -0.008 - m.cos(a) * ry, z))
        rows.append(row)
    b.sheet(rows, steel, edge, HEAD, 0.008)
    b.box((0, -0.143, 1.67), (0.014, 0.01, 0.06), dark, HEAD)  # the slit's downstroke, a T over the face
    for s_, x in sides(b):  # the visor's edges, darker
        b.loft([(Vector((x * 0.1, -0.075, 1.70)), 0.005, 0.005), (Vector((x * 0.105, -0.07, 1.62)), 0.005, 0.005),
                (Vector((x * 0.095, -0.075, 1.555)), 0.005, 0.005)], edge, HEAD, n=6, side=(0, 1, 0))
    for i in range(4):  # breathing holes on the right of the visor
        for j in range(2):
            a = 0.45 + 0.1 * j
            b.ellipsoid((-m.sin(a) * 0.12, -0.008 - m.cos(a) * 0.138, 1.64 - i * 0.022), (0.005, 0.005, 0.005), dark, HEAD, 6, 4)
    for s_, x in sides(b):
        b.ellipsoid((x * 0.118, -0.02, 1.69), (0.012, 0.012, 0.012), edge, HEAD, 8, 5)  # the visor's pivots
        for a in (0.35, 0.75):
            b.ellipsoid((x * 0.107 * m.sin(a), -0.008 - 0.123 * m.cos(a), 1.735), (0.006, 0.006, 0.006), steel, HEAD, 6, 4)
    b.export(name)


def great_helm(name, p=PALADIN):
    """A great helm: a tall, nearly flat-topped steel pot with eye slits across the front, a gold
    cross over the face (a band across the brow and a strip down to the chin), gold rims,
    breathing holes in a cross below the slits on the right, and a gold crown band on top."""
    b = Body()
    steel, gold = b.mat("Steel", p.plate, metal=True), b.mat("Gold", p.gold, metal=True)
    dark = b.mat("Slit", (20, 20, 26))
    y0 = -0.008
    rings = [(1.52, 0.113, 0.128), (1.6, 0.116, 0.131), (1.7, 0.117, 0.132), (1.78, 0.115, 0.129), (1.83, 0.109, 0.122),
             (1.855, 0.092, 0.103), (1.87, 0.055, 0.062), (1.876, 0.01, 0.01)]
    b.loft([((0, y0, z), rx, ry) for z, rx, ry in rings], steel, HEAD, n=28, power=2.4, side=(1, 0, 0))
    ring_band(b, (0, y0, 1.522), 0.117, 0.132, 0.016, gold, HEAD + ["spine_03"], n=28, power=2.4)  # bottom rim
    ring_band(b, (0, y0, 1.83), 0.112, 0.125, 0.018, gold, HEAD, n=28, power=2.4)  # crown band
    b.loft([((0, y0, 1.712), 0.118, 0.133), ((0, y0, 1.698), 0.118, 0.133)], dark, HEAD, n=28, power=2.4, side=(1, 0, 0),
           caps=False)  # the eye slits, round the front (the cross's strip parts them)
    ring_band(b, (0, y0, 1.735), 0.119, 0.134, 0.024, gold, HEAD, n=28, power=2.4)  # the cross's arm, over the brow
    front = lambda z: y0 - 0.136 - 0.004 * (1.7 - z)
    b.loft([(Vector((0, front(z), z)), 0.016, 0.006) for z in (1.748, 1.7, 1.62, 1.54)], gold, HEAD, n=8, power=2.6,
           side=(1, 0, 0))  # and its strip down the face
    for da, dz in ((0, 0), (0.12, 0), (-0.12, 0), (0, 0.016), (0, -0.016)):  # breathing holes in a small cross
        a = 0.55 + da
        b.ellipsoid((-math.sin(a) * 0.127, y0 - math.cos(a) * 0.141, 1.62 + dz), (0.005, 0.005, 0.005), dark, HEAD, 6, 4)
    for s_, x in sides(b):
        for z in (1.6, 1.78):  # rivets down the sides
            b.ellipsoid((x * 0.12, y0, z), (0.007, 0.007, 0.007), gold, HEAD, 6, 4)
    for i in range(8):  # small points on the crown band
        a = 2 * math.pi * (i + 0.5) / 8
        q = Vector((math.cos(a) * 0.11, y0 + math.sin(a) * 0.123, 1.838))
        b.rod(q, q + Vector((0, 0, 0.03)), 0.01, gold, HEAD, 4, taper=0.1)
    b.export(name)


def m_sin(a):
    import math
    return math.sin(a)


def m_cos(a):
    import math
    return math.cos(a)


def knight_cuirass(name, p=KNIGHT, emblem=False):
    """The torso: a gorget, a breastplate with a ridge and a plackart over the belly, fauld
    lames over the hips, a mail skirt below, a red sash and a leather belt, and a red tabard
    hanging in front and behind. With emblem, a gold sun on the tabard's front."""
    import math as m
    b = Body()
    (steel, edge), mail = p.mats(b), knight_mail(b)
    red, red_dark, sash = b.mat("Cloth", p.cloth), b.mat("ClothDark", p.cloth_dark), b.mat("Sash", p.sash)
    leather, gold = b.mat("Leather", LEATHER), b.mat("Gold", p.gold, metal=True)
    b.loft(torso_rings(b, 0.016, top=1.49, bottom=0.95), mail, TORSO, n=20, power=2.3)  # mail under it all
    # Breastplate: one broad, flat-fronted plate, boxier than the body under it, only a little
    # narrower at the waist (a deep chest over a narrow waist reads as a bust).
    chest = [(1.04, 0.176, 0.132, -0.012), (1.14, 0.186, 0.138, -0.02), (1.26, 0.21, 0.145, -0.028), (1.36, 0.222, 0.145, -0.025),
             (1.43, 0.21, 0.128, -0.012), (1.47, 0.15, 0.1, 0.0)]
    b.loft([((0, 0.01 + f, z), rx, ry) for z, rx, ry, f in chest], steel, TORSO, n=24, power=2.8, side=(1, 0, 0))
    for z, r in ((1.2, 0.188), (1.145, 0.187)):  # lames across the belly, under the chest plate
        ring_band(b, (0, -0.008, z), r, 0.141, 0.008, edge, TORSO, n=24, power=2.8)
    ridge = [(0, -0.135, 1.42), (0, -0.166, 1.3), (0, -0.158, 1.18), (0, -0.138, 1.06)]
    b.loft([(Vector(p), 0.006, 0.008) for p in ridge], steel, TORSO, n=6, side=(1, 0, 0))
    # The plackart: a second plate over the belly, rising to a point.
    plack = [(1.03, 0.178, 0.137, -0.012), (1.1, 0.186, 0.143, -0.02), (1.16, 0.12, 0.1, -0.05)]
    b.loft([((0, 0.01 + f, z), rx, ry) for z, rx, ry, f in plack], steel, TORSO, n=24, power=2.6, side=(1, 0, 0))
    ring_band(b, (0, 0.0, 1.042), 0.18, 0.14, 0.012, edge, TORSO, n=24, power=2.6)
    for s, x in sides(b):  # the armholes' rolled edges
        a = b.head(f"upperarm_{s}")
        b.loft([(Vector((x * 0.17, 0.0, 1.47)), 0.012, 0.012), (Vector((x * 0.205, -0.02, 1.38)), 0.012, 0.012),
                (Vector((x * 0.19, -0.04, 1.3)), 0.01, 0.01)], edge, TORSO, n=6)
    # Gorget: two overlapping rings around the neck.
    for i, (z, r) in enumerate(((1.475, 0.13), (1.51, 0.1))):
        b.loft([((0, 0.0, z + 0.022), r * 0.92, r * 0.85), ((0, 0.0, z - 0.022), r, r * 0.92)], steel, NECK + TORSO, n=20, power=2.3,
               caps=False)
        ring_band(b, (0, 0.0, z - 0.022), r + 0.002, r * 0.92 + 0.002, 0.008, edge, NECK + TORSO)
    # Faulds: three lames flaring over the hips.
    for i in range(3):
        z = 1.0 - i * 0.05
        r = 0.19 + i * 0.012
        b.loft([((0, 0.01, z + 0.03), r - 0.01, r * 0.74 - 0.01), ((0, 0.01, z - 0.03), r, r * 0.74)], steel, TORSO, n=24, power=2.6,
               caps=False)
        ring_band(b, (0, 0.01, z - 0.03), r + 0.003, r * 0.74 + 0.003, 0.008, edge, TORSO, n=24, power=2.6)
    b.loft([((0, 0.01, 0.88), 0.2, 0.15), ((0, 0.01, 0.78), 0.21, 0.155)], mail, TORSO, n=24, power=2.4, caps=False)  # mail skirt
    # Tassets: a plate hanging over each thigh from the faulds, in two lames.
    for s_, x in sides(b):
        for i, (z0, z1) in enumerate(((0.9, 0.79), (0.8, 0.69))):
            rows = []
            for z in (z0, (z0 + z1) / 2, z1):
                t = (z0 - z) / (z0 - z1)
                row = []
                for j in range(7):
                    a = -1.0 + 2.0 * j / 6  # across the thigh's front and outside
                    ang = x * (0.55 + 0.7 * (a + 1) / 2) * 1.0
                    r = 0.122 + 0.012 * i + 0.006 * t
                    c = Vector((x * 0.1, -0.005, z))
                    row.append(c + Vector((m_sin(ang) * r, -m_cos(ang) * r * 0.9, 0)))
                rows.append(row if x > 0 else list(reversed(row)))
            b.sheet(rows, steel, edge, TORSO, 0.008)
    # Belt and sash: a leather belt with a gold buckle, a red sash tied above it.
    ring_band(b, (0, 0.0, 1.0), 0.196, 0.148, 0.03, leather, TORSO, n=24, power=2.6)
    b.ellipsoid((0, -0.152, 1.0), (0.03, 0.01, 0.026), gold, TORSO, 12, 6)  # a round buckle with a blue stone
    b.ellipsoid((0, -0.16, 1.0), (0.016, 0.006, 0.014), b.mat("Gem", p.gem), TORSO, 10, 5)
    ring_band(b, (0, 0.0, 1.04), 0.19, 0.145, 0.03, sash, TORSO, n=24, power=2.6)
    b.ellipsoid((0.11, -0.13, 1.04), (0.035, 0.02, 0.025), sash, TORSO)  # the knot
    b.loft([(Vector((0.11, -0.14, 1.03)), 0.02, 0.006), (Vector((0.125, -0.15, 0.95)), 0.025, 0.006), (Vector((0.13, -0.155, 0.88)), 0.02, 0.006)],
           red_dark, TORSO, n=8, power=3, side=(1, 0, 0))  # its end
    # Tabard: a red panel hanging in front to the knees (it follows both thighs) and one behind
    # (it hangs from the hips).
    for y, sgn, bones in ((-0.16, -1, TORSO), (0.165, 1, ["pelvis", "spine_01", "spine_02"])):
        top, bottom = Vector((0, y, 0.98)), Vector((0, y + sgn * 0.03, 0.58))
        rows = []
        for i in range(6):
            t = i / 5
            p = top.lerp(bottom, t)
            w = 0.065 + 0.02 * t
            rows.append([p + Vector((x * w, sgn * 0.01 * abs(x) * t, 0)) for x in (-1, -0.5, 0, 0.5, 1)])
        b.sheet(rows if sgn > 0 else [list(reversed(r)) for r in rows], red, red_dark, bones, 0.008)
        b.loft([(bottom + Vector((-0.086, 0, 0.012)), 0.004, 0.008), (bottom + Vector((0.086, 0, 0.012)), 0.004, 0.008)], gold, bones, n=6,
               side=(0, 0, 1))  # gold hem
    if emblem:  # a sun: a disc with rays, on the front panel
        c = Vector((0, -0.172, 0.8))
        b.ellipsoid(c, (0.03, 0.006, 0.03), gold, TORSO, 14, 5)
        for i in range(8):
            a = i * m.pi / 4
            d = Vector((m.cos(a), 0, m.sin(a)))
            b.loft([(c + d * 0.03, 0.009, 0.003), (c + d * 0.055, 0.001, 0.003)], gold, TORSO, n=6, side=(0, 1, 0))
    b.export(name)


def knight_arms(name, p=KNIGHT):
    """Mail sleeves (the upper arm shows its mail under the pauldron), a couter with a fan over
    the elbow, a vambrace, and gauntlets with flared cuffs, plates over the back of the hand and
    the fingers."""
    b = Body()
    (steel, edge), mail = p.mats(b), knight_mail(b)
    for s, x in sides(b):
        bones = arm_bones(s)
        b.loft(limb(b, f"upperarm_{s}", f"upperarm_{s}", UPPER_ARM, 0.012, 0.0, 1.0), mail, bones, n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", FOREARM, 0.01, 0.0, 0.85), mail, bones, n=12, side=(0, 0, 1))
        ring_band_along(b, f"upperarm_{s}", 0.86, 0.063, edge, bones)  # a leather-backed strap where the couter hangs
        el = b.head(f"lowerarm_{s}")
        b.ellipsoid(el, (0.06, 0.063, 0.063), steel, bones)  # couter
        b.ellipsoid(el + Vector((0, 0.05, 0)), (0.06, 0.014, 0.068), steel, bones, rot=(0, 0, 0))  # its fan, behind
        b.ellipsoid(el + Vector((0, 0.058, 0)), (0.012, 0.004, 0.012), edge, bones, 6, 4)
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.056, 0.052), (0.055, 0.05), (0.05, 0.045)], 0.008, 0.14, 0.72), steel, bones,
               n=14, power=2.3, side=(0, 0, 1))  # vambrace
        # Gauntlet: a cuff flaring over the wrist, plates over the back of the hand, plated
        # fingers along their bones.
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.045, 0.04), (0.058, 0.052), (0.066, 0.06)], 0.0, 0.7, 1.0), steel, bones,
               n=14, power=2.3, side=(0, 0, 1), caps=False)
        hb = [f"hand_{s}", f"lowerarm_{s}"]
        wrist, knuckles = b.head(f"hand_{s}"), b.head(f"middle_01_{s}")
        b.loft([(wrist, 0.036, 0.026), (wrist.lerp(knuckles, 0.5), 0.05, 0.026), (knuckles, 0.05, 0.022)], steel, hb, n=12, power=2.8,
               side=(0, 1, 0))
        for f in ("index", "middle", "ring", "pinky"):
            chain = [f"{f}_01_{s}", f"{f}_02_{s}", f"{f}_03_{s}"]
            pts = [b.head(c) for c in chain] + [b.tail(chain[-1])]
            r = 0.0125 if f != "pinky" else 0.011
            b.loft([(p, r, r) for p in pts], steel, chain + [f"hand_{s}"], n=8, power=2.3)
            for p in pts[1:3]:
                b.ellipsoid(p, (r * 1.15,) * 3, edge, chain + [f"hand_{s}"], 8, 5)  # knuckle rings
        chain = [f"thumb_01_{s}", f"thumb_02_{s}", f"thumb_03_{s}"]
        pts = [b.head(c) for c in chain] + [b.tail(chain[-1])]
        b.loft([(p, 0.015, 0.014) for p in pts], steel, chain + [f"hand_{s}"], n=8, power=2.3)
    b.export(name)


def knight_legs(name, p=KNIGHT):
    """Mail chausses under cuisses on the thighs, poleyns with side fans over the knees, and
    greaves with a ridge down the shin."""
    b = Body()
    (steel, edge), mail = p.mats(b), knight_mail(b)
    b.loft(torso_rings(b, 0.012, top=1.0), mail, TORSO, n=20, power=2.3)
    for s, x in sides(b):
        bones = leg_bones(s)
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", THIGH, 0.014, 0.0, 1.0), mail, bones, n=14, side=(1, 0, 0))
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", CALF[:4], 0.01, 0.0, 0.8), mail, bones, n=14, side=(1, 0, 0))
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", [(0.1, 0.104), (0.092, 0.096), (0.08, 0.084), (0.072, 0.074)], 0.016, 0.3, 0.92), steel,
               bones, n=16, power=2.4, side=(1, 0, 0))  # cuisse
        kn = b.head(f"calf_{s}") + Vector((0, -0.03, 0))
        b.ellipsoid(kn, (0.068, 0.058, 0.074), steel, bones)  # poleyn
        b.ellipsoid(kn + Vector((x * 0.055, 0.015, 0)), (0.014, 0.066, 0.07), steel, bones)  # the fan, on the outside
        b.ellipsoid(kn + Vector((x * 0.07, 0.015, 0)), (0.007, 0.014, 0.014), edge, bones, 6, 4)
        b.ellipsoid(kn + Vector((0, -0.045, 0.0)), (0.012, 0.016, 0.012), edge, bones, 6, 4)  # a rivet in front
        for dz in (0.075, -0.075):  # lames above and below the knee
            ring_band(b, kn + Vector((0, 0.02, dz)), 0.07, 0.072, 0.03, steel, bones, n=16, power=2.4)
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", [(0.064, 0.07), (0.064, 0.074), (0.054, 0.06), (0.046, 0.05)], 0.012, 0.16, 0.9), steel,
               bones, n=16, power=2.4, side=(1, 0, 0))  # greave
        c0, c1 = b.head(f"calf_{s}"), b.tail(f"calf_{s}")
        b.loft([(c0.lerp(c1, t) + Vector((0, -0.08 + 0.025 * t, 0)), 0.006, 0.008) for t in (0.18, 0.5, 0.85)], steel, bones, n=6,
               side=(1, 0, 0))  # the shin's ridge
        ring_band(b, c0.lerp(c1, 0.9), 0.06, 0.064, 0.012, edge, bones, n=16, power=2.4)
    b.export(name)


def knight_sabatons(name, p=KNIGHT):
    """Steel shoes over leather: an ankle cuff, overlapping lames following the foot down to
    the toes, a pointed toe cap, a sole."""
    b = Body()
    (steel, edge), sole = p.mats(b), b.mat("Sole", LEATHER_DARK)
    for s, x in sides(b):
        bones = leg_bones(s)
        foot(b, s, x, sole, 0.004, 0.0)  # the leather shoe underneath
        ankle, ball, toe = b.head(f"foot_{s}"), b.head(f"ball_{s}"), b.tail(f"ball_{s}")

        def profile(t):
            """The foot's cross-section from the ankle (0) to the ball (1), as foot() makes it:
            (y, center height, half width, half height)."""
            y = ankle.y + (ball.y - ankle.y) * t
            if t < 0.5:
                k = t / 0.5
                return y, 0.06 - 0.015 * k, 0.045 + 0.003 * k, 0.06 - 0.015 * k
            k = (t - 0.5) / 0.5
            return y, 0.045 - 0.015 * k, 0.048 + 0.002 * k, 0.045 - 0.015 * k

        grow = 0.014
        for i in range(5):  # lames from the ankle to the toes, each overlapping the next
            t0, t1 = i / 5, min(1.0, (i + 1.3) / 5)
            r0, r1 = profile(t0), profile(t1)
            rings = [(Vector((x * 0.089, r[0], r[1])), r[2] + grow, r[3] + grow) for r in (r0, r1)]
            b.loft(rings, steel, bones, n=16, power=2.4, side=(1, 0, 0), caps=False)
            b.loft([(rings[1][0] + Vector((0, 0.004, 0)), rings[1][1] + 0.002, rings[1][2] + 0.002), (rings[1][0], rings[1][1] + 0.002, rings[1][2] + 0.002)],
                   edge, bones, n=16, power=2.4, side=(1, 0, 0), caps=False)
        yb, zb, wb, hb = profile(1.0)
        b.loft([(Vector((x * 0.089, yb, zb)), wb + grow, hb + grow), (Vector((x * 0.089, (yb + toe.y) / 2, 0.026)), 0.056, 0.036),
                (Vector((x * 0.089, toe.y + 0.004, 0.02)), 0.046, 0.026), (Vector((x * 0.089, toe.y - 0.024, 0.014)), 0.016, 0.012)],
               steel, bones, n=16, power=2.4, side=(1, 0, 0))  # toe cap, over the leather toe
        b.loft([(Vector((x * 0.089, ankle.y + 0.055, 0.008)), 0.052, 0.008), (Vector((x * 0.089, toe.y - 0.01, 0.008)), 0.03, 0.008)], sole,
               bones, n=12, power=3, side=(1, 0, 0))
        ring_band(b, ankle + Vector((0, 0.01, 0.06)), 0.062, 0.068, 0.05, steel, bones, n=16, power=2.4)  # ankle cuff
        ring_band(b, ankle + Vector((0, 0.01, 0.034)), 0.064, 0.07, 0.008, edge, bones, n=16, power=2.4)
    b.export(name)


def knight_mantle(name, p=KNIGHT, scarf=True):
    """Big layered pauldrons with raised guards, a scarf around the neck, and a long cloak
    hanging from the shoulders in folds."""
    import math as m
    b = Body()
    steel, edge = p.mats(b)
    red, cloak, lining = b.mat("Cloth", p.cloth), b.mat("Cloak", p.cloak), b.mat("Lining", p.lining)
    for s, x in sides(b):
        bones = arm_bones(s)
        sh = b.head(f"upperarm_{s}") + Vector((x * 0.02, 0, 0.03))
        b.ellipsoid(sh + Vector((x * 0.01, 0, 0.025)), (0.125, 0.135, 0.09), steel, bones, 20, 10)  # the dome over the shoulder
        b.ellipsoid(sh + Vector((x * 0.01, 0, 0.02)), (0.128, 0.138, 0.06), edge, bones, 20, 6)  # its rim
        for i in range(3):  # lames down the upper arm, each wider and lower
            c = sh + Vector((x * (0.055 + 0.03 * i), 0, -0.04 - 0.04 * i))
            r = 0.115 + 0.006 * i
            b.ellipsoid(c, (r * 0.72, r * 1.05, r * 0.4), steel, bones, 18, 8, rot=(0, x * (-32 - 6 * i), 0))
            b.ellipsoid(c + Vector((x * 0.004, 0, -0.012)), (r * 0.73, r * 1.06, r * 0.3), edge, bones, 18, 6, rot=(0, x * (-32 - 6 * i), 0))
        # A raised guard on top, against blows at the neck.
        g0 = sh + Vector((-x * 0.03, 0, 0.09))
        b.loft([(g0 + Vector((0, -0.08, -0.01)), 0.008, 0.03), (g0, 0.008, 0.04), (g0 + Vector((0, 0.08, -0.01)), 0.008, 0.03)], steel,
               bones + TORSO, n=8, power=2.6, side=(1, 0, 0))
        for a in (-0.06, 0.0, 0.06):
            b.ellipsoid(sh + Vector((x * 0.07, a, 0.05)), (0.007, 0.007, 0.007), edge, bones, 6, 4)
    # The scarf: a cowl around the neck, bunched over the shoulders.
    if scarf:
        b.loft([((0, 0.01, 1.575), 0.095, 0.095), ((0, 0.015, 1.545), 0.125, 0.12), ((0, 0.025, 1.5), 0.165, 0.135), ((0, 0.03, 1.47), 0.18, 0.14)],
               red, NECK + TORSO, n=24, power=2.3, caps=False)
        b.loft([((0, 0.01, 1.548), 0.128, 0.123), ((0, 0.012, 1.535), 0.135, 0.13)], b.mat("ClothDark", p.cloth_dark), NECK + TORSO, n=24,
               power=2.3, caps=False)  # a fold
    # The cloak: from the shoulders around the back down to the calves, widening, with folds
    # that deepen toward the hem. Columns run around from the left shoulder to the right.
    rows = []
    n_rows, n_cols = 16, 29
    for i in range(n_rows):
        t = i / (n_rows - 1)
        z = 1.47 - t * 1.3
        rx, ry = 0.215 + 0.2 * t, 0.165 + 0.2 * t
        row = []
        for j in range(n_cols):
            u = j / (n_cols - 1)
            a = m.pi * (-0.05 + 1.1 * u)  # 0 at the left side, round the back to the right
            fold = (0.004 + 0.05 * t) * m.sin(u * m.pi * 9 + 0.6 * t)
            px = m.cos(a) * (rx + fold)
            py = 0.02 + m.sin(a) * (ry + fold) + 0.08 * t * t
            row.append((px, py, z))
        rows.append(row)
    cloak_bones = ["spine_03", "spine_02", "spine_01", "pelvis", "clavicle_l", "clavicle_r"]
    b.power = 3.0
    b.sheet(rows, lining, cloak, cloak_bones, 0.014)  # the grid is the inner face; sheet offsets the outer one
    b.power = 6.0
    b.export(name)


class Robes:
    """The colors of a caster's robe set (the robe_* pieces): the robe, its darker mantle, the
    trim (a band of gold, say, embroidered in the mantle's color), the undersleeves, and the
    leather of belts and pouches."""

    def __init__(self, name, cloth, dark, band, under, leather=LEATHER, gold=(225, 180, 70), panel=None):
        self.name, self.cloth, self.dark, self.band, self.under = name, cloth, dark, band, under
        self.leather, self.gold, self.panel_color = leather, gold, panel

    def trim(self, b):
        return b.textured("Trim", embroidery_png(self.name, self.dark, self.band), 0.06)

    def panel(self, b):
        """The broad front panel: the trim, or with a panel color, the band's color embroidered on it."""
        if not self.panel_color:
            return self.trim(b)
        return b.textured("Panel", embroidery_png(self.name + "_Panel", self.band, self.panel_color), 0.07)


MAGE = Robes("Mage", (76, 110, 186), (40, 60, 132), (232, 190, 80), (150, 175, 222), panel=(30, 40, 88))  # blue and gold, after art/concept/mage_s51.png


def arm_frame(b, s):
    """The forearm's axis and two directions across it, in the T-pose."""
    d = (b.tail(f"lowerarm_{s}") - b.head(f"lowerarm_{s}")).normalized()
    u = Vector((0, 0, 1))
    v = d.cross(u).normalized()
    return d, u, v


def robe_belt(b, p, z=0.99, book=False):
    """A broad leather belt studded in gold with a big round buckle, pouches on both hips and a
    scroll case; with book, a small book hanging at the left hip."""
    leather, dark = b.mat("Leather", p.leather), b.mat("LeatherDark", LEATHER_DARK)
    gold = b.mat("Gold", p.gold, metal=True)
    ring_band(b, (0, 0.0, z), 0.198, 0.146, 0.05, leather, TORSO, n=24, power=2.5)
    for i in range(14):  # studs along it
        a = -math.pi / 2 + (i - 6.5) * 0.2
        if abs(i - 6.5) > 1:
            b.ellipsoid((math.cos(a) * 0.205, math.sin(a) * 0.153, z), (0.008, 0.006, 0.008), gold, TORSO, 6, 4)
    b.ellipsoid((0, -0.152, z), (0.042, 0.014, 0.036), gold, TORSO, 16, 6)
    b.ellipsoid((0, -0.162, z), (0.026, 0.008, 0.021), leather, TORSO, 12, 5)
    b.ellipsoid((0, -0.166, z), (0.014, 0.006, 0.012), b.mat("Stone", (70, 160, 230)), TORSO, 10, 5)
    for x, y in ((-0.15, -0.09), (0.17, -0.02)):  # pouches: a bag and a flap with a stud
        b.loft([(Vector((x, y - 0.025, z - 0.02)), 0.032, 0.026), (Vector((x * 1.04, y - 0.035, z - 0.07)), 0.036, 0.03),
                (Vector((x * 1.04, y - 0.03, z - 0.1)), 0.03, 0.024)], leather, TORSO, n=10, power=2.6, side=(1, 0, 0))
        b.box((x, y - 0.058, z - 0.03), (0.07, 0.012, 0.04), dark, TORSO, rot=(8, 0, 0))
        b.ellipsoid((x, y - 0.066, z - 0.045), (0.008, 0.005, 0.008), gold, TORSO, 6, 4)
    # A scroll case behind the right hip, slanting.
    b.rod((-0.17, 0.08, z + 0.04), (-0.2, 0.1, z - 0.2), 0.022, dark, TORSO, 8)
    b.ellipsoid((-0.17, 0.08, z + 0.04), (0.025, 0.025, 0.012), gold, TORSO, 8, 4)
    if book:  # a small book on a strap: covers, pages, gold corners
        c = Vector((0.2, 0.05, z - 0.13))
        b.box(c, (0.035, 0.1, 0.13), b.mat("Cover", (110, 40, 40)), TORSO, rot=(0, 0, 10))
        b.box(c + Vector((0.0, -0.002, 0)), (0.026, 0.104, 0.12), b.mat("Pages", (235, 225, 195)), TORSO, rot=(0, 0, 10))
        for dy in (-0.045, 0.045):
            for dz in (-0.058, 0.058):
                b.ellipsoid(c + Vector((0.018, dy, dz)), (0.008, 0.012, 0.012), gold, TORSO, 6, 4)
        b.rod(c + Vector((0, 0, 0.065)), Vector((0.19, 0.03, z - 0.01)), 0.006, leather, TORSO, 5)


def skirt_power(p):
    """How firmly a long skirt follows the legs at a point (see Body.weigh): soft above the
    knees, firm toward the hem."""
    return 3.0 + 5.0 * max(0.0, min(1.0, (0.65 - p.z) / 0.45))


def robe_body(name, p, book=True, rags=0.0, extra=None, panel=0.05):
    """A long robe: the bodice, a skirt to the ankles falling in folds, an embroidered hem (or
    a ragged one, see folded_skirt), a panel down the front (panel meters wide, none if 0) and
    a standing collar, and a belt with pouches. extra(b) adds more before the export."""
    b = Body()
    robe = b.mat("Robe", p.cloth)
    b.loft(torso_rings(b, 0.018, top=1.49, bottom=0.95), robe, TORSO, n=20, power=2.3)
    levels = [(0.99, 0.19, 0.134, 0.0, 0.01), (0.8, 0.214, 0.152, 0.006, 0.01), (0.55, 0.244, 0.176, 0.013, 0.015),
              (0.3, 0.27, 0.198, 0.019, 0.02), (0.07, 0.3, 0.222, 0.025, 0.025)]
    # The skirt follows the legs: softly near the hips, so it drapes when they bend, firmly
    # toward the hem, so the hem goes where the feet go in a long stride. (Robed outfits wear
    # the body without legs, Male_Body_Robed, so no leg shows pushing through.)
    b.power = skirt_power
    point = folded_skirt(b, levels, 9, robe, SKIRT, hem=None if rags else p.trim(b), phase=0.4, rags=rags)
    if panel:
        front_strip(b, point, 0.0, 4.0, panel, p.panel(b), SKIRT, widen=1.5)
    b.power = 6.0
    if panel:
        pts = [Vector((0, -0.142, 1.44)), Vector((0, -0.15, 1.3)), Vector((0, -0.148, 1.15)), Vector((0, -0.145, 1.0))]
        b.loft([(q, panel / 2, 0.006) for q in pts], p.panel(b), TORSO, n=8, power=3, side=(1, 0, 0))  # the panel up to the collar
        b.loft([((0, 0.0, 1.47), 0.12, 0.105), ((0, 0.005, 1.52), 0.088, 0.085), ((0, 0.01, 1.57), 0.09, 0.088)], p.trim(b), NECK + TORSO, n=20,
               power=2.2, caps=False)  # standing collar
    robe_belt(b, p, book=book)
    if extra:
        extra(b)
    b.export(name)


def robe_arms(name, p):
    """Bell sleeves: snug at the shoulder, widening down the forearm in folds to an embroidered
    cuff, with a lighter undersleeve showing at the wrist."""
    b = Body()
    robe, trim, under = b.mat("Robe", p.cloth), p.trim(b), b.mat("Under", p.under)
    for s, x in sides(b):
        bones = arm_bones(s)
        sh = b.head(f"upperarm_{s}")
        b.ellipsoid(sh + Vector((-x * 0.01, 0, 0.0)), (0.092, 0.087, 0.087), robe, bones)
        b.loft(limb(b, f"upperarm_{s}", f"upperarm_{s}", [(0.066, 0.066), (0.064, 0.064), (0.06, 0.06)], 0.016, 0.0, 1.0), robe, bones,
               n=12, side=(0, 0, 1))
        d, u, v = arm_frame(b, s)
        e0, e1 = b.head(f"lowerarm_{s}"), b.tail(f"lowerarm_{s}")
        n, folds = 30, 5
        rings = []
        for k in range(7):
            t = k / 6
            c = e0.lerp(e1, -0.05 + 0.95 * t)
            r = 0.062 + 0.055 * t ** 1.4
            f = 0.004 + 0.016 * t
            rings.append([c + (u * math.cos(2 * math.pi * j / n) + v * math.sin(2 * math.pi * j / n)) * (r + f * math.sin(folds * 2 * math.pi * j / n))
                          for j in range(n)])
        b.tube(rings, robe, bones)
        last = rings[-1]
        c = e0.lerp(e1, 0.9)
        b.tube([[c + (q - c) * 1.03 - d * 0.035 for q in last], [c + (q - c) * 1.03 + d * 0.004 for q in last]], trim, bones)  # the cuff
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.046, 0.042), (0.04, 0.036), (0.036, 0.032)], 0.006, 0.55, 1.0), under, bones,
               n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.041, 0.037), (0.041, 0.037)], 0.006, 0.97, 1.0), trim, bones, n=12,
               side=(0, 0, 1))
    b.export(name)


def long_cloak(name, p, hem=True, lining=None, collar=None):
    """A long cloak from the shoulders to the floor, around the back and sides, in folds that
    deepen toward the hem, the lining (a darker shade unless given) showing at its edges; with
    hem, an embroidered hem; with collar (a color), a tall stiff collar standing up behind the
    head, flaring out."""
    b = Body()
    cloak, lining = b.mat("Cloak", p.dark), b.mat("Lining", lining or tuple(int(c * 0.62) for c in p.dark))
    rows = []
    n_rows, n_cols = 18, 33
    for i in range(n_rows):
        t = i / (n_rows - 1)
        z = 1.5 - t * 1.42
        rx, ry = 0.22 + 0.21 * t, 0.17 + 0.18 * t
        row = []
        for j in range(n_cols):
            u = j / (n_cols - 1)
            a = math.pi * (-0.12 + 1.24 * u)  # round the back from the left side to the right
            fold = (0.004 + 0.045 * t) * math.sin(u * math.pi * 11 + 0.5 * t)
            row.append((math.cos(a) * (rx + fold), 0.02 + math.sin(a) * (ry + fold) + 0.05 * t * t, z))
        rows.append(row)
    # The thighs too, softly, so the robe under it (which follows the legs) stays inside it in a
    # wide stance.
    bones = ["spine_03", "spine_02", "spine_01", "pelvis", "clavicle_l", "clavicle_r", "thigh_l", "thigh_r"]
    b.power = 3.0
    b.sheet(rows, lining, cloak, bones, 0.012)  # the grid is the inner face; sheet offsets the outer one
    if hem:
        b.sheet([[Vector(q) + Vector((0, 0, 0.035 * k)) for q in rows[-1]] for k in (1, 0)], p.trim(b), p.trim(b), bones, 0.016)
    if collar:
        white = b.mat("Collar", collar)
        rows = []
        for i, (z, r) in enumerate(((1.5, 0.15), (1.58, 0.18), (1.66, 0.23), (1.72, 0.29))):
            rows.append([Vector((math.cos(a) * r, 0.03 + math.sin(a) * r * 0.85, z)) for a in
                         (math.pi * (-0.12 + 1.24 * j / 12) for j in range(13))])
        b.sheet(rows, white, white, NECK + ["spine_03"], 0.012)
    b.power = 6.0
    b.export(name)


def cowl_hood(name, p, beard=None):
    """A hood pulled up, open at the front so the face shows, its point falling back, and a cowl
    over the shoulders; with beard (a hair color), the wizard's beard and the hair beside the
    face, which the hood would otherwise hide."""
    b = Body()
    cloth, lining = b.mat("Hood", p.dark), b.mat("Lining", tuple(int(c * 0.55) for c in p.dark))
    rings = [(1.56, 0.118, 0.128, 0.0), (1.64, 0.126, 0.138, 0.0), (1.72, 0.124, 0.138, 0.0), (1.79, 0.11, 0.128, 0.01),
             (1.85, 0.08, 0.1, 0.03), (1.88, 0.04, 0.06, 0.06), (1.86, 0.012, 0.02, 0.12)]
    # Set back, so the face shows through the front and the hood frames it.
    b.loft([((0, y + 0.04, z), rx, rz) for z, rx, rz, y in rings], cloth, HEAD, n=22, power=2.2, side=(1, 0, 0))
    rim = []
    for j in range(17):
        a = math.pi * (0.02 + 0.96 * j / 16)  # round the face from one cheek over the brow to the other
        rim.append((Vector((math.cos(a) * 0.112, -0.09 - 0.012 * math.sin(a), 1.57 + math.sin(a) * 0.2)), 0.018, 0.026))
    b.loft(rim, cloth, HEAD, n=8, power=2.0, side=(0, 1, 0))
    b.loft([(Vector(q) + Vector((0, 0.012, 0)), rx * 0.8, rz) for q, rx, rz in rim], lining, HEAD, n=8, power=2.0, side=(0, 1, 0))
    b.loft([((0, 0.02, 1.6), 0.15, 0.145), ((0, 0.02, 1.52), 0.215, 0.158), ((0, 0.02, 1.44), 0.25, 0.168)], cloth, NECK + TORSO, n=22,
           power=2.3, caps=False)  # the cowl over the shoulders
    if beard:
        wizard_beard(b, beard)
    b.export(name)


WARLOCK = Robes("Warlock", (46, 42, 46), (150, 28, 32), (176, 40, 36), (70, 62, 66), leather=(140, 34, 32),
                gold=(120, 116, 122))  # red and black, after art/concept/warlock_s51.png
PRIEST = Robes("Priest", (238, 236, 230), (36, 34, 38), (238, 236, 230), (238, 236, 230), leather=(40, 36, 38), gold=(200, 204, 212),
               panel=(36, 34, 38))  # white and black
BONE = (226, 216, 190)


def warlock_wraps(b):
    """Strips of cloth wound round the body, and a second belt slung from the right hip with
    a strap hanging from it."""
    wrap = b.mat("Wrap", tuple(int(c * 0.82) for c in WARLOCK.cloth))
    leather, gold = b.mat("Leather", WARLOCK.leather), b.mat("Gold", WARLOCK.gold, metal=True)
    for i, z in enumerate((1.4, 1.3, 1.2, 1.12)):
        b.loft([((0, 0.0, z + 0.016), 0.17 + 0.01 * (i == 0), 0.136), ((0, 0.0, z - 0.016), 0.176 + 0.01 * (i == 0), 0.14)], wrap, TORSO,
               n=22, power=2.4, caps=False)
    b.loft([((0, 0.0, 0.95), 0.215, 0.16), ((0.0, 0.0, 0.915), 0.218, 0.163)], leather, TORSO, n=24, power=2.6, caps=False)
    b.ellipsoid((-0.11, -0.15, 0.93), (0.026, 0.01, 0.022), gold, TORSO, 10, 5)
    b.loft([(Vector((-0.11, -0.16, 0.92)), 0.014, 0.004), (Vector((-0.115, -0.17, 0.8)), 0.014, 0.004), (Vector((-0.11, -0.172, 0.7)), 0.012, 0.004)],
           leather, TORSO, n=6, power=3, side=(1, 0, 0))
    b.ellipsoid((-0.11, -0.176, 0.7), (0.012, 0.006, 0.012), gold, TORSO, 6, 4)


CULTIST = Robes("Cultist", (128, 122, 116), (126, 20, 28), (150, 120, 80), (100, 94, 90), leather=(92, 62, 40),
                gold=(150, 152, 160))  # art/concept/cultist_s51.png


def cultist_chains(b):
    """Chains hanging from the belt down the front, a ring at the end of each."""
    iron = b.mat("Chain", (120, 122, 130), metal=True)
    for x, length in ((-0.06, 0.34), (0.05, 0.26), (0.11, 0.3)):
        for k in range(int(length / 0.035)):
            q = Vector((x, -0.17 - 0.004 * k, 0.95 - 0.035 * k))
            b.ellipsoid(q, (0.01, 0.006, 0.016), iron, SKIRT, 6, 4, rot=(0, 90 * (k % 2), 0))
        b.ellipsoid(Vector((x, -0.18, 0.93 - length)), (0.022, 0.008, 0.022), iron, SKIRT, 8, 4)


def wrapped_arms(name, p):
    """Ragged sleeves to the elbow, forearms and wrists bound in strips of cloth."""
    b = Body()
    cloth, wrap = b.mat("Sleeve", p.cloth), b.mat("Wrap", tuple(min(255, int(c * 1.18)) for c in p.cloth))
    for s, x in sides(b):
        bones = arm_bones(s)
        sh = b.head(f"upperarm_{s}")
        b.ellipsoid(sh + Vector((-x * 0.01, 0, 0.0)), (0.09, 0.085, 0.085), cloth, bones)
        b.loft(limb(b, f"upperarm_{s}", f"upperarm_{s}", [(0.064, 0.064), (0.062, 0.062), (0.064, 0.066)], 0.016, 0.0, 1.0), cloth, bones,
               n=12, side=(0, 0, 1))
        el = b.head(f"lowerarm_{s}")
        for k in range(8):  # ragged ends at the elbow
            a = 2 * math.pi * k / 8
            q = el + Vector((0, math.cos(a) * 0.07, math.sin(a) * 0.07))
            b.rod(q - Vector((x * 0.03, 0, 0)), q + Vector((x * (0.03 + 0.02 * (k % 2)), 0, 0)), 0.02, cloth, bones, 4, taper=0.2)
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.048, 0.045), (0.046, 0.042), (0.04, 0.036)], 0.008, 0.1, 0.98), wrap, bones,
               n=12, side=(0, 0, 1))
        for k in range(8):  # the binding, wound slantwise
            t = 0.12 + 0.11 * k
            c = el.lerp(b.tail(f"lowerarm_{s}"), t)
            b.box(c, (0.014, 0.105 - 0.02 * t, 0.096 - 0.02 * t), cloth, bones, rot=(22, 0, 0))
    b.export(name)


def hair_short(name, color):
    """Short, neat hair: close on top, trimmed at the back and the sides."""
    b = Body()
    hair = b.mat("Hair", color)
    scalp(b, hair, 0.004)
    b.loft([((0, 0.02, 1.6), 0.08, 0.06), ((0, 0.04, 1.66), 0.096, 0.07), ((0, 0.03, 1.71), 0.098, 0.08)], hair, HEAD, n=16)  # back
    for s, x in sides(b):
        b.box((x * 0.092, -0.03, 1.665), (0.018, 0.04, 0.055), hair, HEAD)  # sideburns
    b.export(name)


def stole(name, p):
    """A priest's stole: a band around the back of the neck, its two ends hanging down the
    front of the robe to the knees, embroidered, with fringed ends."""
    b = Body()
    panel, fringe = p.panel(b), b.mat("Fringe", p.band)
    b.loft([((0, 0.01, 1.53), 0.1, 0.095), ((0, 0.015, 1.49), 0.135, 0.115)], panel, NECK + TORSO, n=22, power=2.3, caps=False)
    front = [(1.49, -0.11), (1.42, -0.138), (1.3, -0.158), (1.15, -0.152), (1.0, -0.158), (0.85, -0.178), (0.7, -0.19), (0.58, -0.2)]
    b.power = 3.0
    for x in (-1, 1):
        rows = []
        for z, y in front:
            c = 0.06 + 0.02 * min(1, (1.49 - z) / 0.2)  # from beside the neck out a little
            rows.append([Vector((x * (c + w), y - 0.006, z)) for w in (-0.032, 0, 0.032)])
        b.sheet(rows if x > 0 else [list(reversed(r)) for r in rows], panel, panel, SKIRT, 0.005)
        for k in range(5):
            q = Vector((x * (0.08 + (k - 2) * 0.014), -0.205, 0.58))
            b.rod(q, q + Vector((0, -0.004, -0.05)), 0.004, fringe, SKIRT, 4)
    b.power = 6.0
    b.export(name)


def deep_hood(name, p, skull=True):
    """A deep hood pulled up and forward, rising to a point that falls back, and in its shadow a
    bone mask like a skull: eye holes, a nose, teeth (or, without skull, a smooth blank mask with
    only narrow eye slits). A cowl over the shoulders."""
    b = Body()
    hood, lining = b.mat("Hood", p.dark), b.mat("Lining", tuple(int(c * 0.5) for c in p.dark))
    shadow, bone, socket = b.mat("Shadow", (14, 8, 16)), b.mat("Bone", BONE), b.mat("Socket", (24, 14, 26))
    rings = [(1.56, 0.122, 0.132, 0.0), (1.64, 0.13, 0.142, -0.005), (1.72, 0.128, 0.142, 0.0), (1.79, 0.114, 0.132, 0.01),
             (1.85, 0.082, 0.102, 0.03), (1.885, 0.042, 0.062, 0.06), (1.87, 0.012, 0.02, 0.12)]
    b.loft([((0, y + 0.012, z), rx, rz) for z, rx, rz, y in rings], hood, HEAD, n=22, power=2.2, side=(1, 0, 0))
    rim = []
    for j in range(17):
        a = math.pi * (0.02 + 0.96 * j / 16)  # round the face from one cheek over the brow to the other, drawn forward
        rim.append((Vector((math.cos(a) * 0.112, -0.138 - 0.03 * math.sin(a), 1.57 + math.sin(a) * 0.2)), 0.018, 0.026))
    b.loft(rim, hood, HEAD, n=8, power=2.0, side=(0, 1, 0))
    b.loft([(Vector(q) + Vector((0, 0.014, 0)), rx * 0.8, rz) for q, rx, rz in rim], lining, HEAD, n=8, power=2.0, side=(0, 1, 0))
    b.ellipsoid((0, -0.1, 1.66), (0.1, 0.05, 0.13), shadow, HEAD, 16, 8)
    # The mask: a brow and cheeks of bone narrowing to the jaw.
    b.loft([((0, -0.132, 1.72), 0.062, 0.022), ((0, -0.142, 1.67), 0.066, 0.026), ((0, -0.14, 1.62), 0.052, 0.024), ((0, -0.128, 1.575), 0.03, 0.018)],
           bone, HEAD, n=16, power=2.2, side=(1, 0, 0))
    if not skull:
        for x in (-0.026, 0.026):
            b.ellipsoid((x, -0.164, 1.685), (0.017, 0.005, 0.005), socket, HEAD, 8, 4)  # eye slits
        b.loft([(Vector((0, -0.15, 1.69)), 0.008, 0.006), (Vector((0, -0.16, 1.62)), 0.006, 0.006)], bone, HEAD, n=6)  # a ridge
    for x in (-0.026, 0.026) if skull else ():
        b.ellipsoid((x, -0.163, 1.68), (0.019, 0.008, 0.016), socket, HEAD, 10, 5)
        b.ellipsoid((x, -0.17, 1.68), (0.005, 0.003, 0.004), b.mat("Glint", (235, 150, 255)), HEAD, 6, 4)
    if skull:
        b.ellipsoid((0, -0.167, 1.645), (0.007, 0.005, 0.012), socket, HEAD, 6, 4)  # nose
        for k in range(5):  # teeth
            b.box(((k - 2) * 0.01, -0.158, 1.605), (0.003, 0.005, 0.016), socket, HEAD)
    b.loft([((0, 0.02, 1.6), 0.15, 0.145), ((0, 0.025, 1.52), 0.21, 0.16), ((0, 0.025, 1.44), 0.245, 0.17)], hood, NECK + TORSO, n=22,
           power=2.3, caps=False)  # the cowl over the shoulders
    b.export(name)


class Leathers:
    """The colors of a light armor set (the rogue_* pieces): the cloth of hood and tunic, the
    leather of belts, pouches and boots and its darker straps, the trousers, the sleeves, the
    cape, the steel of bracers and buckles, and if not the hood's, the tunic's color and a sash."""

    def __init__(self, cloth, cloth_dark, leather, strap, under, sleeve, cape, steel=(150, 154, 164), tunic=None, sash=None):
        self.cloth, self.cloth_dark, self.leather, self.strap, self.under, self.steel = cloth, cloth_dark, leather, strap, under, steel
        self.sleeve, self.cape, self.tunic, self.sash = sleeve, cape, tunic or cloth, sash

    def mats(self, b):
        """(leather, strap, buckle steel) materials."""
        return b.mat("Leather", self.leather), b.mat("Strap", self.strap), b.mat("Buckle", self.steel, metal=True)


BANDIT = Leathers((172, 112, 52), (70, 58, 50), (104, 68, 42), (64, 42, 30), (120, 116, 74), (150, 82, 46), (150, 82, 46),
                  tunic=(132, 128, 84))  # art/concept/bandit_s51.png
NOMAD = Leathers((168, 40, 36), (120, 88, 58), (126, 84, 50), (78, 50, 32), (68, 58, 56), (222, 206, 170), (168, 40, 36),
                 tunic=(222, 206, 170), sash=(168, 40, 36))  # art/concept/nomad_s51.png
ROGUE = Leathers((104, 108, 106), (72, 76, 76), (94, 82, 72), (58, 52, 48), (50, 50, 54), (138, 138, 144), (82, 82, 88))  # greys, after art/concept/rogue_s51.png


def strap(b, pts, width, mat, bones, thick=0.007):
    """A flat strap through points (lying on a surface, its width across)."""
    b.loft([(Vector(q), width / 2, thick) for q in pts], mat, bones, n=8, power=3, side=(0, 1, 0) if abs(pts[0][1]) > 0.05 else (1, 0, 0))


def buckle(b, c, mat, bones, size=0.022, rot=(0, 0, 0)):
    """A square buckle frame, seen from the front."""
    c = Vector(c)
    b.box(c, (size, 0.006, size), mat, bones, rot=rot)


def rogue_jerkin(name, p, rags=0.07, quilted=False, length=0.74):
    """A cloth tunic to mid-thigh (its hem at length) with a ragged hem (rags meters), laced at
    the chest, two belts slung at the hips with buckles and pouches, a red strip of cloth
    hanging from them (or a sash, if the colors have one), and a dagger sheathed across the
    back. quilted: stitched in vertical rows, like a gambeson."""
    b = Body()
    leather, dark, steel = p.mats(b)
    cloth, cloth_dark = b.mat("Tunic", p.tunic), b.mat("TunicDark", tuple(int(c * 0.72) for c in p.tunic))
    b.loft(torso_rings(b, 0.02, top=1.49, bottom=0.95), cloth, TORSO, n=20, power=2.3)
    mid = (0.99 + length) / 2
    levels = [(0.99, 0.19, 0.134, 0.0, 0.01), (mid, 0.205 + 0.03 * (0.74 - length), 0.146, 0.007, 0.01),
              (length, 0.224 + 0.08 * (0.74 - length), 0.16 + 0.05 * (0.74 - length), 0.012, 0.012)]
    b.power = 3.0
    folded_skirt(b, levels, 7, cloth, TORSO + (["thigh_l", "thigh_r"] if length < 0.7 else []), steps=2, rags=rags)
    b.power = 6.0
    if quilted:  # stitched rows: thin dark lines down the body
        for k in range(16):
            a = 2 * math.pi * (k + 0.5) / 16
            pts = [Vector((math.cos(a) * (rx + 0.004), 0.01 + f + math.sin(a) * (ry + 0.004), z)) for z, rx, ry, f in
                   ((1.42, 0.214, 0.124, -0.005), (1.3, 0.2, 0.135, -0.02), (1.15, 0.18, 0.125, -0.01), (1.02, 0.174, 0.12, -0.004))]
            b.loft([(q, 0.003, 0.003) for q in pts], cloth_dark, TORSO, n=4)
    b.loft([((0, -0.143, 1.45), 0.012, 0.006), ((0, -0.152, 1.3), 0.012, 0.006), ((0, -0.15, 1.2), 0.012, 0.006)], cloth_dark, TORSO, n=8,
           power=3, side=(1, 0, 0))  # the opening
    for z in (1.42, 1.37, 1.32, 1.27, 1.22):  # laces across it
        b.box((0, -0.158, z), (0.04, 0.006, 0.006), dark, TORSO, rot=(0, 25, 0))
        b.box((0, -0.158, z), (0.04, 0.006, 0.006), dark, TORSO, rot=(0, -25, 0))
    # Two belts slung at the hips, one tilted, buckles and pouches.
    ring_band(b, (0, 0.0, 0.99), 0.2, 0.15, 0.034, leather, TORSO, n=24, power=2.6)
    for i in range(4):
        b.box(((i - 1.5) * 0.06, -0.157, 0.99), (0.02, 0.008, 0.024), steel, TORSO)  # a row of buckles
    b.loft([((0, 0.0, 0.95), 0.212, 0.158), ((0, 0.0, 0.92), 0.214, 0.16)], dark, TORSO, n=24, power=2.6, caps=False)
    buckle(b, (0.1, -0.15, 0.935), steel, TORSO, 0.024)
    for x in (-0.16, 0.18):
        b.box((x, -0.1, 0.92), (0.05, 0.035, 0.065), leather, TORSO)
        b.box((x, -0.12, 0.945), (0.054, 0.012, 0.03), dark, TORSO)
        b.ellipsoid((x, -0.128, 0.94), (0.007, 0.004, 0.007), steel, TORSO, 6, 4)
    if p.sash:  # a sash wound above the belts, its end hanging at the left hip
        sash = b.mat("Sash", p.sash)
        ring_band(b, (0, 0.0, 1.05), 0.192, 0.145, 0.06, sash, TORSO, n=24, power=2.5)
        b.loft([(Vector((0.11, -0.14, 1.04)), 0.03, 0.006), (Vector((0.12, -0.16, 0.85)), 0.034, 0.006), (Vector((0.115, -0.17, 0.68)), 0.03, 0.006)],
               sash, TORSO + ["thigh_l"], n=6, power=3, side=(1, 0, 0))
    else:
        b.loft([(Vector((0.1, -0.155, 0.95)), 0.02, 0.004), (Vector((0.11, -0.17, 0.75)), 0.024, 0.004), (Vector((0.105, -0.175, 0.55)), 0.02, 0.004)],
               b.mat("Red", (160, 40, 36)), TORSO + ["thigh_l"], n=6, power=3, side=(1, 0, 0))  # the red strip
    # The dagger across the small of the back: a sheath and a hilt.
    b.box((0.02, 0.17, 1.03), (0.26, 0.022, 0.045), dark, TORSO, rot=(0, -10, 0))
    b.rod((-0.12, 0.17, 1.05), (-0.2, 0.17, 1.06), 0.012, leather, TORSO, 6)
    b.box((-0.115, 0.17, 1.05), (0.012, 0.03, 0.06), steel, TORSO)
    b.export(name)


def rogue_arms(name, p):
    """Cloth sleeves, steel bracers strapped over them, fingerless gloves."""
    b = Body()
    leather, dark, steel = p.mats(b)
    under = b.mat("Sleeve", p.sleeve)
    for s, x in sides(b):
        bones = arm_bones(s)
        sh = b.head(f"upperarm_{s}")
        b.ellipsoid(sh + Vector((-x * 0.01, 0, 0.0)), (0.088, 0.083, 0.083), under, bones)
        b.loft(limb(b, f"upperarm_{s}", f"upperarm_{s}", UPPER_ARM[:3] + [(0.056, 0.054)], 0.012, 0.0, 0.95), under, bones, n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", FOREARM[:3], 0.008, 0.0, 0.4), under, bones, n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.052, 0.049), (0.05, 0.046), (0.045, 0.041)], 0.01, 0.3, 0.95), steel, bones,
               n=14, power=2.4, side=(0, 0, 1))  # bracer
        ring_band_along(b, f"lowerarm_{s}", 0.31, 0.056, steel, bones, 0.012)  # its rolled edges
        ring_band_along(b, f"lowerarm_{s}", 0.94, 0.05, steel, bones, 0.012)
        for t in (0.45, 0.75):
            ring_band_along(b, f"lowerarm_{s}", t, 0.057 - 0.006 * t, dark, bones, 0.014)
            c = b.head(f"lowerarm_{s}").lerp(b.tail(f"lowerarm_{s}"), t)
            b.box(c + Vector((0, -0.056 + 0.006 * t, 0)), (0.016, 0.006, 0.018), steel, bones)
        b.ellipsoid(b.head(f"lowerarm_{s}"), (0.05, 0.05, 0.05), dark, bones, 10, 6)  # elbow pad
        # Fingerless gloves: the palm and the first joints in leather.
        wrist, knuckles = b.head(f"hand_{s}"), b.head(f"middle_01_{s}")
        b.loft([(wrist, 0.035, 0.026), (wrist.lerp(knuckles, 0.5), 0.047, 0.024), (knuckles, 0.048, 0.02)], dark, [f"hand_{s}", f"lowerarm_{s}"],
               n=12, power=2.6, side=(0, 1, 0))
    b.export(name)


def rogue_legs(name, p):
    """Close trousers, leather knee guards, a strap around the right thigh holding a knife."""
    b = Body()
    leather, dark, steel = p.mats(b)
    under = b.mat("Under", p.under)
    b.loft(torso_rings(b, 0.01, top=1.0), under, TORSO, n=20, power=2.3)
    for s, x in sides(b):
        bones = leg_bones(s)
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", THIGH, 0.012, 0.0, 1.0), under, bones, n=14, side=(1, 0, 0))
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", CALF[:4], 0.01, 0.0, 0.75), under, bones, n=14, side=(1, 0, 0))
        kn = b.head(f"calf_{s}") + Vector((0, -0.035, 0))
        b.ellipsoid(kn, (0.06, 0.04, 0.07), leather, bones, 12, 7)
        ring_band(b, kn + Vector((0, 0.03, 0.05)), 0.07, 0.068, 0.012, dark, bones, n=14)
        ring_band(b, kn + Vector((0, 0.03, -0.05)), 0.066, 0.066, 0.012, dark, bones, n=14)
    t0, t1 = b.head("thigh_r"), b.tail("thigh_r")  # a knife strapped to the thigh
    for t in (0.35, 0.6):
        ring_band(b, t0.lerp(t1, t), 0.1 - 0.02 * t, 0.1 - 0.02 * t, 0.016, dark, leg_bones("r"), n=14)
    c = t0.lerp(t1, 0.48) + Vector((-0.095, -0.02, 0))
    b.box(c, (0.016, 0.035, 0.15), leather, leg_bones("r"))
    b.rod(c + Vector((0, 0, 0.075)), c + Vector((0, 0, 0.14)), 0.011, dark, leg_bones("r"), 6)
    b.export(name)


def rogue_boots(name, p):
    """Tall soft boots folded over at the top, buckled straps around the shin."""
    b = Body()
    leather, dark, steel = p.mats(b)
    for s, x in sides(b):
        bones = leg_bones(s)
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", [(0.066, 0.072), (0.058, 0.062), (0.05, 0.054), (0.047, 0.05)], 0.008, 0.25, 0.96),
               leather, bones, n=14, side=(1, 0, 0))
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", [(0.08, 0.086), (0.076, 0.082)], 0.0, 0.2, 0.32), dark, bones, n=14, side=(1, 0, 0))
        for t in (0.5, 0.72):
            ring_band_along(b, f"calf_{s}", t, 0.064 - 0.012 * t, dark, bones, 0.016)
            c = b.head(f"calf_{s}").lerp(b.tail(f"calf_{s}"), t)
            b.box(c + Vector((x * (0.06 - 0.012 * t), 0, 0)), (0.008, 0.022, 0.022), steel, bones)
        foot(b, s, x, leather, 0.012, 0.02)
        b.loft([((x * 0.089, b.head(f"foot_{s}").y + 0.05, 0.01), 0.05, 0.01), ((x * 0.089, b.tail(f"ball_{s}").y + 0.005, 0.01), 0.048, 0.01)],
               dark, bones, n=12, power=3, side=(1, 0, 0))
    b.export(name)


def rogue_hood(name, p):
    """A hood pulled up over a cloth mask that covers the nose and mouth, leaving the eyes."""
    b = Body()
    cloth, dark = b.mat("Hood", p.cloth), b.mat("Mask", p.cloth_dark)
    rings = [(1.56, 0.116, 0.126, 0.0), (1.64, 0.124, 0.136, 0.0), (1.72, 0.122, 0.136, 0.0), (1.79, 0.108, 0.126, 0.01),
             (1.84, 0.076, 0.096, 0.025), (1.865, 0.03, 0.05, 0.05)]
    b.loft([((0, y + 0.045, z), rx, rz) for z, rx, rz, y in rings], cloth, HEAD, n=22, power=2.2, side=(1, 0, 0))
    rim = []
    for j in range(17):
        a = math.pi * (0.02 + 0.96 * j / 16)
        rim.append((Vector((math.cos(a) * 0.11, -0.085 - 0.01 * math.sin(a), 1.58 + math.sin(a) * 0.19)), 0.016, 0.022))
    b.loft(rim, cloth, HEAD, n=8, power=2.0, side=(0, 1, 0))
    # The mask: from the cheeks over the nose and down under the chin into the hood.
    b.loft([((0, -0.035, 1.66), 0.1, 0.098), ((0, -0.04, 1.615), 0.102, 0.1), ((0, -0.03, 1.565), 0.09, 0.094), ((0, -0.01, 1.53), 0.08, 0.085)],
           dark, HEAD, n=22, power=2.2, side=(1, 0, 0))
    b.loft([((0, 0.02, 1.6), 0.15, 0.145), ((0, 0.02, 1.56), 0.18, 0.15), ((0, 0.02, 1.52), 0.2, 0.156)], cloth, NECK + TORSO, n=22,
           power=2.3, caps=False)  # a short cowl, so the cape shows over the shoulders
    b.export(name)


def long_coat(name, color):
    """A long coat open in front, as a shoulder piece."""
    b = Body()
    open_coat(b, color, top=1.47, length=1.15)
    b.export(name)


def rogue_cape(name, p):
    """A cape to the knees lined in the tunic's color, its upper part a shorter layer over the
    shoulders, ragged at the hems."""
    b = Body()
    cape, lining = b.mat("Cape", p.cape), b.mat("Lining", p.cloth_dark)
    bones = ["spine_03", "spine_02", "spine_01", "pelvis", "clavicle_l", "clavicle_r"]

    def layer(z0, length, rx0, grow, span, folds, cols):
        rows = []
        for i in range(10):
            t = i / 9
            rx, ry = rx0 + grow * t, rx0 * 0.76 + grow * 0.9 * t
            row = []
            for j in range(cols):
                u = j / (cols - 1)
                a = math.pi * (span[0] + (span[1] - span[0]) * u)
                fold = (0.003 + 0.03 * t) * math.sin(u * math.pi * folds + 0.5 * t)
                rag = 0.05 * t ** 4 * math.sin(u * math.pi * 17) ** 2
                row.append((math.cos(a) * (rx + fold), 0.02 + math.sin(a) * (ry + fold) + 0.04 * t, z0 - length * t - rag))
            rows.append(row)
        return rows

    b.power = 3.0
    b.sheet(layer(1.47, 0.95, 0.22, 0.14, (-0.1, 1.1), 9, 25), lining, cape, bones, 0.01)  # the grid is the inner face
    b.sheet(layer(1.55, 0.3, 0.17, 0.15, (-0.35, 1.35), 7, 29), lining, cape, bones + ["upperarm_l", "upperarm_r"], 0.01)
    b.power = 6.0
    b.export(name)


HIDE = (152, 112, 74)
HIDE_DARK = (108, 76, 50)
FUR = (150, 120, 86)
FUR_LIGHT = (200, 178, 140)
COAT = (164, 78, 48)  # art/concept/shaman_s51.png: a rust-red coat under the fur
PAINT = (52, 140, 150)
PAINT_RED = (170, 62, 46)


def fur_ring(b, center, rx, ry, count, length, mats, bones, droop=0.7, r=0.024, span=(0.0, 2 * math.pi)):
    """A ring of shaggy tufts hanging outward and down from an ellipse, alternating materials."""
    c = Vector(center)
    for i in range(count):
        a = span[0] + (span[1] - span[0]) * (i + 0.5) / count
        out = Vector((math.cos(a), math.sin(a), 0))
        base = c + Vector((out.x * rx, out.y * ry, 0))
        l = length * (0.75 + 0.25 * math.sin(i * 2.3))
        tip = base + out * l * (1 - droop) + Vector((0, 0, -l * droop))
        b.rod(base, tip, r, mats[i % len(mats)], bones, 5, taper=0.15)


def tooth_necklace(b, bones):
    """A cord around the neck strung with bone beads and teeth, a medallion in the middle."""
    bone, cord = b.mat("Bone", BONE), b.mat("Cord", LEATHER_DARK)
    pts = []
    for j in range(13):
        a = math.pi * (0.1 + 0.8 * j / 12)
        pts.append(Vector((math.cos(a) * 0.12, -0.03 - math.sin(a) * 0.115, 1.47 - math.sin(a) * 0.1)))
    b.loft([(q, 0.004, 0.004) for q in pts], cord, bones, n=5)
    for j, q in enumerate(pts[1:-1], 1):
        if j % 2:
            b.ellipsoid(q, (0.01, 0.01, 0.01), bone, bones, 6, 4)
        elif j != 6:
            b.rod(q, q + Vector((0, -0.01, -0.04)), 0.009, bone, bones, 5, taper=0.1)
    # The medallion in the middle: a bronze disc with a red stone.
    m = pts[6] + Vector((0, -0.012, -0.04))
    b.rod(pts[6], m, 0.004, cord, bones, 4)
    b.ellipsoid(m, (0.034, 0.01, 0.034), b.mat("Bronze", (176, 120, 60), metal=True), bones, 14, 6)
    b.ellipsoid(m + Vector((0, -0.01, 0)), (0.014, 0.006, 0.014), b.mat("Stone", (180, 40, 36)), bones, 10, 5)


def open_coat(b, color, top=1.4, length=1.05, lining=None, vent=False, hem=None):
    """A long coat from the shoulders, open in front, falling in folds to the calves, ragged at
    the hem; it follows the thighs a little, so walking doesn't push the legs through it. The
    lining is a darker shade of the coat unless given.

    With vent, the coat is split up the back from the hem to the waist and each half follows
    its own leg, softly, so in a wide or crouching stance the tails part and swing with the legs
    instead of the whole back tilting with the hips like a board; its folds are deeper and more,
    so the shading breaks up its surface. hem (a material) adds a band along the hem."""
    rows = []
    depth, flare, folds = (0.06, 0.13, 15) if vent else (0.028, 0.1, 11)
    for i in range(14):
        t = i / 13
        rx, ry = 0.23 + flare * t, 0.17 + 0.08 * t
        row = []
        for j in range(27):
            u = j / 26
            a = math.pi * (-0.3 + 1.6 * u)  # round the back from the front left to the front right
            fold = (0.003 + depth * t) * math.sin(u * math.pi * folds + 0.4 * t)
            rag = 0.06 * t ** 4 * math.sin(u * math.pi * 19) ** 2
            row.append((math.cos(a) * (rx + fold), 0.01 + math.sin(a) * (ry + fold), top - length * t - rag))
        rows.append(row)
    lining, coat = b.mat("CoatLining", lining or tuple(int(c * 0.6) for c in color)), b.mat("Coat", color)
    power, b.power = b.power, 3.0
    upper = ["spine_03", "spine_02", "spine_01", "pelvis", "clavicle_l", "clavicle_r"]
    if not vent:
        b.sheet(rows, lining, coat, upper + ["thigh_l", "thigh_r"], 0.012)  # the grid is the inner face
        b.power = power
        return
    waist = next(i for i, row in enumerate(rows) if row[0][2] < 1.0)  # the first row below the waist
    b.sheet(rows[:waist + 1], lining, coat, upper, 0.012)
    # The tails: firm at the waist, where they meet the top, softer below, so they bend in a
    # curve; columns 0..13 go round one side to the back, 13..26 the other.
    b.power = lambda q: 3.0 if q.z > 0.95 else 2.2
    middle = len(rows[0]) // 2
    for cols in (slice(0, middle + 1), slice(middle, None)):
        tail = [row[cols] for row in rows[waist:]]
        side = "l" if (sum(q[0] for q in tail[-1]) > 0) == (b.head("thigh_l").x > 0) else "r"
        bones = ["pelvis", "spine_01", f"thigh_{side}"]
        b.sheet(tail, lining, coat, bones, 0.012)
        if hem:  # a band just above the ragged hem, on the outside
            band = [[Vector(q) * 1.012 + Vector((0, 0, dz)) for q in tail[-2]] for dz in (0.05, -0.01)]
            b.sheet(band, hem, hem, bones, 0.006)
    b.power = power


def shaman_tunic(name):
    """A hide tunic to mid-thigh with a painted band across the chest and down the front, a
    fringed hem, a belt with a bone buckle, pouches and a hanging charm, and a tooth necklace."""
    b = Body()
    hide, dark = b.mat("Hide", HIDE), b.mat("HideDark", HIDE_DARK)
    paint = b.textured("Paint", embroidery_png("Shaman", PAINT_RED, PAINT), 0.07)
    bone = b.mat("Bone", BONE)
    b.loft(torso_rings(b, 0.018, top=1.49, bottom=0.95), hide, TORSO, n=20, power=2.3)
    levels = [(0.99, 0.19, 0.134, 0.0, 0.01), (0.86, 0.205, 0.145, 0.006, 0.01), (0.72, 0.222, 0.158, 0.01, 0.012)]
    point = folded_skirt(b, levels, 7, hide, TORSO, steps=2)
    for j in range(36):  # the fringe: strips hanging from the hem
        a = 2 * math.pi * (j + 0.5) / 36
        q = point(2.0, a)
        b.rod(q + Vector((0, 0, 0.01)), q + Vector((math.cos(a) * 0.006, math.sin(a) * 0.006, -0.07 - 0.02 * (j % 2))), 0.007, dark,
              TORSO, 4, taper=0.6)
    ring_band(b, (0, 0.0, 1.33), 0.2, 0.15, 0.05, paint, TORSO, n=24, power=2.4)  # the painted band
    front_strip(b, point, 0.0, 2.0, 0.06, paint, TORSO)
    b.loft([((0, -0.14, 1.31), 0.03, 0.006), ((0, -0.148, 1.15), 0.03, 0.006), ((0, -0.145, 1.0), 0.03, 0.006)], paint, TORSO, n=8,
           power=3, side=(1, 0, 0))
    ring_band(b, (0, 0.0, 0.99), 0.198, 0.146, 0.04, dark, TORSO, n=24, power=2.5)
    b.ellipsoid((0, -0.152, 0.99), (0.03, 0.012, 0.024), bone, TORSO, 12, 6)  # a carved bone buckle
    for x in (-0.15, 0.17):
        b.loft([(Vector((x, -0.1, 0.97)), 0.03, 0.025), (Vector((x * 1.04, -0.11, 0.91)), 0.036, 0.03), (Vector((x * 1.04, -0.1, 0.87)), 0.026, 0.02)],
               dark, TORSO, n=10, power=2.2, side=(1, 0, 0))
    # A charm on a thong at the left hip: a ring of bone and two feathers.
    c = Vector((0.12, -0.15, 0.88))
    b.rod(Vector((0.12, -0.155, 0.97)), c, 0.004, dark, TORSO, 4)
    b.ellipsoid(c, (0.022, 0.008, 0.022), bone, TORSO, 10, 5)
    for dx, col in ((-0.012, PAINT_RED), (0.012, (235, 230, 220))):
        b.loft([(c + Vector((dx, 0, -0.015)), 0.008, 0.002), (c + Vector((dx * 1.5, -0.005, -0.06)), 0.012, 0.002),
                (c + Vector((dx * 2, -0.01, -0.1)), 0.002, 0.002)], b.mat(f"Feather{col[0]}", col), TORSO, n=6, power=2, side=(1, 0, 0))
    tooth_necklace(b, NECK + TORSO)
    b.export(name)


def shaman_mantle(name, fur=FUR, light=FUR_LIGHT, coat=COAT, feathers=True, claws=False):
    """A great fur mantle over the shoulders and down the back, shaggy at its edge, closed with
    a bone clasp; with feathers, feathers hanging from the shoulders; with coat (a color), over
    a long coat open in front; with claws, a bear's paws hanging over the chest (a pelt)."""
    b = Body()
    fur, light, bone = b.mat("Fur", fur), b.mat("FurLight", light), b.mat("Bone", BONE)
    levels = [(1.57, 0.1, 0.095, 0.0, 0.015), (1.5, 0.19, 0.145, 0.006, 0.015), (1.42, 0.26, 0.175, 0.012, 0.018),
              (1.33, 0.29, 0.195, 0.016, 0.02)]
    bones = TORSO + ["upperarm_l", "upperarm_r"]
    b.power = 3.0
    folded_skirt(b, levels, 10, fur, bones, rags=0.05)
    fur_ring(b, (0, 0.02, 1.33), 0.28, 0.19, 34, 0.08, [fur, light], bones, droop=0.85, r=0.026)
    fur_ring(b, (0, 0.02, 1.43), 0.25, 0.17, 26, 0.07, [light, fur], bones, droop=0.8, r=0.024)
    fur_ring(b, (0, 0.015, 1.535), 0.15, 0.125, 18, 0.06, [fur, light], NECK + TORSO, droop=0.5, r=0.022)  # the collar, standing
    # Down the back: a hide hanging to the hips, its fur edge shaggy.
    rows = []
    for i in range(8):
        t = i / 7
        z = 1.42 - 0.5 * t
        row = []
        for j in range(11):
            u = j / 10
            a = math.pi * (0.2 + 0.6 * u)
            w = 0.24 - 0.06 * t
            row.append((math.cos(a) * w * 1.1, 0.03 + math.sin(a) * (0.17 + 0.02 * t), z - 0.05 * t * math.sin(u * math.pi * 5) ** 2))
        rows.append(row)
    b.sheet(rows, b.mat("Hide", HIDE_DARK), fur, ["spine_03", "spine_02", "spine_01", "clavicle_l", "clavicle_r"], 0.016)
    b.power = 6.0
    if coat:
        paint = b.textured("Paint", embroidery_png("Shaman", PAINT_RED, PAINT), 0.07)
        open_coat(b, coat, length=0.95, vent=True, hem=paint)  # under the fur, to the knees
    b.ellipsoid((0, -0.12, 1.49), (0.034, 0.014, 0.02), bone, NECK + TORSO, 10, 5)  # the clasp: a bone toggle
    b.rod((-0.045, -0.125, 1.49), (0.045, -0.125, 1.49), 0.008, bone, NECK + TORSO, 6)
    if claws:  # the pelt's forepaws, hanging down the chest, claws pale
        for x in (-1, 1):
            top = Vector((x * 0.13, -0.12, 1.45))
            b.loft([(top, 0.045, 0.025), (top + Vector((x * 0.01, -0.03, -0.12)), 0.05, 0.03), (top + Vector((x * 0.01, -0.04, -0.2)), 0.045, 0.03)],
                   fur, TORSO, n=10, power=2.2, side=(1, 0, 0))
            for c in range(4):
                q = top + Vector((x * 0.01 + (c - 1.5) * 0.02, -0.06, -0.22))
                b.rod(q, q + Vector((0, -0.01, -0.035)), 0.007, bone, TORSO, 4, taper=0.1)
    for s, x in sides(b) if feathers else ():  # feathers on cords from the shoulders
        sh = b.head(f"upperarm_{s}") + Vector((x * 0.03, -0.06, 0.0))
        for k, col in enumerate(((235, 230, 220), PAINT_RED, (60, 50, 45))):
            q = sh + Vector((x * 0.02 * k, -0.01 * k, -0.02 * k))
            b.loft([(q, 0.008, 0.002), (q + Vector((0, -0.005, -0.05)), 0.014, 0.002), (q + Vector((0, -0.01, -0.11)), 0.002, 0.002)],
                   b.mat(f"Feather{k}", col), arm_bones(s) + TORSO, n=6, power=2, side=(1, 0, 0))
    b.export(name)


def shaman_arms(name):
    """Bare arms with a bone-bead band above each elbow, forearms wrapped in hide strips, fur at
    the wrist."""
    b = Body()
    hide, dark, fur, bone = b.mat("Hide", HIDE), b.mat("HideDark", HIDE_DARK), b.mat("Fur", FUR_LIGHT), b.mat("Bone", BONE)
    paint = b.mat("Paint", PAINT)
    for s, x in sides(b):
        bones = arm_bones(s)
        ring_band_along(b, f"upperarm_{s}", 0.35, 0.06, paint, bones, 0.014)  # painted stripes
        ring_band_along(b, f"upperarm_{s}", 0.45, 0.058, paint, bones, 0.01)
        ring_band_along(b, f"upperarm_{s}", 0.75, 0.054, dark, bones, 0.012)
        for k in range(9):
            a = 2 * math.pi * k / 9
            c = b.head(f"upperarm_{s}").lerp(b.tail(f"upperarm_{s}"), 0.75)
            b.ellipsoid(c + Vector((0, math.cos(a) * 0.056, math.sin(a) * 0.056)), (0.009, 0.009, 0.009), bone, bones, 6, 4)
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.05, 0.047), (0.048, 0.044), (0.042, 0.038)], 0.008, 0.25, 0.95), hide, bones,
               n=12, side=(0, 0, 1))
        for k in range(6):  # the wrapping, wound slantwise
            t = 0.3 + 0.11 * k
            c = b.head(f"lowerarm_{s}").lerp(b.tail(f"lowerarm_{s}"), t)
            b.box(c, (0.012, 0.11 - 0.02 * t, 0.1 - 0.02 * t), dark, bones, rot=(25, 0, 0))
        c = b.head(f"lowerarm_{s}").lerp(b.tail(f"lowerarm_{s}"), 0.96)
        b.loft([(c - Vector((x * 0.02, 0, 0)), 0.05, 0.048), (c, 0.056, 0.054), (c + Vector((x * 0.02, 0, 0)), 0.048, 0.046)], fur, bones, n=12,
               side=(0, 0, 1))
    b.export(name)


def shaman_legs(name, cloth=HIDE_DARK):
    """Hide (or cloth) trousers with leather thongs crossed up the shins."""
    b = Body()
    hide, dark = b.mat("Hide", cloth), b.mat("Thong", LEATHER_DARK)
    b.loft(torso_rings(b, 0.01, top=1.0), hide, TORSO, n=20, power=2.3)
    for s, x in sides(b):
        bones = leg_bones(s)
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", THIGH, 0.014, 0.0, 1.0), hide, bones, n=14, side=(1, 0, 0))
        b.ellipsoid(b.head(f"calf_{s}") + Vector((0, -0.01, 0)), (0.07, 0.072, 0.072), hide, bones)
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", CALF[:4], 0.012, 0.0, 0.75), hide, bones, n=14, side=(1, 0, 0))
        for k in range(4):
            c = b.head(f"calf_{s}").lerp(b.tail(f"calf_{s}"), 0.12 + 0.12 * k)
            for sgn in (-1, 1):
                b.box(c, (0.13, 0.14, 0.01), dark, bones, rot=(sgn * 22, 0, 0))
    b.export(name)


def hair_feathers(name):
    """Feathers tied into the hair at one side, on a beaded cord."""
    b = Body()
    bone = b.mat("Bone", BONE)
    q = Vector((-0.1, 0.02, 1.68))
    for k, col in enumerate(((235, 230, 220), PAINT_RED, (70, 58, 50))):
        top = q + Vector((-0.01 * k, 0.015 * k, 0.0))
        b.loft([(top, 0.009, 0.002), (top + Vector((-0.01, 0.01, -0.06)), 0.016, 0.002), (top + Vector((-0.02, 0.02, -0.13)), 0.002, 0.002)],
               b.mat(f"Feather{k}", col), HEAD, n=6, power=2, side=(0, 1, 0))
    for k in range(3):
        b.ellipsoid(q + Vector((0, 0, 0.02 + 0.016 * k)), (0.009, 0.009, 0.009), bone, HEAD, 6, 4)
    b.export(name)


IRON = (112, 116, 124)  # art/concept/berserker_s51.png
RED_BEARD = (176, 82, 40)


def berserker_head(name):
    """Wild red hair falling down the back, a great beard to the chest (the wizard's, in red),
    a few braids."""
    b = Body()
    hair, dark = b.mat("Hair", RED_BEARD), b.mat("HairDark", tuple(int(c * 0.78) for c in RED_BEARD))
    scalp(b, hair, 0.008)
    b.loft([((0, 0.03, 1.74), 0.11, 0.09), ((0, 0.07, 1.62), 0.12, 0.07), ((0, 0.09, 1.5), 0.13, 0.05), ((0, 0.1, 1.4), 0.1, 0.03)],
           hair, NECK + ["spine_02"], n=16, power=2.4)  # the mane down the back
    for x, y in ((-0.05, -0.06), (0.02, -0.08), (0.07, -0.03), (-0.08, 0.0), (0.0, 0.02), (0.06, 0.05)):  # unruly tufts on top
        b.rod((x, y, 1.79), (x * 1.6, y * 1.4 + 0.02, 1.86), 0.025, hair if x > 0 else dark, HEAD, 5, taper=0.2)
    wizard_beard(b, RED_BEARD)
    for x in (-0.1, 0.1):
        for k in range(4):  # braids beside the face
            b.ellipsoid((x, -0.03, 1.6 - 0.04 * k), (0.016, 0.016, 0.022), dark if k % 2 else hair, HEAD + ["spine_03"], 8, 5)
    b.export(name)


def berserker_harness(name):
    """Bare-chested: leather straps crossed over the chest through an iron ring, a broad belt
    with an iron buckle, a green wool kilt to the knees with a red sash hanging from the belt,
    a knife at the hip."""
    b = Body()
    leather, dark = b.mat("Leather", LEATHER), b.mat("LeatherDark", LEATHER_DARK)
    iron, kilt, red = b.mat("Iron", IRON, metal=True), b.mat("Kilt", (96, 112, 58)), b.mat("Sash", (164, 40, 34))
    for x in (-1, 1):  # straps from each shoulder across to the other hip, front and back
        for y, sgn in ((-0.135, 1), (0.11, -1)):
            strap(b, [Vector((x * 0.12, y, 1.46)), Vector((0, y - 0.02 * sgn, 1.26)), Vector((-x * 0.14, y + 0.005 * sgn, 1.04))], 0.045, leather,
                  TORSO, 0.008)
    b.ellipsoid((0, -0.162, 1.26), (0.032, 0.01, 0.032), iron, TORSO, 12, 5)
    b.ellipsoid((0, -0.168, 1.26), (0.018, 0.008, 0.018), dark, TORSO, 10, 4)
    ring_band(b, (0, 0.0, 1.0), 0.2, 0.15, 0.07, dark, TORSO, n=24, power=2.5)
    b.box((0, -0.157, 1.0), (0.06, 0.012, 0.05), iron, TORSO)
    levels = [(0.98, 0.2, 0.145, 0.0, 0.01), (0.84, 0.22, 0.16, 0.008, 0.01), (0.7, 0.236, 0.172, 0.012, 0.012)]
    b.power = 3.0
    folded_skirt(b, levels, 8, kilt, TORSO + ["thigh_l", "thigh_r"], steps=2, rags=0.06)
    b.loft([(Vector((-0.1, -0.15, 0.97)), 0.03, 0.006), (Vector((-0.11, -0.17, 0.75)), 0.034, 0.006), (Vector((-0.105, -0.175, 0.5)), 0.028, 0.006)],
           red, TORSO + ["thigh_r"], n=6, power=3, side=(1, 0, 0))  # the sash
    b.power = 6.0
    b.box((0.17, -0.06, 0.88), (0.02, 0.035, 0.16), dark, TORSO)  # a knife's sheath, and its hilt
    b.rod((0.17, -0.06, 0.96), (0.17, -0.06, 1.04), 0.012, leather, TORSO, 6)
    b.export(name)


def berserker_arms(name):
    """Bare arms with iron rings above the elbows and leather bracers, fur at the wrists."""
    b = Body()
    leather, iron, fur = b.mat("Leather", LEATHER_DARK), b.mat("Iron", IRON, metal=True), b.mat("Fur", (150, 122, 90))
    for s, x in sides(b):
        bones = arm_bones(s)
        for t in (0.55, 0.68):
            ring_band_along(b, f"upperarm_{s}", t, 0.06 - 0.004 * t, iron, bones, 0.018)
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.052, 0.049), (0.05, 0.046), (0.045, 0.041)], 0.01, 0.3, 0.92), leather, bones,
               n=14, power=2.4, side=(0, 0, 1))
        for t in (0.45, 0.75):
            ring_band_along(b, f"lowerarm_{s}", t, 0.057 - 0.006 * t, iron, bones, 0.01)
        c = b.head(f"lowerarm_{s}").lerp(b.tail(f"lowerarm_{s}"), 0.95)
        b.loft([(c - Vector((x * 0.02, 0, 0)), 0.052, 0.05), (c, 0.058, 0.056), (c + Vector((x * 0.02, 0, 0)), 0.05, 0.048)], fur, bones, n=12,
               side=(0, 0, 1))
    b.export(name)


CORSAIR_RED = (176, 44, 40)
CORSAIR_TEAL = (44, 118, 128)
BRASS = (196, 152, 78)


VAMPIRE_RED = (168, 28, 34)


def vampire_coat(name):
    """A red brocade waistcoat with gold buttons over a white shirt and cravat, under a long
    cream coat open in front."""
    b = Body()
    red, gold, white = b.mat("Waistcoat", VAMPIRE_RED), b.mat("Gold", BRASS, metal=True), b.mat("Shirt", (236, 232, 224))
    b.loft(torso_rings(b, 0.022, top=1.49, bottom=0.9), red, TORSO, n=20, power=2.3)
    for z in (1.36, 1.28, 1.2, 1.12, 1.04):
        b.ellipsoid((0, -0.158, z), (0.012, 0.008, 0.012), gold, TORSO, 6, 4)
    for x in (-1, 1):  # brocade: thin gold stripes down the waistcoat
        b.loft([(Vector((x * 0.06, -0.152, 1.4)), 0.004, 0.004), (Vector((x * 0.07, -0.155, 1.0)), 0.004, 0.004)], gold, TORSO, n=4)
    b.loft([((0, 0.0, 1.5), 0.09, 0.09), ((0, -0.01, 1.45), 0.12, 0.11)], white, NECK + TORSO, n=18, power=2.2, caps=False)
    b.ellipsoid((0, -0.13, 1.43), (0.05, 0.02, 0.05), white, TORSO, 10, 5)  # the cravat
    b.ellipsoid((0, -0.135, 1.37), (0.035, 0.018, 0.05), white, TORSO, 10, 5)
    open_coat(b, (220, 212, 192), top=1.47, length=1.1)
    b.export(name)


def tricorn(name, hair_color):
    """A tricorn hat: a low crown and a brim turned up into three corners (front and the two
    back sides), edged in gold, over long hair tied back."""
    b = Body()
    felt, gold = b.mat("Felt", (128, 98, 60)), b.mat("Gold", BRASS, metal=True)
    hair = b.mat("Hair", hair_color)
    scalp(b, hair, 0.004)
    b.loft([((0, 0.04, 1.7), 0.1, 0.07), ((0, 0.08, 1.6), 0.08, 0.05), ((0, 0.1, 1.5), 0.05, 0.03)], hair, NECK + ["spine_03"], n=12)
    b.loft([((0, -0.005, 1.765), 0.112, 0.122), ((0, -0.005, 1.83), 0.104, 0.114), ((0, -0.005, 1.88), 0.085, 0.092),
            ((0, -0.005, 1.9), 0.03, 0.03)], felt, HEAD, n=22, power=2.2, side=(1, 0, 0))  # the crown
    corners = [math.radians(a) for a in (-90, 30, 150)]  # front, back left, back right (around from +X)
    edge, inner = [], []
    for i in range(3):
        a0, a1 = corners[i], corners[(i + 1) % 3] + (2 * math.pi if i == 2 else 0)
        p0 = Vector((math.cos(a0) * 0.23, -0.005 + math.sin(a0) * 0.25, 1.82))
        p1 = Vector((math.cos(a1) * 0.23, -0.005 + math.sin(a1) * 0.25, 1.82))
        for j in range(8):  # a straight edge from corner to corner, turned up highest midway
            t = j / 8
            q = p0.lerp(p1, t)
            q.z += 0.05 * math.sin(math.pi * t)
            edge.append(q)
            a = a0 + (a1 - a0) * t
            inner.append(Vector((math.cos(a) * 0.11, -0.005 + math.sin(a) * 0.12, 1.77)))
    edge.append(edge[0])
    inner.append(inner[0])
    b.sheet([inner, edge], felt, felt, HEAD, 0.012)
    b.loft([(q, 0.008, 0.008) for q in edge], gold, HEAD, n=6)  # the gold edging
    b.export(name)


def corsair_shirt(name):
    """A loose cream shirt with a red scarf knotted at the neck, a waistcoat in the coat's teal,
    a bandolier across the chest, a broad belt with a brass buckle and a pistol tucked in it."""
    b = Body()
    shirt, red, teal = b.mat("Shirt", (226, 214, 186)), b.mat("Scarf", CORSAIR_RED), b.mat("Waistcoat", CORSAIR_TEAL)
    leather, dark, brass = b.mat("Leather", (112, 72, 44)), b.mat("LeatherDark", LEATHER_DARK), b.mat("Brass", BRASS, metal=True)
    b.loft(torso_rings(b, 0.02, top=1.49, bottom=0.86), shirt, TORSO, n=20, power=2.3)
    for x in (-1, 1):  # the waistcoat's front panels, open over the shirt
        rows = [[Vector((x * w, -0.148 - 0.004 * (1.4 - z), z)) for w in (0.05, 0.11, 0.17)] for z in (1.42, 1.3, 1.15, 1.02)]
        b.sheet(rows if x > 0 else [list(reversed(r)) for r in rows], teal, teal, TORSO, 0.006)
    b.loft([((0, 0.0, 1.5), 0.1, 0.095), ((0, -0.005, 1.46), 0.13, 0.115)], red, NECK + TORSO, n=20, power=2.2, caps=False)  # scarf
    b.ellipsoid((0.02, -0.13, 1.44), (0.04, 0.025, 0.03), red, TORSO, 10, 5)
    b.loft([(Vector((0.02, -0.14, 1.42)), 0.03, 0.006), (Vector((0.04, -0.155, 1.33)), 0.035, 0.006)], red, TORSO, n=6, power=3,
           side=(1, 0, 0))
    a, c = Vector((-0.13, -0.14, 1.44)), Vector((0.16, -0.14, 1.0))
    strap(b, [a, a.lerp(c, 0.5) + Vector((0, -0.03, 0)), c], 0.045, leather, TORSO, 0.008)  # bandolier
    ring_band(b, (0, 0.0, 0.99), 0.2, 0.15, 0.06, dark, TORSO, n=24, power=2.6)
    b.box((0, -0.157, 0.99), (0.07, 0.014, 0.055), brass, TORSO)
    b.box((0, -0.165, 0.99), (0.04, 0.006, 0.03), dark, TORSO)
    b.box((-0.1, -0.16, 1.0), (0.03, 0.04, 0.16), b.mat("Wood", (90, 60, 40)), TORSO, rot=(0, 30, 0))  # the pistol's grip
    b.rod((-0.07, -0.165, 0.94), (0.0, -0.165, 0.86), 0.014, b.mat("Iron", (90, 92, 100), metal=True), TORSO, 6)  # and its barrel
    b.export(name)


def corsair_coat(name):
    """A long red coat to the calves, open in front and lined in teal."""
    b = Body()
    open_coat(b, CORSAIR_RED, top=1.47, length=1.12, lining=CORSAIR_TEAL)
    b.export(name)


def coat_arms(name, sleeve=CORSAIR_RED, cuff=CORSAIR_TEAL, trim=BRASS, shirt=(226, 214, 186), glove=(110, 72, 44)):
    """A coat's sleeves with big turned-back cuffs edged in metal, shirt ruffles at the wrists
    and gloves (the corsair's: red, teal cuffs, brass, leather gloves)."""
    b = Body()
    red, teal, brass = b.mat("Sleeve", sleeve), b.mat("Cuff", cuff), b.mat("Brass", trim, metal=True)
    shirt, glove = b.mat("Shirt", shirt), b.mat("Glove", glove)
    for s, x in sides(b):
        bones = arm_bones(s)
        sh = b.head(f"upperarm_{s}")
        b.ellipsoid(sh + Vector((-x * 0.01, 0, 0.0)), (0.095, 0.09, 0.09), red, bones)
        b.loft(limb(b, f"upperarm_{s}", f"upperarm_{s}", [(0.068, 0.068), (0.064, 0.064), (0.062, 0.062)], 0.016, 0.0, 1.0), red, bones,
               n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.06, 0.058), (0.056, 0.052), (0.05, 0.046)], 0.012, 0.0, 0.62), red, bones,
               n=12, side=(0, 0, 1))
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.07, 0.068), (0.08, 0.078)], 0.0, 0.58, 0.86), teal, bones, n=14,
               side=(0, 0, 1), caps=False)  # the turned-back cuff
        ring_band_along(b, f"lowerarm_{s}", 0.86, 0.082, brass, bones, 0.012)
        ring_band_along(b, f"lowerarm_{s}", 0.6, 0.071, brass, bones, 0.01)
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.05, 0.048), (0.062, 0.058)], 0.0, 0.86, 0.98), shirt, bones, n=12,
               side=(0, 0, 1), caps=False)  # ruffles
        wrist, knuckles = b.head(f"hand_{s}"), b.head(f"middle_01_{s}")
        b.loft([(wrist, 0.036, 0.027), (wrist.lerp(knuckles, 0.5), 0.047, 0.024), (knuckles, 0.048, 0.02)], glove, [f"hand_{s}", f"lowerarm_{s}"],
               n=12, power=2.6, side=(0, 1, 0))
    b.export(name)


def breeches(name, color):
    """Plain trousers, tucked into boots."""
    b = Body()
    cloth = b.mat("Trousers", color)
    b.loft(torso_rings(b, 0.01, top=1.0), cloth, TORSO, n=20, power=2.3)
    for s, x in sides(b):
        b.loft(limb(b, f"thigh_{s}", f"thigh_{s}", THIGH, 0.016, 0.0, 1.0), cloth, leg_bones(s), n=14, side=(1, 0, 0))
        b.ellipsoid(b.head(f"calf_{s}") + Vector((0, -0.01, 0)), (0.07, 0.072, 0.072), cloth, leg_bones(s))
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", CALF[:4], 0.012, 0.0, 0.6), cloth, leg_bones(s), n=14, side=(1, 0, 0))
    b.export(name)


def corinthian_helm(name, p=BRONZE):
    """A Corinthian helm: a bronze bowl closed over the face by cheek guards, leaving a T of
    dark openings for the eyes and the mouth, a nose guard down the middle; a tall crest of
    red horsehair from front to back on a bronze ridge."""
    b = Body()
    bronze, edge = p.mats(b)
    dark = b.mat("Slit", (20, 16, 14))
    red, white = b.mat("Crest", p.cloth), b.mat("CrestLight", (236, 226, 206))
    rings = [(1.52, 0.108, 0.126), (1.58, 0.11, 0.13), (1.66, 0.112, 0.13), (1.73, 0.108, 0.124), (1.79, 0.096, 0.11),
             (1.84, 0.074, 0.086), (1.87, 0.042, 0.05), (1.88, 0.008, 0.01)]
    b.loft([((0, -0.008, z), rx, ry) for z, rx, ry in rings], bronze, HEAD, n=24, power=2.2, side=(1, 0, 0))
    ring_band(b, (0, -0.008, 1.525), 0.112, 0.13, 0.012, edge, HEAD + ["spine_03"], n=24, power=2.2)
    ring_band(b, (0, -0.008, 1.735), 0.11, 0.127, 0.012, edge, HEAD, n=24, power=2.2)
    for x in (-1, 1):  # the eye openings, swept back, and the mouth slit under the nose guard
        b.ellipsoid((x * 0.045, -0.13, 1.69), (0.032, 0.012, 0.014), dark, HEAD, 10, 5, rot=(0, x * 10, x * -20))
    b.box((0, -0.136, 1.6), (0.026, 0.012, 0.09), dark, HEAD)
    b.box((0, -0.141, 1.66), (0.018, 0.012, 0.07), bronze, HEAD)  # the nose guard
    # The crest: a thick fan of horsehair from the brow over to the back, white at its ends.
    base, top = [], []
    for i in range(11):
        t = i / 10
        q = Vector((0, -0.12 + 0.26 * t, 1.79 + 0.11 * math.sin(math.pi * t)))  # sitting on the helm's curve
        base.append(q)
        top.append(q + Vector((0, 0.35 * (t - 0.3), 1)).normalized() * (0.15 + 0.06 * math.sin(math.pi * t)))
    b.sheet([base, top], red, red, HEAD, 0.04)
    for q, tq in ((base[0], top[0]), (base[-1], top[-1])):
        b.rod(q, tq, 0.022, white, HEAD, 6, taper=0.6)
    b.export(name)


def muscle_cuirass(name, p=BRONZE):
    """A bronze cuirass, smooth and broad (no carved chest: it read as a bust), its belly
    marked by a few raised lines; a red belt; a skirt of white leather strips (pteruges) tipped
    in bronze, two layers to mid-thigh."""
    b = Body()
    bronze, edge = p.mats(b)
    belt, strip, tip = b.mat("Belt", p.cloth), b.mat("Strip", (232, 222, 196)), b.mat("Tip", p.plate, metal=True)
    b.loft(torso_rings(b, 0.03, top=1.48, bottom=0.95), bronze, TORSO, n=22, power=2.5)
    for z in (1.22, 1.15, 1.08):  # lines across the belly
        ring_band(b, (0, -0.005, z), 0.186, 0.138, 0.006, edge, TORSO, n=22, power=2.5)
    b.loft([(Vector((0, -0.16, 1.42)), 0.005, 0.006), (Vector((0, -0.166, 1.25)), 0.005, 0.006)], edge, TORSO, n=6, side=(1, 0, 0))
    ring_band(b, (0, 0.0, 1.47), 0.15, 0.12, 0.02, edge, NECK + TORSO, n=22, power=2.3)  # the rolled neck edge
    ring_band(b, (0, 0.0, 0.98), 0.2, 0.15, 0.05, belt, TORSO, n=24, power=2.5)
    b.power = 3.0
    for layer, (z, length, r) in enumerate(((0.95, 0.15, 0.205), (0.86, 0.17, 0.215))):
        for k in range(18):
            a = 2 * math.pi * (k + 0.5 * layer) / 18
            q = Vector((math.cos(a) * r, math.sin(a) * r * 0.75, z))
            out = Vector((math.cos(a), math.sin(a) * 0.75, 0)) * 0.03
            b.box(q + out * 0.5 - Vector((0, 0, length / 2)), (0.05, 0.012, length), strip, TORSO, rot=(0, 0, math.degrees(a) + 90))
            b.box(q + out * 0.8 - Vector((0, 0, length + 0.01)), (0.052, 0.014, 0.025), tip, TORSO, rot=(0, 0, math.degrees(a) + 90))
    b.power = 6.0
    b.export(name)


def hoplite_arms(name, p=BRONZE):
    """Bare arms with bronze bracers and a leather wrap at the wrist."""
    b = Body()
    bronze, edge = p.mats(b)
    wrap = b.mat("Wrap", (120, 84, 52))
    for s, x in sides(b):
        bones = arm_bones(s)
        b.loft(limb(b, f"lowerarm_{s}", f"lowerarm_{s}", [(0.052, 0.049), (0.05, 0.046), (0.045, 0.041)], 0.01, 0.3, 0.9), bronze, bones,
               n=14, power=2.4, side=(0, 0, 1))
        ring_band_along(b, f"lowerarm_{s}", 0.31, 0.057, edge, bones, 0.01)
        ring_band_along(b, f"lowerarm_{s}", 0.95, 0.047, wrap, bones, 0.03)
    b.export(name)


def greaves(name, p=BRONZE):
    """Bronze greaves from below the knee to the ankle, over bare legs, and a short linen
    loincloth under the strip skirt."""
    b = Body()
    bronze, edge = p.mats(b)
    b.loft(torso_rings(b, 0.008, top=0.98), b.mat("Linen", (226, 216, 190)), TORSO, n=20, power=2.3)
    for s, x in sides(b):
        bones = leg_bones(s)
        b.ellipsoid(b.head(f"calf_{s}") + Vector((0, -0.035, 0)), (0.06, 0.04, 0.068), bronze, bones, 12, 7)  # over the knee
        b.loft(limb(b, f"calf_{s}", f"calf_{s}", [(0.064, 0.07), (0.064, 0.074), (0.054, 0.06), (0.046, 0.05)], 0.012, 0.1, 0.9), bronze,
               bones, n=16, power=2.4, side=(1, 0, 0))
        ring_band_along(b, f"calf_{s}", 0.9, 0.06, edge, bones, 0.012)
    b.export(name)


BROWN_HAIR = (112, 74, 46)
PIECES = {
    "Male_Body": lambda n: male_body(n),
    "Male_Body_Robed": lambda n: male_body(n, legs=False),
    "Hair_Long": lambda n: hair_long(n, BROWN_HAIR),
    "Beard_Long": lambda n: beard(n, BROWN_HAIR, full=True),
    "Shoes": lambda n: shoes(n, (80, 55, 40)),  # under the casters' robes
    "Boots_Fur": lambda n: boots(n, (110, 75, 48), (75, 50, 32), fur=(150, 125, 95)),  # the shaman's
    # The bandit: an ochre hood and mask, a long rust coat over a quilted tunic, baggy trousers.
    "Bandit_Hood": lambda n: rogue_hood(n, BANDIT),
    "Bandit_Tunic": lambda n: rogue_jerkin(n, BANDIT, rags=0.0, quilted=True, length=0.8),
    "Bandit_Arms": lambda n: rogue_arms(n, BANDIT),
    "Bandit_Legs": lambda n: rogue_legs(n, BANDIT),
    "Bandit_Boots": lambda n: rogue_boots(n, BANDIT),
    "Bandit_Coat": lambda n: long_coat(n, BANDIT.cape),
    # The nomad raider: a red hood and cape over a cream tunic with a red sash, baggy trousers.
    "Nomad_Hood": lambda n: rogue_hood(n, NOMAD),
    "Nomad_Tunic": lambda n: rogue_jerkin(n, NOMAD, rags=0.0, length=0.62),
    "Nomad_Arms": lambda n: rogue_arms(n, NOMAD),
    "Nomad_Legs": lambda n: rogue_legs(n, NOMAD),
    "Nomad_Boots": lambda n: rogue_boots(n, NOMAD),
    "Nomad_Cape": lambda n: rogue_cape(n, NOMAD),
    # The knight (art/concept/knight_s51_*.png).
    "Knight_Helm": knight_helm,
    "Knight_Cuirass": knight_cuirass,
    "Knight_Arms": knight_arms,
    "Knight_Legs": knight_legs,
    "Knight_Sabatons": knight_sabatons,
    "Knight_Mantle": knight_mantle,
    # The paladin: the same plate in bright steel and gold, a great helm, a white tabard with a sun.
    "Paladin_Helm": great_helm,
    "Paladin_Cuirass": lambda n: knight_cuirass(n, PALADIN, emblem=True),
    "Paladin_Arms": lambda n: knight_arms(n, PALADIN),
    "Paladin_Legs": lambda n: knight_legs(n, PALADIN),
    "Paladin_Sabatons": lambda n: knight_sabatons(n, PALADIN),
    "Paladin_Mantle": lambda n: knight_mantle(n, PALADIN, scarf=False),
    # The mage: a blue robe and hooded cloak with gold embroidery, bell sleeves.
    "Mage_Robe": lambda n: robe_body(n, MAGE),
    "Mage_Sleeves": lambda n: robe_arms(n, MAGE),
    "Mage_Cloak": lambda n: long_cloak(n, MAGE),
    "Mage_Hood": lambda n: cowl_hood(n, MAGE, beard=(232, 230, 224)),
    # The warlock: a red cloak and deep hood over a skull mask, ragged black wrappings, red belts.
    "Warlock_Robe": lambda n: robe_body(n, WARLOCK, book=False, rags=0.1, extra=warlock_wraps, panel=0),
    "Warlock_Sleeves": lambda n: wrapped_arms(n, WARLOCK),
    "Warlock_Cloak": lambda n: long_cloak(n, WARLOCK, hem=False),
    "Warlock_Hood": lambda n: deep_hood(n, WARLOCK),
    # The priest: a white robe, a black stole embroidered in white, short neat hair.
    "Priest_Robe": lambda n: robe_body(n, PRIEST, panel=0),
    "Priest_Sleeves": lambda n: robe_arms(n, PRIEST),
    "Priest_Stole": lambda n: stole(n, PRIEST),
    "Hair_Short": lambda n: hair_short(n, (84, 58, 40)),
    # The rogue: dark leather with straps and knives, a green hood over a masked face.
    "Rogue_Jerkin": lambda n: rogue_jerkin(n, ROGUE),
    "Rogue_Arms": lambda n: rogue_arms(n, ROGUE),
    "Rogue_Legs": lambda n: rogue_legs(n, ROGUE),
    "Rogue_Boots": lambda n: rogue_boots(n, ROGUE),
    "Rogue_Hood": lambda n: rogue_hood(n, ROGUE),
    "Rogue_Cape": lambda n: rogue_cape(n, ROGUE),
    # The shaman: painted hide, a great fur mantle, bones, teeth and feathers.
    "Shaman_Tunic": shaman_tunic,
    "Shaman_Mantle": shaman_mantle,
    "Shaman_Arms": shaman_arms,
    "Shaman_Legs": shaman_legs,
    "Hair_Feathers": hair_feathers,
    # The berserker (an enemy of the North): bare-chested under a bear pelt, wild red hair and beard.
    "Berserker_Head": berserker_head,
    "Berserker_Harness": berserker_harness,
    "Berserker_Arms": berserker_arms,
    "Berserker_Legs": lambda n: shaman_legs(n, (92, 96, 112)),
    # The corsair (an enemy of the Mediterranean): a tricorn, a long red coat lined in teal.
    "Corsair_Hat": lambda n: tricorn(n, (150, 60, 36)),
    "Corsair_Shirt": corsair_shirt,
    "Corsair_Coat": corsair_coat,
    "Corsair_Arms": coat_arms,
    "Corsair_Legs": lambda n: breeches(n, (110, 114, 72)),
    "Corsair_Boots": lambda n: rogue_boots(n, Leathers(CORSAIR_RED, CORSAIR_TEAL, (150, 110, 62), (96, 66, 40), (110, 114, 72),
                                                       CORSAIR_RED, CORSAIR_RED)),
    # The hoplite (an enemy of the Mediterranean): bronze, a red crest and cloak.
    "Hoplite_Helm": corinthian_helm,
    "Hoplite_Cuirass": muscle_cuirass,
    "Hoplite_Arms": hoplite_arms,
    "Hoplite_Legs": greaves,
    "Hoplite_Feet": lambda n: knight_sabatons(n, BRONZE),
    "Hoplite_Mantle": lambda n: knight_mantle(n, BRONZE, scarf=False),
    # The vampire (an enemy of the unholy lands): pale, a tall collar, a red cape lined in white.
    "Male_Body_Pale": lambda n: male_body(n, skin=(214, 204, 208)),
    "Hair_Long_Black": lambda n: hair_long(n, (30, 26, 30)),
    "Vampire_Coat": vampire_coat,
    "Vampire_Arms": lambda n: coat_arms(n, sleeve=(50, 52, 48), cuff=(236, 232, 224), trim=(190, 194, 204), shirt=(236, 232, 224),
                                        glove=(214, 204, 208)),
    "Vampire_Legs": lambda n: breeches(n, (120, 96, 70)),
    "Vampire_Boots": lambda n: rogue_boots(n, Leathers(VAMPIRE_RED, (40, 36, 40), (36, 32, 36), (24, 22, 26), (120, 96, 70),
                                                       VAMPIRE_RED, VAMPIRE_RED)),
    "Vampire_Cape": lambda n: long_cloak(n, Robes("Vampire", VAMPIRE_RED, VAMPIRE_RED, BRASS, VAMPIRE_RED), hem=False,
                                         lining=(232, 228, 222), collar=(236, 232, 226)),
    # The cultist (an enemy of the unholy lands): a crimson hooded cloak over a grey robe, a blank mask, chains.
    "Cultist_Hood": lambda n: deep_hood(n, CULTIST, skull=False),
    "Cultist_Robe": lambda n: robe_body(n, CULTIST, book=False, rags=0.08, extra=cultist_chains, panel=0),
    "Cultist_Arms": lambda n: wrapped_arms(n, CULTIST),
    "Cultist_Cloak": lambda n: long_cloak(n, CULTIST, hem=False),
    "Berserker_Pelt": lambda n: shaman_mantle(n, (78, 58, 42), (118, 90, 66), coat=None, feathers=False, claws=True),
}
for name, build in PIECES.items():
    if not only or name in only:
        build(name)
