"""Helpers for building rigged, animated creatures (scarab.py, snake.py, mummy.py, scorpion.py,
guardian.py, ember.py, beasts.py, goblin.py) and held props (weapons.py) from simple shapes in
Blender. Parts are
bound rigidly to one bone each, unless a script assigns its own weights. Bones point straight
up from their pivot unless given a tail, so in a bone's own space Y is up, X is world X and Z
is world -Y (backward); creatures face -Y (glTF +Z).
"""

import math
import os
from pathlib import Path

import bpy
from mathutils import Vector


def write_png(path, rows):
    """Writes rows of (r, g, b) pixels as an 8-bit RGB PNG (no imaging library in Blender's Python)."""
    import struct
    import zlib
    raw = b"".join(b"\x00" + bytes(c for p in r for c in p) for r in rows)
    chunk = lambda t, d: struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d))
    png = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", len(rows[0]), len(rows), 8, 2, 0, 0, 0))
    png += chunk(b"IDAT", zlib.compress(raw, 9)) + chunk(b"IEND", b"")
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(png)
    return path


def textured_material(name, png, scale):
    """A material showing an image, tiled every scale meters on parts mapped with box_uv."""
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    bsdf = m.node_tree.nodes["Principled BSDF"]
    bsdf.inputs["Roughness"].default_value = 0.6
    tex = m.node_tree.nodes.new("ShaderNodeTexImage")
    img = bpy.data.images.load(str(png))
    img.name = Path(png).stem
    tex.image = img
    m.node_tree.links.new(tex.outputs["Color"], bsdf.inputs["Base Color"])
    m["tile"] = scale
    return m


def box_uv(obj, size):
    """Box-maps UVs: each face takes the two world axes across its normal, so a texture tiles at
    its size (meters) whatever the shape."""
    me = obj.data
    uv = me.uv_layers.get("UVMap") or me.uv_layers.new(name="UVMap")
    mw = obj.matrix_world
    for poly in me.polygons:
        n = mw.to_3x3() @ poly.normal
        ax = max(range(3), key=lambda i: abs(n[i]))
        u_i, v_i = [(1, 2), (0, 2), (0, 1)][ax]
        for li in poly.loop_indices:
            co = mw @ me.vertices[me.loops[li].vertex_index].co
            uv.data[li].uv = (co[u_i] / size, co[v_i] / size)


class Kit:
    def __init__(self):
        bpy.ops.wm.read_factory_settings(use_empty=True)
        bpy.context.scene.render.fps = 30
        self.parts = []  # (object, bone or None when the object has its own vertex groups)
        self.arm = None

    # ---- shapes ----

    def material(self, name, rgb, metal=False):
        """metal: the game shades it as steel (see finish)."""
        m = bpy.data.materials.new(name)
        if metal:
            m["metal"] = True
        m.diffuse_color = (*[c / 255 for c in rgb], 1)
        m.use_nodes = True
        bsdf = m.node_tree.nodes["Principled BSDF"]
        bsdf.inputs["Base Color"].default_value = m.diffuse_color
        bsdf.inputs["Roughness"].default_value = 0.6
        return m

    def textured(self, name, png, scale):
        """A material showing an image (see textured_material); parts get box-mapped UVs."""
        return textured_material(name, png, scale)

    def finish(self, obj, mat, bone, smooth=True):
        if "tile" in mat:
            box_uv(obj, mat["tile"])
        obj.data.materials.append(mat)
        # A white vertex color on every part; metal's alpha is 0.5, which the game's toon shader
        # shades as steel. All parts need it, or joining fills it in black.
        col = obj.data.color_attributes.new("Color", "FLOAT_COLOR", "POINT")
        for d in col.data:
            d.color = (1.0, 1.0, 1.0, 0.5 if mat.get("metal") else 1.0)
        if smooth:
            for p in obj.data.polygons:
                p.use_smooth = True
        self.parts.append((obj, bone))
        return obj

    def ellipsoid(self, center, radii, mat, bone, segments=18, rings=10, rot=(0, 0, 0)):
        bpy.ops.mesh.primitive_uv_sphere_add(segments=segments, ring_count=rings, radius=1, location=center,
                                             rotation=[math.radians(a) for a in rot])
        o = bpy.context.object
        o.scale = radii
        return self.finish(o, mat, bone)

    def rod(self, a, b, radius, mat, bone, verts=8, taper=1.0):
        """A cylinder (or cone, with taper < 1) from point a to point b."""
        a, b = Vector(a), Vector(b)
        d = b - a
        bpy.ops.mesh.primitive_cone_add(vertices=verts, radius1=radius, radius2=radius * taper, depth=d.length,
                                        location=(a + b) / 2)
        o = bpy.context.object
        o.rotation_mode = "QUATERNION"
        o.rotation_quaternion = Vector((0, 0, 1)).rotation_difference(d.normalized())
        return self.finish(o, mat, bone, smooth=False)

    def box(self, center, size, mat, bone, rot=(0, 0, 0)):
        bpy.ops.mesh.primitive_cube_add(size=1, location=center, rotation=[math.radians(a) for a in rot])
        o = bpy.context.object
        o.scale = size
        return self.finish(o, mat, bone, smooth=False)

    def mesh(self, name, verts, faces, mat, bone, smooth=True):
        me = bpy.data.meshes.new(name)
        me.from_pydata([tuple(v) for v in verts], [], faces)
        me.validate()
        o = bpy.data.objects.new(name, me)
        bpy.context.scene.collection.objects.link(o)
        return self.finish(o, mat, bone, smooth=smooth)

    def loft(self, name, rings, mat, bone, n=14, power=2.4, keel=1.0, side=(1, 0, 0)):
        """A smooth tube through cross-sections: rings is a list of (center, half width, half
        height), following the path. power > 2 makes the sections boxier than ellipses; keel < 1
        narrows the underside (a deep chest). Both ends are closed."""
        side = Vector(side)
        pts = [Vector(c) for c, _, _ in rings]
        verts, faces = [], []
        for i, (c, rx, rz) in enumerate(rings):
            t = (pts[min(i + 1, len(pts) - 1)] - pts[max(i - 1, 0)]).normalized()
            up = side.cross(t).normalized()
            sd = t.cross(up).normalized()
            for j in range(n):
                a = 2 * math.pi * j / n
                ca, sa = math.cos(a), math.sin(a)
                x = math.copysign(abs(ca) ** (2 / power), ca) * rx
                z = math.copysign(abs(sa) ** (2 / power), sa) * rz
                if z < 0:
                    x *= keel + (1 - keel) * (1 + z / rz)
                verts.append(pts[i] + sd * x + up * z)
        m = len(rings)
        for i in range(m - 1):
            for j in range(n):
                a, b = i * n + j, i * n + (j + 1) % n
                faces.append((a, b, b + n, a + n))
        verts.append(pts[0])
        verts.append(pts[-1])
        s, e = len(verts) - 2, len(verts) - 1
        for j in range(n):
            faces.append((s, (j + 1) % n, j))
            faces.append((e, (m - 1) * n + j, (m - 1) * n + (j + 1) % n))
        return self.mesh(name, verts, faces, mat, bone)

    def shard(self, base, direction, width, length, mat, bone, flat=(0, 0, 1)):
        """A flat, pointed clump of fur (or a bristle): a thin wedge from base along direction."""
        d = Vector(direction).normalized()
        w = Vector(flat).cross(d)
        if w.length < 0.1:
            w = Vector((1, 0, 0)).cross(d)
        w = w.normalized() * width
        n = d.cross(w).normalized() * width * 0.35
        b = Vector(base)
        tip = b + d * length
        verts = [b - w, b + n, b + w, b - n, tip]
        faces = [(0, 1, 4), (1, 2, 4), (2, 3, 4), (3, 0, 4), (0, 3, 2, 1)]
        return self.mesh("shard", verts, faces, mat, bone, smooth=False)

    # ---- rig ----

    def bones(self, specs):
        """specs: (name, head, parent or None[, tail]); bones without a tail point up."""
        bpy.ops.object.armature_add(location=(0, 0, 0))
        self.arm = bpy.context.object
        bpy.ops.object.mode_set(mode="EDIT")
        eb = self.arm.data.edit_bones
        eb.remove(eb[0])
        for spec in specs:
            name, head, parent = spec[:3]
            b = eb.new(name)
            b.head = head
            b.tail = spec[3] if len(spec) > 3 else Vector(head) + Vector((0, 0, 0.15))
            b.roll = 0
            if parent:
                b.parent = eb[parent]
        bpy.ops.object.mode_set(mode="OBJECT")
        for p in self.arm.pose.bones:
            p.rotation_mode = "XYZ"
        self.arm.animation_data_create()

    def bind(self, name):
        """Joins all parts into one mesh deformed by the armature."""
        for obj, bone in self.parts:
            if bone:
                vg = obj.vertex_groups.new(name=bone)
                vg.add(range(len(obj.data.vertices)), 1.0, "REPLACE")
        bpy.ops.object.select_all(action="DESELECT")
        for obj, _ in self.parts:
            obj.select_set(True)
        bpy.context.view_layer.objects.active = self.parts[0][0]
        bpy.ops.object.join()
        mesh = bpy.context.object
        mesh.name = name + "Mesh"
        bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
        mesh.parent = self.arm
        mesh.modifiers.new("Armature", "ARMATURE").object = self.arm
        self.arm.name = name

    def static(self, name):
        """Joins all parts into one mesh without a rig, for props that are held or placed."""
        bpy.ops.object.select_all(action="DESELECT")
        for obj, _ in self.parts:
            obj.select_set(True)
        bpy.context.view_layer.objects.active = self.parts[0][0]
        bpy.ops.object.join()
        mesh = bpy.context.object
        mesh.name = name
        bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)

    # ---- animation ----

    def key(self, frame, bone, rot=None, loc=None, scale=None):
        p = self.arm.pose.bones[bone]
        p.rotation_euler = [math.radians(a) for a in (rot or (0, 0, 0))]
        p.location = loc or (0, 0, 0)
        p.scale = (scale,) * 3 if isinstance(scale, (int, float)) else scale or (1, 1, 1)
        p.keyframe_insert("rotation_euler", frame=frame)
        p.keyframe_insert("location", frame=frame)
        p.keyframe_insert("scale", frame=frame)

    def action(self, name, frames, build):
        """Records an action: every bone keyed at rest on the first and last frame, then
        build() adds its keys. Each action becomes an NLA track, exported as an animation."""
        act = bpy.data.actions.new(name)
        act.use_fake_user = True
        self.arm.animation_data.action = act
        for p in self.arm.pose.bones:  # so actions don't leak poses into each other
            self.key(1, p.name)
            self.key(frames, p.name)
        build()
        track = self.arm.animation_data.nla_tracks.new()
        track.name = name
        track.strips.new(name, 1, act)
        self.arm.animation_data.action = None

    # ---- output ----

    def export(self, out):
        out = Path(out)
        out.parent.mkdir(parents=True, exist_ok=True)
        bpy.ops.object.select_all(action="SELECT")
        bpy.ops.export_scene.gltf(filepath=str(out), export_format="GLTF_SEPARATE", export_animations=self.arm is not None,
                                  export_animation_mode="NLA_TRACKS", export_yup=True,
                                  export_vertex_color="ACTIVE")
        print("wrote", out)

    def preview(self, out_dir, distance=3.0, target_z=0.4, textured=False):
        """Renders the first, middle and last frame of every animation to PNG, or every Nth
        frame with PREVIEW_STEP=N in the environment."""
        out_dir = Path(out_dir)
        out_dir.mkdir(parents=True, exist_ok=True)
        scene = bpy.context.scene
        cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
        scene.collection.objects.link(cam)
        cam.location = (distance * 0.6, -distance * 0.7, target_z + distance * 0.35)
        direction = Vector((0, 0, target_z)) - cam.location
        cam.rotation_euler = direction.to_track_quat("-Z", "Y").to_euler()
        scene.camera = cam
        sun = bpy.data.objects.new("sun", bpy.data.lights.new("sun", "SUN"))
        scene.collection.objects.link(sun)
        sun.rotation_euler = (math.radians(40), 0, math.radians(30))
        scene.render.engine = "BLENDER_WORKBENCH"
        scene.display.shading.color_type = "TEXTURE" if textured else "MATERIAL"
        scene.render.resolution_x, scene.render.resolution_y = 400, 300
        tracks = self.arm.animation_data.nla_tracks
        for track in tracks:
            for other in tracks:
                other.mute = other != track
            end = int(track.strips[0].frame_end)
            frames = (1, end // 2, end)
            if step := int(os.environ.get("PREVIEW_STEP", "0")):
                frames = sorted(set(range(1, end + 1, step)) | {end})
            for f in frames:
                scene.frame_set(f)
                scene.render.filepath = str(out_dir / f"{track.name}_{f:02d}.png")
                bpy.ops.render.render(write_still=True)
