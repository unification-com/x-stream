package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Streams: []StreamExport{},
	}
}

func NewGenesisState(streams []StreamExport, params Params) *GenesisState {
	return &GenesisState{
		Params:  params,
		Streams: streams,
	}
}

// Validate performs full validation of the genesis state: params validity,
// each stream's address/denom/amount/flow-rate sanity, and uniqueness of the
// (sender, receiver, denom) triple. Runs before InitGenesis so a chain
// authoring an invalid genesis fails at boot with a clear error rather than
// panicking deeper in the InitGenesis state-mutation path.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(gs.Streams))
	for i, exp := range gs.Streams {
		if _, err := sdk.AccAddressFromBech32(exp.Sender); err != nil {
			return fmt.Errorf("stream[%d]: invalid sender address %q: %w", i, exp.Sender, err)
		}
		if _, err := sdk.AccAddressFromBech32(exp.Receiver); err != nil {
			return fmt.Errorf("stream[%d]: invalid receiver address %q: %w", i, exp.Receiver, err)
		}
		if exp.Sender == exp.Receiver {
			return fmt.Errorf("stream[%d]: sender and receiver are the same address", i)
		}

		if err := exp.Stream.Deposit.Validate(); err != nil {
			return fmt.Errorf("stream[%d]: invalid deposit %s: %w", i, exp.Stream.Deposit, err)
		}
		if exp.Stream.Deposit.IsNegative() {
			return fmt.Errorf("stream[%d]: negative deposit %s", i, exp.Stream.Deposit)
		}

		if exp.Stream.FlowRate <= 0 {
			return fmt.Errorf("stream[%d]: flow rate must be > 0, got %d", i, exp.Stream.FlowRate)
		}

		key := exp.Sender + "|" + exp.Receiver + "|" + exp.Stream.Deposit.Denom
		if _, dup := seen[key]; dup {
			return fmt.Errorf("stream[%d]: duplicate (sender=%s, receiver=%s, denom=%s)",
				i, exp.Sender, exp.Receiver, exp.Stream.Deposit.Denom)
		}
		seen[key] = struct{}{}
	}

	return nil
}
