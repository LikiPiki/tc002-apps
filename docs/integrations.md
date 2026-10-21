# Screen/server integrations

The [service-status integration](integrations/service-status.md) is implemented and tested end to end. Integration registry/config wiring remains experimental: there is no accepted `[integrations]` TOML section or automatic component matching yet.

An integration is one user-facing feature that may span multiple runtime components. Runtime directories answer **where code executes**; an integration ID answers **which components work together**.

## Current state

The bundled Berry animations and display policy run autonomously. `status-provider` checks a configured HTTP endpoint and supplies `service-status`; `worker` remains a periodic-process example. There is no cross-runtime registry, automatic pairing, or MQTT transport yet. Existing TOML selections choose artifacts independently.

## Convention for a real integration

Give the feature a stable ID, such as `weather`, and document its wiring together:

```text
weather (illustrative future integration)
├── consumer: apps/clock/screens/weather/main.be
├── producer: apps/server/cmd/weather-provider/main.go
└── contract: data format, delivery channel, freshness, and failure behavior
```

The component names do not have to match. One producer may feed several screens; a feature may also use a headless receiver/provider on the clock. A server can push a native AWTRIX app with no custom Berry screen, and an autonomous Berry app may need no server at all.

When adding the first integration, create `docs/integrations/<id>.md` and link it from every participating component's README. The integration document should specify:

1. Component IDs/paths and their responsibilities.
2. The delivery channel and address or topic, defined once rather than hardcoded independently.
3. Payload version, fields, units, and an example; add a schema when it helps validation.
4. Update frequency, freshness/expiry, and behavior before the first value or after a disconnect.
5. Which component owns each action. Data producers should not compete with display-policy over power.
6. A local end-to-end check using the real producer and consumer.

Runtime coupling must go through that contract, not imports between server and tooling internals or shared directories containing unrelated Go and Berry files.

## Future configuration wiring

The first implementation should extend the existing project TOML with integration-to-component mappings and transport settings. For example, a future `integrations.weather` entry could reference its screens, server apps, and shared MQTT topic. This is a design direction, not a config section accepted by the current loader; add it alongside a working integration and its tests.

Deployment utilities can then use the mapping to validate references and deliver the same channel settings to Go and Berry components. The broker configuration belongs in one place. App-specific rendering preferences remain in `clock.screens.<name>`, and server-runtime settings remain in their server namespace.

Keep the [runtime boundaries](architecture.md#runtime-boundaries) intact even when components are deployed or tested together.
