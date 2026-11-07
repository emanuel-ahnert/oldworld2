#!/usr/bin/env python3
"""Hand fixes for bg_grassland (seed 241), applied to artgen's output:

  - a red dot on the right tree, which looks like a glowing eye, is covered with bark from
    25 px to its right;
  - the dark teal sky becomes a daytime blue, keeping the brush texture: sky pixels are found
    by hue (blue clearly above red, unlike the grey-green trees) in the top part of the image,
    and their brightness is mapped onto a blue gradient. The far forest is lightened to match;
  - the light ray falling into the clearing has the same teal as the sky, so it is found by
    its position instead and turned into warm sunlight;
  - three thin far trunks end in the gaps between the crowns, which against the blue sky
    looks like trees without tops, so Stable Diffusion paints forest over them (inpainting a
    crop of the far forest, scaled up to SDXL size).

    $SD_MODELS/venv/bin/python -I tools/artgen/fix/bg_grassland.py IN.png OUT.png
"""

import sys
import tempfile
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw, ImageFilter

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import artgen  # noqa: E402

TOP = 340  # the sky never reaches below this row
# The ray, as its left and right edge at row 100 and how far they move right per row, the rows
# it covers and the strength of its sunlight: the bright core, and the faint haze on its left
# (over trees). Right of the core is open sky down to the treeline, so the soft fringe there
# only starts below it.
# The far forest crop that is painted again, the trunks in it (x, lowest row) and the seed.
FOREST = (560, 40, 1000, 370)
TRUNKS = ((730, 275), (822, 265), (945, 250))
HALF = 16  # half width of the painted strips (wider ones get whole new trees in another style)
FOREST_SEED = 4
FOREST_PROMPT = ("comic book illustration of a dense line of leafy forest trees with round bushy crowns under a "
                 "clear blue sky, hand-drawn fantasy RPG illustration, detailed clean dark ink linework, muted "
                 "earthy color palette, cel shading with fine ink hatching")
FOREST_NEGATIVE = ("photo, realistic, blurry, text, poles, posts, sticks, bare trunks, dead trees, people, animals, "
                   "clouds, fir trees, pine trees")
RAY = (482, 510, 0.46, 50, 335, 0.55)
HAZE = (420, 482, 0.46, 40, 260, 0.25)
FRINGE = (510, 534, 0.46, 180, 335, 0.3)


def soft(m, radius=1.2):
    return np.asarray(Image.fromarray((m * 255).astype(np.uint8)).filter(ImageFilter.GaussianBlur(radius))) / 255.0


def main():
    img = np.asarray(Image.open(sys.argv[1]).convert("RGB")).astype(np.float32)
    img[274:285, 1234:1248] = img[274:285, 1259:1273]
    h, w, _ = img.shape
    r, g, b = img[..., 0], img[..., 1], img[..., 2]
    ys, xs = np.mgrid[0:h, 0:w].astype(np.float32)
    lum = 0.3 * r + 0.59 * g + 0.11 * b

    def band(spec):
        left, right, slope, y0, y1, strength = spec
        shift = (ys - 100) * slope
        return ((xs >= left + shift) & (xs <= right + shift) & (ys >= y0) & (ys < y1)) * strength

    light = np.maximum(np.maximum(band(RAY), band(HAZE)), band(FRINGE))  # sunlight strength
    ray = (light > 0).astype(np.float32)

    teal = np.clip((b - r - 8) / 10, 0, 1) * (g > r)
    teal[TOP:] = 0
    sky = soft(teal * (1 - ray))
    beam = soft(teal * ray)
    light = soft(light, 3)

    rows = (np.arange(h, dtype=np.float32) / TOP).clip(0, 1)[:, None, None]
    top, horizon = np.array([95, 155, 215]), np.array([175, 210, 235])
    blue = np.clip((top + (horizon - top) * rows) * (0.75 + 0.25 * np.clip(lum / 85, 0.6, 1.6)[..., None]), 0, 255)
    sun = np.clip(np.array([255, 240, 190]) * np.clip(lum / 100, 0.7, 1.2)[..., None], 0, 255)

    # The far forest gets lighter and a little hazy, as in daylight. It is the dark green below
    # the sky (the trunks are brown, the lit grass bright).
    far = np.clip((g - r) / 10, 0, 1) * np.clip((110 - lum) / 30, 0, 1) * (1 - np.maximum(sky, beam))
    far *= np.clip((TOP + 20 - np.arange(h)) / 80, 0, 1)[:, None]  # fades out toward the grass
    far = soft(far)[..., None]
    lit = img * 1.3 * 0.8 + horizon * 0.2

    out = img * (1 - far) + lit * far
    out = out * (1 - sky[..., None]) + blue * sky[..., None]
    # The beam stays see-through: the lightened forest shows behind it (its teal turned into
    # the far forest's green at the same brightness).
    forest = np.array([47, 68, 49]) / 63.8 * lum[..., None]
    behind = forest * 1.3 * 0.8 + horizon * 0.2
    beam, light = beam[..., None], light[..., None]
    out = out * (1 - beam) + (behind * (1 - light) + sun * light) * beam

    out = repaint_trunks(out, top, horizon)
    Image.fromarray(np.clip(out, 0, 255).astype(np.uint8)).save(sys.argv[2])


def repaint_trunks(img, top, horizon):
    x0, y0, x1, y1 = FOREST
    size = (1024, 768)
    mask = Image.new("L", img.shape[1::-1], 0)
    d = ImageDraw.Draw(mask)
    for x, bottom in TRUNKS:
        d.rectangle([x - HALF, 110, x + HALF, bottom], fill=255)
    with tempfile.TemporaryDirectory() as tmp:
        tmp = Path(tmp)
        Image.fromarray(np.clip(img, 0, 255).astype(np.uint8)).crop(FOREST).resize(size, Image.LANCZOS).save(tmp / "in.png")
        mask.crop(FOREST).resize(size, Image.NEAREST).save(tmp / "mask.png")
        artgen.generate(FOREST_PROMPT, FOREST_NEGATIVE, *size, FOREST_SEED, tmp / "out.png",
                        init=tmp / "in.png", strength=0.85, mask=tmp / "mask.png")
        painted = np.asarray(Image.open(tmp / "out.png").convert("RGB").resize((x1 - x0, y1 - y0), Image.LANCZOS)).astype(np.float32)

    # The leaves SD painted are a slightly different green; each strip is matched to the forest
    # on both sides of it.
    before = img[y0:y1, x0:x1]
    ys, xs = np.mgrid[y0:y1, x0:x1]
    for x, bottom in TRUNKS:
        rows = (ys >= 110) & (ys <= bottom)
        inside = rows & (np.abs(xs - x) <= HALF)
        beside = rows & (np.abs(xs - x) > HALF) & (np.abs(xs - x) <= HALF + 14)
        a = inside & (painted[..., 1] > painted[..., 2] + 4) & (painted[..., 1] > painted[..., 0])
        b = beside & (before[..., 1] > before[..., 2] + 4) & (before[..., 1] > before[..., 0])
        if a.any() and b.any():
            painted[inside] *= before[b].mean(0) / painted[a].mean(0)

    # The sky SD painted is a slightly different blue: it gets the gradient of the rest.
    rows = (np.arange(y0, y1, dtype=np.float32) / TOP).clip(0, 1)[:, None, None]
    blue = top + (horizon - top) * rows
    r, g, b = painted[..., 0], painted[..., 1], painted[..., 2]
    sky = np.clip((b - g - 10) / 10, 0, 1)[..., None]
    painted = painted * (1 - sky) + np.broadcast_to(blue, painted.shape) * sky

    m = soft(np.asarray(mask.crop(FOREST)) / 255.0, 5)[..., None]
    out = img.copy()
    out[y0:y1, x0:x1] = img[y0:y1, x0:x1] * (1 - m) + painted * m
    return out


if __name__ == "__main__":
    main()
