#!/usr/bin/env python3
"""Makes light copies of the glTF models in assets/quaternius for the game.

    python3 tools/models/prepare.py

The packs ship 2K–4K textures with normal, roughness and occlusion maps. The game's toon
shader only uses the base color, so for each model X.gltf this writes X.lite.gltf next to it
that references only the base color texture, scaled down to at most 1024 px (written as
<texture>_1k.png). The game loads the .lite.gltf when it exists. Re-running only redoes what's
missing. Needs ImageMagick (magick).

Some meshes (the female ranger outfit) are skinned with two sets of joints and weights; raylib
only reads the first, so vertices weighted mostly by the second collapse. The light copy
merges them: each vertex keeps its four strongest joints, renormalized (in <name>.lite.bin).
"""

import json
import struct
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PACKS = ROOT / "assets" / "quaternius"
MAX_PX = 1024


def small_image(folder, uri):
    src = folder / uri
    dst_uri = str(Path(uri).with_suffix("")) + "_1k.png"
    dst = folder / dst_uri
    if not dst.exists():
        subprocess.run(["magick", str(src), "-resize", f"{MAX_PX}x{MAX_PX}>", str(dst)], check=True)
    return dst_uri


COMPONENTS = {5121: "B", 5123: "H", 5125: "I", 5126: "f"}
NORMALIZED = {5121: 255.0, 5123: 65535.0}


def read_accessor(doc, folder, i, blobs):
    a = doc["accessors"][i]
    v = doc["bufferViews"][a["bufferView"]]
    buf = v["buffer"]
    if buf not in blobs:
        blobs[buf] = (folder / doc["buffers"][buf]["uri"]).read_bytes()
    fmt = COMPONENTS[a["componentType"]]
    size = struct.calcsize(fmt)
    stride = v.get("byteStride", size * 4)
    off = v.get("byteOffset", 0) + a.get("byteOffset", 0)
    rows = [struct.unpack_from("<" + fmt * 4, blobs[buf], off + k * stride) for k in range(a["count"])]
    if fmt != "f" and a.get("normalized"):
        rows = [tuple(x / NORMALIZED[a["componentType"]] for x in r) for r in rows]
    return rows


def merge_skin_sets(doc, folder, out):
    """Folds JOINTS_1/WEIGHTS_1 into JOINTS_0/WEIGHTS_0 (top four joints per vertex)."""
    blobs, data = {}, bytearray()
    for mesh in doc.get("meshes", []):
        for prim in mesh["primitives"]:
            at = prim["attributes"]
            if "JOINTS_1" not in at or "WEIGHTS_1" not in at:
                continue
            j0, j1 = (read_accessor(doc, folder, at[k], blobs) for k in ("JOINTS_0", "JOINTS_1"))
            w0, w1 = (read_accessor(doc, folder, at[k], blobs) for k in ("WEIGHTS_0", "WEIGHTS_1"))
            joints, weights = [], []
            for a, b, c, d in zip(j0, j1, w0, w1):
                best = sorted(zip(c + d, a + b), reverse=True)[:4]
                total = sum(w for w, _ in best) or 1.0
                weights.append([w / total for w, _ in best])
                joints.append([j for _, j in best])
            for key, rows, fmt, comp in (("JOINTS_0", joints, "H", 5123), ("WEIGHTS_0", weights, "f", 5126)):
                while len(data) % 4:
                    data.append(0)
                start = len(data)
                for row in rows:
                    data.extend(struct.pack("<" + fmt * 4, *row))
                doc["bufferViews"].append({"buffer": len(doc["buffers"]), "byteOffset": start, "byteLength": len(data) - start})
                doc["accessors"].append({"bufferView": len(doc["bufferViews"]) - 1, "componentType": comp,
                                         "count": len(rows), "type": "VEC4"})
                at[key] = len(doc["accessors"]) - 1
            del at["JOINTS_1"], at["WEIGHTS_1"]
    if data:
        bin_path = out.with_suffix(".bin")
        bin_path.write_bytes(bytes(data))
        doc["buffers"].append({"uri": bin_path.name, "byteLength": len(data)})


def needs_merge(path):
    return path.exists() and '"JOINTS_1"' in path.read_text()


def prepare(path):
    out = path.with_name(path.stem + ".lite.gltf")
    if out.exists() and not needs_merge(out):
        return False
    doc = json.loads(path.read_text())
    for m in doc.get("materials", []):
        for key in ("normalTexture", "occlusionTexture", "emissiveTexture"):
            m.pop(key, None)
        pbr = m.get("pbrMetallicRoughness", {})
        pbr.pop("metallicRoughnessTexture", None)
    used = {m.get("pbrMetallicRoughness", {}).get("baseColorTexture", {}).get("index") for m in doc.get("materials", [])}
    used.discard(None)
    for i, tex in enumerate(doc.get("textures", [])):
        if i not in used:
            continue
        img = doc["images"][tex["source"]]
        if "uri" in img and (path.parent / img["uri"]).exists():
            img["uri"] = small_image(path.parent, img["uri"])
    merge_skin_sets(doc, path.parent, out)
    out.write_text(json.dumps(doc))
    return True


def main():
    n = 0
    for p in PACKS.rglob("*.gltf"):
        if p.name.endswith(".lite.gltf") or "FBX" in p.parts:
            continue
        n += prepare(p)
    print(f"prepared {n} models")


if __name__ == "__main__":
    main()
