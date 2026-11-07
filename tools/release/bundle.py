#!/usr/bin/env python3
"""Zips the assets the game needs at runtime, for tools/release/windows.sh.

    python3 tools/release/bundle.py OUT.zip

That is assets/generated without the images art generation keeps for itself (raw/, candidates/),
and of the Quaternius packs only the models in pack_models.txt with the files they refer to.
TestPackModelsInRelease checks that the list holds every pack model the game loads.
"""

import json
import sys
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ASSETS = ROOT / "assets"
PACKS = ASSETS / "quaternius"


def gltf_files(path):
    """The buffers and the images a glTF's materials use (raylib loads no others)."""
    d = json.loads(path.read_text())
    names = [b["uri"] for b in d.get("buffers", []) if "uri" in b]
    for m in d.get("materials", []):
        pbr = m.get("pbrMetallicRoughness", {})
        slots = [pbr.get("baseColorTexture"), pbr.get("metallicRoughnessTexture"),
                 m.get("normalTexture"), m.get("occlusionTexture"), m.get("emissiveTexture")]
        for s in slots:
            if s:
                uri = d["images"][d["textures"][s["index"]]["source"]].get("uri")
                if uri:
                    names.append(uri)
    return {path.parent / n for n in names}


def main():
    files = {p for p in (ASSETS / "generated").rglob("*")
             if p.is_file() and not {"raw", "candidates"} & set(p.parts)}
    for line in (Path(__file__).parent / "pack_models.txt").read_text().splitlines():
        if line:
            model = PACKS / line
            files.add(model)
            if model.suffix == ".gltf":
                files |= gltf_files(model)
    with zipfile.ZipFile(sys.argv[1], "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for p in sorted(files):
            z.write(p, p.relative_to(ROOT).as_posix())
    print(f"{sys.argv[1]}: {len(files)} files, {Path(sys.argv[1]).stat().st_size / 1e6:.0f} MB")


if __name__ == "__main__":
    main()
