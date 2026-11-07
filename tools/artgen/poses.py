#!/usr/bin/env python3
"""Draws OpenPose skeletons for character art (ControlNet "control" images in the prompt files).

    python3 tools/artgen/poses.py

The skeleton fixes pose and facing, which the prompt alone doesn't do reliably. Poses are
built from a few joint angles, so every direction and walk frame has the same proportions.
Output: tools/artgen/poses/<name>.png, black background, OpenPose (COCO-18) colors.

Facing in the image: "down" faces the viewer, "up" shows the back, "right" is the side view
(left is mirrored in the game). In OpenPose the person's right side is drawn in the warm
colors; from the front it appears on the image's left.
"""

import math
import subprocess
from pathlib import Path

OUT = Path(__file__).with_name("poses")

# COCO-18 keypoints.
NOSE, NECK, RSHO, RELB, RWRI, LSHO, LELB, LWRI, RHIP, RKNE, RANK, LHIP, LKNE, LANK, REYE, LEYE, REAR, LEAR = range(18)

LIMBS = [(NECK, RSHO), (NECK, LSHO), (RSHO, RELB), (RELB, RWRI), (LSHO, LELB), (LELB, LWRI),
         (NECK, RHIP), (RHIP, RKNE), (RKNE, RANK), (NECK, LHIP), (LHIP, LKNE), (LKNE, LANK),
         (NECK, NOSE), (NOSE, REYE), (REYE, REAR), (NOSE, LEYE), (LEYE, LEAR)]
COLORS = ["#ff0000", "#ff5500", "#ffaa00", "#ffff00", "#aaff00", "#55ff00", "#00ff00", "#00ff55",
          "#00ffaa", "#00ffff", "#00aaff", "#0055ff", "#0000ff", "#5500ff", "#aa00ff", "#ff00ff",
          "#ff00aa", "#ff0055"]

# Body proportions in units of the figure height.
SHOULDER, HIP, UPPER_ARM, FOREARM, THIGH, SHIN = 0.11, 0.06, 0.15, 0.14, 0.22, 0.22
NECK_Y, HIP_Y = 0.18, 0.47


def figure(facing, swing=0.0, sword_up=False, turned=False):
    """Returns keypoints (x, y in 0..1 of a 1×1 box, y down) or None for hidden ones.
    swing: walk phase, -1..1 (legs and arms swing in opposite directions)."""
    p = [None] * 18
    p[NECK] = (0.5, NECK_Y)
    if facing == "right":
        # Side view: both shoulders and hips nearly on top of each other.
        sx, hx = 0.02, 0.015
        p[RSHO], p[LSHO] = (0.5 - sx, NECK_Y + 0.01), (0.5 + sx, NECK_Y + 0.01)
        p[RHIP], p[LHIP] = (0.5 - hx, HIP_Y), (0.5 + hx, HIP_Y)
        p[NOSE] = (0.56, 0.10)
        p[REYE], p[REAR] = (0.55, 0.085), (0.49, 0.09)
        far, near = (LSHO, LELB, LWRI, LHIP, LKNE, LANK), (RSHO, RELB, RWRI, RHIP, RKNE, RANK)
        for (sho, elb, wri, hip, kne, ank), s in ((near, swing), (far, -swing)):
            leg = s * 0.45  # radians from vertical, forward is +x
            p[kne] = (p[hip][0] + THIGH * math.sin(leg), p[hip][1] + THIGH * math.cos(leg))
            p[ank] = (p[kne][0] + SHIN * math.sin(leg * 0.6), p[kne][1] + SHIN * math.cos(leg * 0.6))
            arm = -s * 0.4
            p[elb] = (p[sho][0] + UPPER_ARM * math.sin(arm), p[sho][1] + UPPER_ARM * math.cos(arm))
            p[wri] = (p[elb][0] + FOREARM * math.sin(arm + 0.5), p[elb][1] + FOREARM * math.cos(arm + 0.5))
        return p

    # Front ("down") or back ("up") view. From the back, the person's right is on the image's right.
    side = -1 if facing == "down" else 1
    for sho, elb, wri, hip, kne, ank, s in ((RSHO, RELB, RWRI, RHIP, RKNE, RANK, side),
                                            (LSHO, LELB, LWRI, LHIP, LKNE, LANK, -side)):
        p[sho] = (0.5 + s * SHOULDER, NECK_Y + 0.01)
        p[elb] = (p[sho][0] + s * 0.03, p[sho][1] + UPPER_ARM)
        p[wri] = (p[elb][0] + s * 0.01, p[elb][1] + FOREARM)
        p[hip] = (0.5 + s * HIP, HIP_Y)
        lift = max(0.0, swing * s) * 0.05  # walking: one knee comes up a little
        p[kne] = (p[hip][0] + s * 0.005, p[hip][1] + THIGH - lift)
        p[ank] = (p[kne][0], p[kne][1] + SHIN - lift * 0.5)
    if facing == "up" and turned:
        # Back view turned toward the upper right: shoulders narrower, head turned right so the
        # right ear comes toward the viewer and the nose shows in profile at the right.
        for k in (RSHO, RELB, RWRI, LSHO, LELB, LWRI):
            p[k] = (0.5 + (p[k][0] - 0.5) * 0.8, p[k][1])
        p[LELB] = (p[LSHO][0] + 0.02, p[LSHO][1] + 0.12)  # shield arm forward, across the body
        p[LWRI] = (p[LELB][0] + 0.09, p[LELB][1] - 0.01)
        p[REAR], p[LEAR] = (0.53, 0.10), None
        p[NOSE], p[REYE] = (0.585, 0.105), (0.57, 0.09)
    if sword_up:
        p[RELB] = (p[RSHO][0] + side * 0.06, p[RSHO][1] + 0.08)
        p[RWRI] = (p[RELB][0] + side * 0.02, p[RELB][1] - 0.06)
    if facing == "down":
        p[NOSE] = (0.5, 0.105)
        p[REYE], p[LEYE] = (0.48, 0.09), (0.52, 0.09)
        p[REAR], p[LEAR] = (0.455, 0.10), (0.545, 0.10)
    elif not turned:
        p[REAR], p[LEAR] = (0.545, 0.10), (0.455, 0.10)  # no face from behind
    return p


def draw(points, w, h, out):
    # The figure fills 86% of the height, centered.
    fh = h * 0.86
    ox, oy = w / 2 - fh / 2, (h - fh) / 2
    xy = [None if q is None else (ox + q[0] * fh, oy + q[1] * fh) for q in points]
    thick = max(4, int(fh / 60))
    cmds = []
    for i, (a, b) in enumerate(LIMBS):
        if xy[a] and xy[b]:
            cmds += ["-stroke", COLORS[i], "-strokewidth", str(thick),
                     "-draw", f"line {xy[a][0]:.0f},{xy[a][1]:.0f} {xy[b][0]:.0f},{xy[b][1]:.0f}"]
    for i, q in enumerate(xy):
        if q:
            r = thick
            cmds += ["-stroke", "none", "-fill", COLORS[i],
                     "-draw", f"circle {q[0]:.0f},{q[1]:.0f} {q[0] + r:.0f},{q[1]:.0f}"]
    subprocess.run(["magick", "-size", f"{w}x{h}", "xc:black", *cmds, str(out)], check=True)


def main():
    OUT.mkdir(exist_ok=True)
    size = (640, 896)
    for facing in ("down", "up", "right"):
        draw(figure(facing), *size, OUT / f"stand_{facing}.png")
        for i, s in enumerate((1.0, -1.0)):
            draw(figure(facing, swing=s), *size, OUT / f"walk_{facing}_{i}.png")
    # Battle: back view, turned toward the enemy at the upper right, weapon raised.
    draw(figure("up", sword_up=True, turned=True), *size, OUT / "battle_back.png")
    print("wrote", len(list(OUT.glob("*.png"))), "poses to", OUT)


if __name__ == "__main__":
    main()
