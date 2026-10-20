# Architecture

The project supports two independent types of apps:

1. **Berry on the TC002:** local logic, rendering, and interaction with the device.
2. **Go on a VPS:** fetching data from APIs, processing it, and publishing it to the clock over HTTP or MQTT.

A simple screen or notification can be generated entirely in Go. For custom animation or interaction, a Berry app receives data and handles rendering on the clock.

The current implementation provides a minimal AWTRIX NG HTTP notification client. MQTT, authentication, specific data sources, and Berry script upload will be added as needed by individual apps.

Each Go command lives in `cmd/<name>`, with reusable logic in `internal/`. A shared Dockerfile builds the command selected by the `APP` build argument. Independent services get separate entries in Compose.

`worker` is a placeholder demonstrating periodic execution and graceful shutdown on SIGTERM. It does not fetch data or publish anything to the clock yet.
