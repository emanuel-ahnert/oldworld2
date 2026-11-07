#!/usr/bin/env python3
"""Cuts a figure out of its background with a segmentation model (rembg), writing a PNG with
transparency. Run with the Python of the venv that has rembg installed:

    $SD_MODELS/venv/bin/python tools/artgen/cutout.py MODEL IN.png OUT.png

Models (downloaded on first use into $U2NET_HOME): birefnet-general (MIT), isnet-anime
(Apache-2.0).
"""

import sys

from PIL import Image
from rembg import new_session, remove

model, src, dst = sys.argv[1:4]
remove(Image.open(src), session=new_session(model)).save(dst)
