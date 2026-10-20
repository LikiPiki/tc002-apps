# Kicking Seagull

A screenshot-derived pixel seagull lying on its back and waving its orange feet. Runs entirely on a TC002 with AWTRIX NG and a 52 x 16 display.

The body is cropped, masked, and reduced to a 48 x 12 sprite with a small palette. The original legs are removed from the sprite and drawn separately with alternating kicks. The screenshot's text, username, avatar, and frame are not included.

## Files

- `main.be`: ready-to-install Berry app, with the sprite embedded.
- `main.be.in`: editable animation template.
- `assets/reference.png`: bird-only crop from the user-provided screenshot.
- `assets/body.png`: generated transparent body sprite.
- `build_sprite.py`: converts the reference and generates `main.be`.

No network, server, downloaded icons, or input data are needed. Only `main.be` is uploaded to the clock. The app stays in rotation for 15 seconds per turn.

## Preview and install

From the repository root, run `make dev`, then upload:

```sh
curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/script/seagull \
  -H 'Content-Type: text/plain' \
  --data-binary @apps/clock/seagull/main.be

curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/active \
  -H 'Content-Type: application/json' \
  -d '{"name":"seagull","fast":true}'
```

Check `"error":null` in the upload response and inspect the live preview at http://localhost:8080. You can also paste `main.be` into the Scripts editor. For a physical TC002, use its URL instead; add curl credentials if authentication is enabled.

## Rebuild the sprite

Requires Python 3 and Pillow (`python3 -m pip install Pillow` in your preferred virtual environment). From the repository root:

```sh
python3 apps/clock/seagull/build_sprite.py
```

Edit `main.be.in` for animation changes and rerun the generator. The reference was supplied by the repository owner; the original artwork's author and license have not been identified.
