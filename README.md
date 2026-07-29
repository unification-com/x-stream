# x-stream

[![Latest Release](https://img.shields.io/github/v/release/unification-com/x-stream?display_name=tag)](https://github.com/unification-com/x-stream/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/unification-com/x-stream)](https://goreportcard.com/report/github.com/unification-com/x-stream)
[![Join the chat at https://discord.com/channels/725618617525207042](https://img.shields.io/discord/725618617525207042?label=discord)](https://discord.com/channels/725618617525207042)

A Cosmos SDK module for recurring payments between two addresses on a single denomination. One stream per `(sender, receiver, denom)` triple, with a configurable per-second flow rate, top-up + claim + cancel + flow-rate-update lifecycle, and a default 1% validator fee on each claim.

Extracted from [unification-com/mainchain](https://github.com/unification-com/mainchain) where the module previously lived at `x/stream/`. Designed to be portable into any Cosmos SDK chain.

## Status

**Early release.** Mainchain is the reference integration. The latest tag is shown in the badge at the top of this README.

## Compatibility

|            | Version     |
|------------|-------------|
| Cosmos SDK | `v0.54.x`   |
| CometBFT   | `v0.39.x`   |
| Go         | `>= 1.26.5` |

## Quick integration

From your chain's repo root, pull in the latest tagged release:

```sh
go get github.com/unification-com/x-stream@latest
```

To pin a specific version (recommended for production chains), substitute the tag:

```sh
go get github.com/unification-com/x-stream@vX.Y.Z
```

Inside your chain's `app.go`:

```go
import (
    "github.com/unification-com/x-stream/x/stream"
    streamkeeper "github.com/unification-com/x-stream/x/stream/keeper"
    streamtypes "github.com/unification-com/x-stream/x/stream/types"
)

// Add the store key:
keys := storetypes.NewKVStoreKeys(
    // ... your existing store keys
    streamtypes.StoreKey,
)

// Add the module account permissions:
maccPerms := map[string][]string{
    // ... your existing module accounts
    streamtypes.ModuleName: nil,
}

// Construct the keeper:
app.StreamKeeper = streamkeeper.NewKeeper(
    keys[streamtypes.StoreKey],
    app.BankKeeper,
    app.AccountKeeper,
    appCodec,
    authtypes.FeeCollectorName,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// Register the module with the module manager:
app.ModuleManager = module.NewManager(
    // ... your existing modules
    stream.NewAppModule(appCodec, app.StreamKeeper, app.AccountKeeper, app.BankKeeper),
)
```

See [mainchain's `app/app.go`](https://github.com/unification-com/mainchain) for a complete reference integration.

## Messages

| Message             | Purpose                                                                  |
|---------------------|--------------------------------------------------------------------------|
| `MsgCreateStream`   | Open a new stream with an initial deposit and flow rate.                 |
| `MsgTopUpDeposit`   | Sender adds to a stream's deposit.                                       |
| `MsgClaimStream`    | Receiver withdraws accrued payout. 1% default validator fee applied.     |
| `MsgUpdateFlowRate` | Sender re-tunes the flow rate.                                           |
| `MsgCancelStream`   | Sender cancels (unless `cancellable=false`). Remaining deposit refunded. |
| `MsgUpdateParams`   | Gov-authority updates module params (e.g. `ValidatorFee`).               |

## State

Streams are keyed by `(receiver, sender, denom)`. Each stream holds:

- `deposit` — total amount the sender has deposited
- `flow_rate` — units of `deposit.denom` per second
- `last_outflow_time` — timestamp of the most recent claim
- `deposit_zero_time` — timestamp when the deposit will be exhausted at the current flow rate
- `cancellable` — whether `MsgCancelStream` is permitted (default `true`)

## Building

```sh
go build ./...
go test ./...
```

Make targets for the additional CI gates:

```sh
make lint                      # gofmt -s check + golangci-lint v2
make fmt                       # apply gofmt -s in place
make test-sim-simple           # single-seed 500-block sim (~35s)
make test-sim-multi-seed-short # 3 seeds × 50 blocks determinism check (~20s)
make test-sim-nondeterminism   # 3 seeds × 100 blocks × 3 runs determinism (~5min)
```

## Proto regeneration

```sh
make proto-all
```

Requires Docker (uses the same buf-based proto chain as upstream Cosmos SDK).

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) for attribution.
