# Service Status

A 52×16 screen showing a short service label and `UP`, `DOWN`, `STALE` or `NO DATA`. It consumes the latest `status` object from AWTRIX script data; it does not perform network requests or manage display power.

Install with `make clock-deploy APP=service-status`, then configure and run the Go provider. See the [service-status integration](../../../../docs/integrations/service-status.md) for configuration, the versioned payload contract, expiry rules and failure behavior.

Run `make clock-check APP=service-status` for a runtime/nonempty-frame check and `make status-check` for the real producer/consumer end-to-end scenarios.
