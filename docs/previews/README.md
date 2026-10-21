# Clock previews

These GIFs are generated from the real headless AWTRIX runtime, enlarged 6× with nearest-neighbor scaling. Run `make clock-check LOCAL_CONFIG=` and then `make previews` to refresh them. They illustrate the apps; raw-pixel golden snapshots in each animation's `check.json` provide deterministic regression checks at 0, 500 and 1000 ms.

Sprite PNGs alongside the Berry sources are lossless reconstructions of the repository's original palette/index artwork. Edit those images and run `make sprites`; generated Berry arrays remain embedded so deployment needs no external assets.
