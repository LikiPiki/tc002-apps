# Swimming Mackerel

A golden pixel mackerel adapted from a user-provided photo. The original diagonal fish is rotated horizontally and reduced to a 44 x 11 sprite with a small palette for the TC002's 52 x 16 display.

The fish bobs gently and bends with a traveling wave that is strongest at its tail. A small bubble rises beside it. The animation uses `now_ms()` for frame-rate-independent timing and takes a 15-second turn in the app rotation.

Only `main.be` is deployed. `sprite.png` is the editable source for the embedded palette/rows; no external assets, server dependencies, configuration or input data are needed at runtime.

## Preview and install

From the repository root, run `make dev`, then upload:

```sh
curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/script/mackerel \
  -H 'Content-Type: text/plain' \
  --data-binary @apps/clock/screens/mackerel/main.be

curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/active \
  -H 'Content-Type: application/json' \
  -d '{"name":"mackerel","fast":true}'
```

Check `"error":null` in the upload response and view the animation at http://localhost:8080. Alternatively, paste `main.be` into a new script named `mackerel` in the Scripts tab and choose **Show on the clock**.

For a physical TC002, replace the local URL with the clock's address. Add `curl -u username:password` if authentication is enabled. Runtime errors appear in the Scripts tab and `GET /api/v1/apps`.

## Editing and compatibility

Edit `sprite.png` in a pixel editor, retaining binary transparency, and run `make sprites` to regenerate the marked Berry palette/rows. The PNG is a lossless reconstruction of the pre-existing embedded artwork. Edit `main.be` outside those markers to change motion. `make sprites-check` detects stale generated code; `make clock-check APP=mackerel` checks runtime/frame validity and records a GIF for visual inspection.

Requires AWTRIX NG with Berry scripting, the `math` module, and a 52 x 16 display. The layout does not target 8-row clocks.

The photo was supplied by the repository owner; the original photographer and license have not been identified.
