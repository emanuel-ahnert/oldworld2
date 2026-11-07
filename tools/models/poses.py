"""Renders outfit pieces from tools/models/human.py in the animation library's poses, to check
for clipping without playing the game.

    blender -b --python tools/models/poses.py -- SKELETON.glb PIECES_DIR OUT_DIR LABEL PIECE,PIECE...

SKELETON.glb is the animation file (UAL1_Standard.glb); its armature plays each action and the
pieces are bound to it (same bones, same rest pose). Writes OUT_DIR/LABEL_ACTION_0.png and
_1.png, at 35% and 70% through each action listed in ACTIONS.
"""

import sys

import bpy
from mathutils import Vector

args = sys.argv[sys.argv.index("--") + 1:]
ual, hum, out, label = args[0], args[1], args[2], args[3]
pieces = args[4].split(",")
ACTIONS = ["Sword_Attack", "Punch_Cross", "Spell_Simple_Shoot", "Hit_Chest", "Death01", "Sitting_Idle_Loop", "Jog_Fwd_Loop", "Sword_Idle"]
bpy.ops.wm.read_factory_settings(use_empty=True)
bpy.ops.import_scene.gltf(filepath=ual)
arm = next(o for o in bpy.data.objects if o.type == "ARMATURE")
for o in list(bpy.data.objects):
    if o.type == "MESH":
        bpy.data.objects.remove(o)
for p in pieces:
    before = set(bpy.data.objects)
    bpy.ops.import_scene.gltf(filepath=f"{hum}/{p}.gltf")
    new = set(bpy.data.objects) - before
    for o in new:
        if o.type == "MESH":
            for m in o.modifiers:
                if m.type == "ARMATURE":
                    m.object = arm
            o.parent = arm
            o.matrix_parent_inverse.identity()
    for o in new:
        if o.type == "ARMATURE":
            bpy.data.objects.remove(o)
scene = bpy.context.scene
cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
scene.collection.objects.link(cam)
cam.location = (2.2, -3.2, 1.6)
cam.rotation_euler = (Vector((0, 0, 0.9)) - cam.location).to_track_quat("-Z", "Y").to_euler()
scene.camera = cam
scene.render.engine = "BLENDER_WORKBENCH"
scene.display.shading.color_type = "MATERIAL"
scene.render.resolution_x, scene.render.resolution_y = 300, 330
arm.animation_data_create()
for name in ACTIONS:
    act = bpy.data.actions.get(name)
    if not act:
        print("missing", name); continue
    arm.animation_data.action = act
    if hasattr(arm.animation_data, "action_slot") and act.slots:
        arm.animation_data.action_slot = act.slots[0]
    a, b = act.frame_range
    for i, t in enumerate((0.35, 0.7)):
        f = int(a + (b - a) * t)
        scene.frame_set(f)
        scene.render.filepath = f"{out}/{label}_{name}_{i}.png"
        bpy.ops.render.render(write_still=True)
