# Hello World

A minimal on-device Berry app for a TC002 running AWTRIX NG. It draws `Hello` in green on the first line and `World` in white on the second line of the 52 x 16 display.

No server, network requests, icons, configuration, or input data are required. AWTRIX calls `draw()` while the app is visible; the script redraws both lines each frame.

## Try it locally

From the repository root:

```sh
make dev
```

1. Open http://localhost:8080 and go to **Scripts**.
2. Create an app named `hello-world`.
3. Paste the contents of `main.be` into the editor and save.
4. Choose **Show on the clock** to view the two-line greeting in the live preview.

Alternatively, upload the source through the HTTP API:

```sh
curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/script/hello-world \
  -H 'Content-Type: text/plain' \
  --data-binary @apps/clock/hello-world/main.be

curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/active \
  -H 'Content-Type: application/json' \
  -d '{"name":"hello-world","fast":true}'
```

Check that the upload response has `"error":null`. HTTP 200 alone does not guarantee that a Berry script compiled and ran successfully. Runtime errors are also reported in the Scripts tab and `GET /api/v1/apps`.

## Install on the TC002

Use the same Scripts-tab steps in your clock's web UI, or replace `http://localhost:8080` in the HTTP examples with your clock's address. If web authentication is enabled, pass credentials to curl with `-u username:password`.

The installed script is stored on AWTRIX and joins the app rotation. It remains installed after `make dev-down` locally because development data is kept in a Docker volume.

## Compatibility

Targets the TC002's 52 x 16 display and uses the standard AWTRIX NG `clear()` and `text()` Berry drawing functions. The second line requires a display at least 16 pixels high; this layout does not target 8-row clocks.

See the [AWTRIX NG scripting guide](https://blueforcer.github.io/awtrix-ng/tc002/guides/scripting/) for the app lifecycle.
