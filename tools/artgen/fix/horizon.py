#!/usr/bin/env python3
"""Moves the horizon of a battle background up, so enemies (feet at about row 305 of 720)
stand on open ground instead of on the skyline: the sky is squeezed a little (its top is
cropped), the ground below the horizon stretched down to the bottom edge.

    $SD_MODELS/venv/bin/python -I tools/artgen/fix/horizon.py IN.png OUT.png --at ROW [--to ROW] [--squash F]
"""

import argparse

import numpy as np
from PIL import Image


def move(src, at, to=185, squash=1.35):
    """Moves the horizon of an image (an array of rows) from row at to row to."""
    h = src.shape[0]
    rows = np.arange(h, dtype=np.float32)
    # For each new row, the old row it comes from.
    sky = at - (to - rows) * squash
    ground = at + (rows - to) * (h - 1 - at) / (h - 1 - to)
    old = np.where(rows < to, sky, ground).clip(0, h - 1)
    lo = np.floor(old).astype(int)
    hi = np.minimum(lo + 1, h - 1)
    f = (old - lo)[:, None, None]
    return src[lo] * (1 - f) + src[hi] * f


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("src")
    ap.add_argument("out")
    ap.add_argument("--at", type=float, required=True, help="the horizon's row in the image")
    ap.add_argument("--to", type=float, default=185, help="the row it moves to")
    ap.add_argument("--squash", type=float, default=1.35, help="how much the sky is squeezed")
    a = ap.parse_args()
    src = np.asarray(Image.open(a.src).convert("RGB")).astype(np.float32)
    Image.fromarray(move(src, a.at, a.to, a.squash).astype(np.uint8)).save(a.out)


if __name__ == "__main__":
    main()
