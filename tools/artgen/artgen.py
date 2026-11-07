#!/usr/bin/env python3
"""Generates comic-style artwork with stable-diffusion.cpp and prepares it for the game.

    python3 tools/artgen/artgen.py tools/artgen/battle.json          # generate what's missing
    python3 tools/artgen/artgen.py tools/artgen/battle.json --only enemy_wolf --force
    python3 tools/artgen/artgen.py tools/artgen/battle.json --post   # redo only the cleanup
    python3 tools/artgen/artgen.py tools/artgen/battle.json --only enemy_wolf --try 21,22,23
        # generate candidates with other seeds into <output>/candidates/ to pick from;
        # set the chosen seed in the prompt file and run again to use it (no regeneration)

For each asset in the prompt file it
  1. runs sd-cli (SDXL-Lightning; LoRAs can be named in the prompt as <lora:name:weight>) and keeps the raw image in raw/;
     an asset can start from a rough layout picture ("init", "strength") to fix its composition,
     or name an OpenPose skeleton ("control", see poses.py) to fix a character's pose,
  2. cleans it up: sprites are cut out of their background (cutout.py), cropped and scaled to their
     target height (and mirrored if asked); backgrounds and textures are cropped and scaled to
     size. Textures ("kind": "texture") are made seamless, so they can be repeated,
  3. writes the result to the output folder with a .json next to it holding the prompt,
     seed and settings, so every image can be regenerated.
Existing raw images are reused, so re-running only generates what's missing.

Needs ImageMagick (magick) and a venv with rembg for the cleanup. Paths can be overridden
with SD_CLI, SD_MODELS and CUTOUT_PY. See design/game-design.md, How the art is made.
"""

import argparse
import json
import os
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SD_CLI = os.environ.get("SD_CLI", "sd-cli")
SD_MODELS = Path(os.environ.get("SD_MODELS", "sd-models"))
# Background removal runs in a venv with rembg (see cutout.py).
CUTOUT_PY = Path(os.environ.get("CUTOUT_PY", SD_MODELS / "venv/bin/python"))
CUTOUT_MODEL = "birefnet-general"
CONTROLNET = "controlnet-openpose-sdxl.safetensors"  # xinsir, Apache-2.0

# Settings found to work with Vulkan: SDXL-Lightning wants 8 steps,
# guidance 1, Euler with sgm_uniform. GPU VAE decoding can need tiling, with tiles small enough that no step
# hits the driver's compute timeout (bigger tiles now and then reset the GPU).
GEN = {
    "model": "sdxl_lightning_8step.safetensors",
    "vae": "sdxl_vae.safetensors",
    "steps": 8,
    "cfg": 1.0,
    "sampler": "euler",
    "scheduler": "sgm_uniform",
    "vae_tile": 256,
    "vae_overlap": 0.25,
}


def generate(prompt, negative, width, height, seed, out, init=None, strength=0.8, control=None, mask=None):
    cmd = [
        SD_CLI,
        "-m", str(SD_MODELS / GEN["model"]),
        "--vae", str(SD_MODELS / GEN["vae"]),
        "--lora-model-dir", str(SD_MODELS),
        "-p", prompt, "-n", negative,
        "--steps", str(GEN["steps"]), "--cfg-scale", str(GEN["cfg"]),
        "--sampling-method", GEN["sampler"], "--scheduler", GEN["scheduler"],
        "-W", str(width), "-H", str(height), "-s", str(seed),
        "--vae-tiling", "--vae-tile-size", str(GEN["vae_tile"]), "--vae-tile-overlap", str(GEN["vae_overlap"]),
        "-o", str(out),
    ]
    if control:
        # A ControlNet image fixes the pose (an OpenPose skeleton from poses.py).
        cmd += ["--control-net", str(SD_MODELS / CONTROLNET), "--control-image", str(ROOT / control),
                "--control-strength", "0.9"]
        # Needs the local ggml-vulkan fix (see tools/artgen/patches/); without it sd-cli crashes
        # with ControlNet on Vulkan. Fallback: add "--backend", "diffusion=cpu,controlnet=cpu"
        # (about 25 minutes per image instead of 5).
    if init:
        # Start from a rough layout picture (img2img) to fix the composition, e.g. the horizon.
        cmd += ["-i", str(ROOT / init), "--strength", str(strength)]
    if mask:
        # Only the white part of the mask is painted again (used by the fix scripts).
        cmd += ["--mask", str(ROOT / mask)]
    log = out.with_suffix(".log")
    # The GPU driver now and then loses the device while decoding; trying again usually works.
    for attempt in range(3):
        with open(log, "w") as f:
            result = subprocess.run(cmd, stdout=f, stderr=subprocess.STDOUT)
        if result.returncode == 0 and out.exists():
            return
        if "ErrorDeviceLost" not in log.read_text():
            break
        print(f"  GPU device lost, trying again", flush=True)
        time.sleep(5)
    sys.exit(f"sd-cli failed for {out.name}, see {log}")


def magick(*args):
    subprocess.run(["magick", *map(str, args)], check=True)


def clean_sprite(raw, out, asset):
    """Cuts the figure out with a segmentation model, drops ground lines and stray bits, crops,
    scales and optionally mirrors."""
    cut = out.with_name(out.stem + ".cut.png")
    mask = out.with_name(out.stem + ".mask.gray")
    try:
        env = dict(os.environ, U2NET_HOME=str(SD_MODELS / "rembg"))
        subprocess.run([str(CUTOUT_PY), "-I", str(Path(__file__).with_name("cutout.py")),
                        CUTOUT_MODEL, str(raw), str(cut)], env=env, check=True)
        w, h = (int(v) for v in subprocess.run(["magick", "identify", "-format", "%w %h", str(cut)],
                                                capture_output=True, text=True).stdout.split())
        alpha = subprocess.run(["magick", cut, "-alpha", "extract", "-depth", "8", "gray:-"],
                               capture_output=True, check=True).stdout
        mask.write_bytes(keep_figure(alpha, w, h))
        args = [cut, "(", "-size", f"{w}x{h}", "-depth", "8", f"gray:{mask}", ")",
                "-compose", "CopyOpacity", "-composite", "-trim", "+repage"]
        if asset.get("flip"):
            args.append("-flop")
        args += ["-resize", f"x{asset['height']}", out]
        magick(*args)
    finally:
        cut.unlink(missing_ok=True)
        mask.unlink(missing_ok=True)


def keep_figure(alpha, w, h):
    """Returns the alpha mask with only the figure: rows in the lower part that run from edge to
    edge (a ground or horizon line) are cleared, then everything not connected to the largest
    shape is dropped, except pieces big enough to be part of the figure (15% of the largest)."""
    a = bytearray(255 if v > 127 else 0 for v in alpha)  # solid parts, for finding shapes
    for y in range(h * 3 // 4, h):
        row = a[y * w:(y + 1) * w]
        if row[1] and row[w - 2]:
            a[y * w:(y + 1) * w] = bytes(w)
    # Label 8-connected shapes.
    label = [0] * (w * h)
    sizes = [0]
    for start in range(w * h):
        if not a[start] or label[start]:
            continue
        n = len(sizes)
        label[start], stack, size = n, [start], 0
        while stack:
            i = stack.pop()
            size += 1
            x, y = i % w, i // w
            for dy in (-1, 0, 1):
                for dx in (-1, 0, 1):
                    nx, ny = x + dx, y + dy
                    if 0 <= nx < w and 0 <= ny < h:
                        j = ny * w + nx
                        if a[j] and not label[j]:
                            label[j] = n
                            stack.append(j)
        sizes.append(size)
    if len(sizes) == 1:
        return bytes(a)
    # Keep the soft edge around kept shapes: semi-transparent pixels next to a kept pixel.
    keep = {n for n, s in enumerate(sizes) if n and s >= max(sizes) * 0.15}
    out = bytearray(w * h)
    for i in range(w * h):
        if label[i] in keep:
            out[i] = alpha[i]
        elif alpha[i] and not a[i]:
            x, y = i % w, i // w
            if any(label[ny * w + nx] in keep for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1))
                   if 0 <= nx < w and 0 <= ny < h):
                out[i] = alpha[i]
    return bytes(out)


def clean(kind):
    """Returns the cleanup step for a kind of asset."""
    return {"sprite": clean_sprite, "texture": clean_texture}.get(kind, clean_background)


def clean_texture(raw, out, asset):
    """Makes a seamless texture from four copies of the image: as is, and shifted by half its
    width, its height, and both. Each copy wraps around without seams at the lines where the
    others have theirs; the copies are blended with weights that fall to zero along each
    copy's own seams (image edges, middle lines). (sd-cli's --circular mode breaks
    SDXL-Lightning images, so it isn't used.)"""
    w, h = asset["size"]
    parts = [out.with_name(f"{out.stem}.part{k}.png") for k in range(4)]
    try:
        magick(raw, "-resize", f"{w}x{h}^", "-gravity", "center", "-extent", f"{w}x{h}", "+repage", parts[0])
        for k, (dx, dy) in enumerate([(w // 2, 0), (0, h // 2), (w // 2, h // 2)], 1):
            magick(parts[0], "-roll", f"+{dx}+{dy}", parts[k])
        # (one-letter names are taken: r, g, b, a... are channels)
        fx = ("ex=min(i,w-1-i)/w; ey=min(j,h-1-j)/h; mx=0.5-ex; my=0.5-ey;"
              "ww=min(1,4*ex)*min(1,4*ey); wx=min(1,4*mx)*min(1,4*ey);"
              "wy=min(1,4*ex)*min(1,4*my); wz=min(1,4*mx)*min(1,4*my);"
              "(u[0]*ww+u[1]*wx+u[2]*wy+u[3]*wz)/(ww+wx+wy+wz)")
        magick(*parts, "-fx", fx, out)
    finally:
        for p in parts:
            p.unlink(missing_ok=True)


def clean_background(raw, out, asset):
    """Backgrounds: scaled and cropped to size."""
    w, h = asset["size"]
    magick(raw, "-resize", f"{w}x{h}^", "-gravity", "center", "-extent", f"{w}x{h}", out)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("prompts", type=Path)
    ap.add_argument("--only", help="comma-separated asset ids")
    ap.add_argument("--force", action="store_true", help="generate again even if a raw image exists")
    ap.add_argument("--post", action="store_true", help="only redo the cleanup from existing raw images")
    ap.add_argument("--try", dest="tries", help="comma-separated seeds: generate candidates to pick from")
    args = ap.parse_args()

    spec = json.loads(args.prompts.read_text())
    out_dir = ROOT / spec["output"]
    raw_dir = out_dir / "raw"
    raw_dir.mkdir(parents=True, exist_ok=True)
    only = set(args.only.split(",")) if args.only else None

    for asset in spec["assets"]:
        aid = asset["id"]
        if only and aid not in only:
            continue
        kind = asset.get("kind", "sprite")
        style = spec["style"][asset.get("style", kind)]  # an asset can pick a style other than its kind's
        suffix = style["suffix"]
        prompt = f"{style['prefix']} {asset['prompt']}, {suffix}"
        negative = style["negative"] + (", " + asset["negative"] if asset.get("negative") else "")
        width, height = asset.get("gen", style.get("gen", [768, 768]))
        seed = asset.get("seed", 1)
        raw = raw_dir / f"{aid}.png"
        cand_dir = out_dir / "candidates"

        if args.tries:
            cand_dir.mkdir(exist_ok=True)
            for t in args.tries.split(","):
                craw = raw_dir / f"{aid}_s{t}.png"
                if not craw.exists():
                    print(f"generating candidate {aid} seed {t}...", flush=True)
                    start = time.time()
                    generate(prompt, negative, width, height, int(t), craw, asset.get("init"), asset.get("strength", 0.8), asset.get("control"))
                    print(f"  done in {time.time() - start:.0f}s", flush=True)
                preview = cand_dir / f"{aid}_s{t}.png"
                # Shown as generated (not mirrored), so the facing can be judged before setting "flip".
                clean(kind)(craw, preview, {**asset, "flip": False})
                print(f"  candidate {preview.relative_to(ROOT)}", flush=True)
            continue

        # A candidate generated earlier with the chosen seed is reused instead of generating again.
        chosen = raw_dir / f"{aid}_s{seed}.png"
        if chosen.exists() and (args.force or not raw.exists() or chosen.stat().st_mtime > raw.stat().st_mtime):
            raw.write_bytes(chosen.read_bytes())
        if not args.post and (args.force and not chosen.exists() or not raw.exists()):
            print(f"generating {aid} ({width}x{height}, seed {seed})...", flush=True)
            start = time.time()
            generate(prompt, negative, width, height, seed, raw, asset.get("init"), asset.get("strength", 0.8), asset.get("control"))
            print(f"  done in {time.time() - start:.0f}s", flush=True)
        if not raw.exists():
            print(f"skipping {aid}: no raw image")
            continue

        out = out_dir / f"{aid}.png"
        clean(kind)(raw, out, asset)
        meta = {"id": aid, "kind": kind, "prompt": prompt, "negative": negative, "seed": seed,
                "size": [width, height], "settings": GEN}
        (out_dir / f"{aid}.json").write_text(json.dumps(meta, indent=2) + "\n")
        print(f"  wrote {out.relative_to(ROOT)}", flush=True)


if __name__ == "__main__":
    main()
