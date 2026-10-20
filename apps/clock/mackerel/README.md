# Swimming Mackerel

A golden pixel mackerel adapted from a user-provided photo. The original diagonal fish is rotated horizontally and reduced to a 44 x 11 sprite with a small palette for the TC002's 52 x 16 display.

The fish bobs gently and bends with a traveling wave that is strongest at its tail. A small bubble rises beside it. The animation uses `now_ms()` for frame-rate-independent timing and takes a 15-second turn in the app rotation.

Only `main.be` is needed. The sprite and palette are embedded directly in the script; there are no external assets, build steps, server dependencies, configuration, or input data.

## Preview and install

From the repository root, run `make dev`, then upload:

```sh
curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/script/mackerel \
  -H 'Content-Type: text/plain' \
  --data-binary @apps/clock/mackerel/main.be

curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/active \
  -H 'Content-Type: application/json' \
  -d '{"name":"mackerel","fast":true}'
```

Check `"error":null` in the upload response and view the animation at http://localhost:8080. Alternatively, paste `main.be` into a new script named `mackerel` in the Scripts tab and choose **Show on the clock**.

For a physical TC002, replace the local URL with the clock's address. Add `curl -u username:password` if authentication is enabled. Runtime errors appear in the Scripts tab and `GET /api/v1/apps`.

## Editing and compatibility

Edit `main.be` directly to change the sprite, palette, or motion. Requires AWTRIX NG with Berry scripting, the `math` module, and a 52 x 16 display. The layout does not target 8-row clocks.

The photo was supplied by the repository owner; the original photographer and license have not been identified.
