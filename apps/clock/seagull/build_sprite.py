"""Build a tiny Berry sprite from the cropped reference image. Requires Pillow."""

import argparse
from pathlib import Path

from PIL import Image, ImageDraw


ROOT = Path(__file__).resolve().parent
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--source", type=Path, help="Edited 388 x 177 bird-only screenshot")
args = parser.parse_args()
assets = ROOT / "assets"
assets.mkdir(exist_ok=True)
if args.source:
    with Image.open(args.source) as source:
        source.convert("RGB").save(assets / "reference.png")

with Image.open(assets / "reference.png") as source:
    bird = source.convert("RGBA")
if bird.size != (388, 177):
    raise ValueError("Expected the edited 388 x 177 bird-only reference image")

# Trace the body separately: the original legs are replaced by animated ones.
outline = [
    (52, 337), (62, 345), (80, 345), (84, 339), (78, 331), (82, 329),
    (96, 340), (102, 330), (121, 320), (146, 310), (177, 303), (213, 304),
    (244, 312), (267, 320), (289, 324), (333, 325), (360, 330), (381, 336),
    (376, 342), (389, 344), (378, 351), (354, 352), (343, 357), (321, 359),
    (311, 365), (280, 368), (247, 374), (208, 377), (170, 377), (147, 374),
    (132, 369), (114, 374), (99, 370), (88, 365), (78, 355), (67, 351), (56, 345),
]
mask = Image.new("L", bird.size)
ImageDraw.Draw(mask).polygon([(x - 24, y - 222) for x, y in outline], fill=255)
bird.putalpha(mask)
bird = bird.crop((21, 78, 376, 163)).resize((48, 12), Image.Resampling.LANCZOS)
palette = [0, 0xF4F4EF, 0xB4B7B4, 0x767D7E, 0x383F43, 0xEAB044, 0xC64B25]
sprite = Image.new("RGBA", bird.size)
rows = []
for y in range(bird.height):
    row = []
    for x in range(bird.width):
        r, g, b, a = bird.getpixel((x, y))
        if a < 100:
            index = 0
        elif r > g * 1.25 and r > b * 1.6:
            index = 5 if g > 110 else 6
        else:
            light = (r + g + b) // 3
            index = 1 if light > 220 else 2 if light > 160 else 3 if light > 100 else 4
        row.append(index)
        color = palette[index]
        sprite.putpixel((x, y), (color >> 16, (color >> 8) & 255, color & 255, 255 if index else 0))
    rows.append(row)
sprite.save(assets / "body.png")

template = (ROOT / "main.be.in").read_text()
source = template.replace("@PALETTE@", str(palette)).replace(
    "@ROWS@", "[\n" + ",\n".join("      " + str(row) for row in rows) + "\n    ]"
)
(ROOT / "main.be").write_text(source)
print("Generated assets/body.png and main.be")
