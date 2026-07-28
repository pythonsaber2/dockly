#!/usr/bin/env python3
"""Generate Dockly-owned marketing visuals from the real dashboard screenshot."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter, ImageFont

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "assets" / "dockly-dashboard.png"
OUT = ROOT / "assets" / "home"
OUT.mkdir(parents=True, exist_ok=True)

W, H = 1422, 1106
BG = (13, 20, 32, 0)
PANEL = (24, 36, 54, 245)
LINE = (57, 75, 99, 255)
TEXT = (242, 247, 249, 255)
MUTED = (143, 160, 183, 255)
AQUA = (98, 230, 198, 255)
BLUE = (61, 149, 255, 255)
ORANGE = (255, 176, 103, 255)
RED = (255, 123, 132, 255)


def font(size, bold=False, mono=False):
    if mono:
        path = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
    elif bold:
        path = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
    else:
        path = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
    return ImageFont.truetype(path, size)


def save_visual(image, name):
    image.save(OUT / f"{name}.webp", "WEBP", quality=86, method=6)


def rounded_image(img, radius):
    mask = Image.new("L", img.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, *img.size), radius=radius, fill=255)
    out = Image.new("RGBA", img.size)
    out.paste(img.convert("RGBA"), mask=mask)
    return out


def shadowed_paste(canvas, img, xy, blur=28, offset=(0, 22), alpha=150):
    shadow = Image.new("RGBA", canvas.size)
    mask = img.getchannel("A")
    layer = Image.new("RGBA", img.size, (0, 0, 0, alpha))
    layer.putalpha(mask.point(lambda p: p * alpha // 255))
    shadow.alpha_composite(layer, (xy[0] + offset[0], xy[1] + offset[1]))
    shadow = shadow.filter(ImageFilter.GaussianBlur(blur))
    canvas.alpha_composite(shadow)
    canvas.alpha_composite(img, xy)


def card(size, title, rows, accent=AQUA, footer=None):
    im = Image.new("RGBA", size)
    d = ImageDraw.Draw(im)
    d.rounded_rectangle((1, 1, size[0]-2, size[1]-2), radius=22, fill=PANEL, outline=LINE, width=2)
    d.ellipse((25, 28, 39, 42), fill=accent)
    d.text((54, 22), title, font=font(24, True), fill=TEXT)
    y = 76
    for label, value, color in rows:
        d.line((24, y-12, size[0]-24, y-12), fill=(44, 59, 80, 220), width=1)
        d.text((26, y), label, font=font(17), fill=MUTED)
        tw = d.textbbox((0, 0), value, font=font(17, True))[2]
        d.text((size[0]-26-tw, y), value, font=font(17, True), fill=color)
        y += 50
    if footer:
        d.text((26, size[1]-39), footer, font=font(14, mono=True), fill=MUTED)
    return im


def base_scene(scale=.68, angle=-3, pos=(170, 180)):
    canvas = Image.new("RGBA", (W, H), BG)
    shot = Image.open(SOURCE).convert("RGBA")
    shot.thumbnail((int(W*scale), int(H*scale)))
    shot = rounded_image(shot, 20)
    shot = shot.rotate(angle, resample=Image.Resampling.BICUBIC, expand=True)
    shadowed_paste(canvas, shot, pos, blur=34, offset=(0, 32), alpha=165)
    return canvas


def save_pipeline():
    c = base_scene(.76, -3.5, (120, 210))
    labels = [("01", "Clone repository", "main@8c1e4a2f", BLUE), ("02", "Build image", "6.8s", ORANGE), ("03", "Health check", "200 OK", AQUA), ("04", "Release", "Running", AQUA)]
    y = 90
    for n, title, value, color in labels:
        im = Image.new("RGBA", (365, 104)); d = ImageDraw.Draw(im)
        d.rounded_rectangle((1,1,363,102), radius=20, fill=PANEL, outline=LINE, width=2)
        d.text((22, 20), n, font=font(17, True, True), fill=color)
        d.text((72, 16), title, font=font(19, True), fill=TEXT)
        d.text((72, 52), value, font=font(15, mono=True), fill=MUTED)
        shadowed_paste(c, im, (930, y), blur=18, offset=(0, 14), alpha=135)
        y += 130
    save_visual(c, "feat-deploy")


def save_health():
    c = base_scene(.72, 3.2, (255, 230))
    checks = card((430, 300), "Service health", [("api-production", "Healthy", AQUA), ("web", "Healthy", AQUA), ("worker", "Healthy", AQUA), ("Last probe", "128 ms", BLUE)], AQUA, "GET /health  ·  every 15s")
    shadowed_paste(c, checks, (75, 82), blur=26, offset=(0, 24), alpha=165)
    badge = card((295, 132), "Monitor", [("Uptime", "99.99%", AQUA)], BLUE)
    shadowed_paste(c, badge, (1010, 760), blur=22, offset=(0, 18), alpha=140)
    save_visual(c, "feat-health")


def save_rollback():
    c = base_scene(.70, -2.2, (110, 180))
    timeline = Image.new("RGBA", (470, 380)); d = ImageDraw.Draw(timeline)
    d.rounded_rectangle((1,1,468,378), radius=24, fill=PANEL, outline=LINE, width=2)
    d.text((28, 22), "Deployment history", font=font(24, True), fill=TEXT)
    d.line((51, 82, 51, 320), fill=LINE, width=3)
    entries = [("8c1e4a2f", "Live", AQUA), ("37bd911c", "Previous", BLUE), ("a4e102f9", "Rollback target", ORANGE)]
    y = 92
    for commit, label, color in entries:
        d.ellipse((41, y, 61, y+20), fill=color)
        d.text((82, y-5), commit, font=font(18, True, True), fill=TEXT)
        d.text((82, y+28), label, font=font(16), fill=MUTED)
        y += 90
    btn = Image.new("RGBA", (260, 72)); bd=ImageDraw.Draw(btn)
    bd.rounded_rectangle((0,0,259,71), radius=36, fill=AQUA)
    bd.text((49,20), "Roll back release", font=font(18, True), fill=(12,40,35,255))
    timeline.alpha_composite(btn, (180, 287))
    shadowed_paste(c, timeline, (870, 110), blur=30, offset=(0, 25), alpha=170)
    save_visual(c, "feat-rollback")


def save_api():
    c = base_scene(.68, 2.8, (290, 255))
    term = Image.new("RGBA", (700, 310)); d = ImageDraw.Draw(term)
    d.rounded_rectangle((1,1,698,308), radius=22, fill=(7, 13, 22, 252), outline=LINE, width=2)
    d.ellipse((24,22,38,36), fill=RED); d.ellipse((47,22,61,36), fill=ORANGE); d.ellipse((70,22,84,36), fill=AQUA)
    d.text((110, 18), "deploy.sh", font=font(15, mono=True), fill=MUTED)
    lines = [("$", AQUA, " curl -X POST \\", TEXT), ("", TEXT, "  -H 'Authorization: Bearer ••••••' \\", MUTED), ("", TEXT, "  http://dockly:8080/api/apps/api/deploy", MUTED), ("→", BLUE, " deployment queued  202 Accepted", TEXT)]
    y=78
    for prompt, pc, text, tc in lines:
        d.text((30,y), prompt, font=font(18, True, True), fill=pc)
        d.text((58,y), text, font=font(16, mono=True), fill=tc); y+=47
    shadowed_paste(c, term, (65, 90), blur=30, offset=(0, 25), alpha=175)
    hook = card((350, 190), "Webhook", [("Trigger", "push", BLUE), ("Response", "202", AQUA)], ORANGE, "X-Dockly-Token: ••••••")
    shadowed_paste(c, hook, (980, 715), blur=24, offset=(0, 20), alpha=145)
    save_visual(c, "feat-api")


if __name__ == "__main__":
    save_pipeline(); save_health(); save_rollback(); save_api()
    for p in sorted(OUT.glob("feat-*.webp")):
        print(p.relative_to(ROOT), p.stat().st_size)
