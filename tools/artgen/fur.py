#!/usr/bin/env python3
"""Paints a seamless fur texture: many short hair strokes, all pointing roughly the same way,
in shades around a base color, evenly spread (Stable Diffusion kept adding large waves and
patches, which drown the hair at game size).

    $SD_MODELS/venv/bin/python tools/artgen/fur.py OUT.png [--seed N] [--color R,G,B]

Needs Pillow and numpy (the art venv has both).
"""

import argparse
import random

from PIL import Image, ImageDraw, ImageFilter

SIZE = 512


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("out")
    ap.add_argument("--seed", type=int, default=1)
    ap.add_argument("--color", default="95,65,45", help="base color R,G,B")
    ap.add_argument("--strokes", type=int, default=3500)
    ap.add_argument("--length", type=float, default=45, help="stroke length in pixels")
    ap.add_argument("--contrast", default="0.35,1.8", help="darkest,lightest stroke as a factor of the base color")
    args = ap.parse_args()
    args.contrast = [float(v) for v in args.contrast.split(",")]
    rnd = random.Random(args.seed)
    base = tuple(int(v) for v in args.color.split(","))

    # Draw on a 3×3 canvas and keep the middle, so strokes crossing an edge wrap around.
    img = Image.new("RGB", (SIZE * 3, SIZE * 3), base)
    d = ImageDraw.Draw(img)
    for _ in range(args.strokes):
        x, y = rnd.uniform(0, SIZE), rnd.uniform(0, SIZE)
        length = args.length * rnd.uniform(0.6, 1.3)
        slope = rnd.uniform(-0.12, 0.12)  # hair runs along +x (along the body), slightly fanned
        shade = rnd.uniform(args.contrast[0], args.contrast[1])
        c = tuple(max(0, min(255, int(v * shade))) for v in base)
        width = rnd.choice((1, 2, 2, 3))
        for ox in (0, SIZE, 2 * SIZE):
            for oy in (0, SIZE, 2 * SIZE):
                d.line([(x + ox, y + oy), (x + ox + length, y + oy + length * slope)], fill=c, width=width)
    img = img.crop((SIZE, SIZE, 2 * SIZE, 2 * SIZE)).filter(ImageFilter.GaussianBlur(0.6))
    img.save(args.out)


if __name__ == "__main__":
    main()
