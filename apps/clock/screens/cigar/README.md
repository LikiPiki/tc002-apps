# Smoking Cigar

An animated pixel-art cigar for the TC002's 52 x 16 display, running entirely in AWTRIX NG's Berry interpreter.

- Brown wrapper with diagonal seams and a gold-and-red band.
- Pulsing orange embers and a small ash cap.
- Three curling smoke wisps that fade as they rise, plus occasional sparks.
- A 15-second slot in the app rotation.

The animation uses `now_ms()` rather than frame counters, so its speed does not depend on the rendering frame rate. It does not need a server, network access, downloaded assets, configuration, or input data.

## Install and preview

Run `make dev` from the repository root, open http://localhost:8080, and create a script named `cigar` in the Scripts tab. Paste `main.be`, save, and choose **Show on the clock**.

Alternatively, from the repository root:

```sh
curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/script/cigar \
  -H 'Content-Type: text/plain' \
  --data-binary @apps/clock/screens/cigar/main.be

curl --fail-with-body -X PUT http://localhost:8080/api/v1/apps/active \
  -H 'Content-Type: application/json' \
  -d '{"name":"cigar","fast":true}'
```

Check that the installation response has `"error":null`; runtime errors also appear in the Scripts tab and `GET /api/v1/apps`. To install on a physical TC002, replace the local URL with the clock's address. If authentication is enabled, provide credentials with `curl -u username:password`.

## Compatibility

Requires AWTRIX NG with Berry scripting, the `math` module, and a 52 x 16 display. The layout uses fixed pixel coordinates and does not target 8-row clocks. No icons or sound are used.
