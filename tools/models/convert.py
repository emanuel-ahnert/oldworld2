"""Converts models (with their animations) to glTF that raylib animates correctly.

    blender -b --python tools/models/convert.py -- OUT_DIR IN.blend|IN.fbx|IN.gltf...

Runs inside Blender without a window. Writes OUT_DIR/<name>.gltf (+ .bin, textures) per input.
Used for packs that ship no glTF (prefer the .blend when a pack has both: Quaternius' FBX
exports sometimes lack animations that the .blend has), and to re-export glTF files whose
skinning raylib gets wrong (the Ultimate Monsters "Big" models stretch their arms when
animated; Blender's export of the same model works). Material colors with alpha 0 (common
in those exports) are made opaque.
"""

import json
import sys
from pathlib import Path

import bpy

args = sys.argv[sys.argv.index("--") + 1:]
out = Path(args[0])
out.mkdir(parents=True, exist_ok=True)
for src in args[1:]:
    if src.endswith(".blend"):
        bpy.ops.wm.open_mainfile(filepath=src)
    elif src.endswith(".gltf") or src.endswith(".glb"):
        bpy.ops.wm.read_factory_settings(use_empty=True)
        bpy.ops.import_scene.gltf(filepath=src)
    else:
        bpy.ops.wm.read_factory_settings(use_empty=True)
        bpy.ops.import_scene.fbx(filepath=src)
    dst = out / (Path(src).stem + ".gltf")
    bpy.ops.export_scene.gltf(filepath=str(dst), export_format="GLTF_SEPARATE", export_animations=True,
                              export_animation_mode="ACTIONS", export_yup=True)
    doc = json.loads(dst.read_text())
    for m in doc.get("materials", []):
        c = m.get("pbrMetallicRoughness", {}).get("baseColorFactor")
        if c and c[3] == 0:
            c[3] = 1
    dst.write_text(json.dumps(doc))
    print("wrote", dst)
