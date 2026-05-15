package types_test

import (
	"testing"
	"time"

	mathmod "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/unification-com/x-stream/x/stream/types"
)

// genAddr produces a fresh prefix-correct bech32 string for use in genesis fixtures.
func genAddr(t *testing.T) string {
	t.Helper()
	return sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
}

func validStream(deposit sdk.Coin, flowRate int64) types.Stream {
	return types.Stream{
		Deposit:         deposit,
		FlowRate:        flowRate,
		LastOutflowTime: time.Now(),
		DepositZeroTime: time.Now().Add(time.Hour),
		Cancellable:     true,
	}
}

func TestGenesisState_Validate(t *testing.T) {
	sender1 := genAddr(t)
	receiver1 := genAddr(t)
	sender2 := genAddr(t)
	receiver2 := genAddr(t)

	tests := []struct {
		desc     string
		genState *types.GenesisState
		expErr   bool
		errSub   string
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			expErr:   false,
		},
		{
			desc: "valid: default params, empty streams",
			genState: &types.GenesisState{
				Params:  types.DefaultParams(),
				Streams: []types.StreamExport{},
			},
			expErr: false,
		},
		{
			desc: "valid: two streams between two pairs",
			genState: &types.GenesisState{
				Params: types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(1, 2)},
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(100_000_000)), 123)},
					{Sender: sender2, Receiver: receiver2, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(200_000_000)), 321)},
				},
			},
			expErr: false,
		},
		{
			desc: "valid: same pair, different denoms",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin("denoma", mathmod.NewIntFromUint64(1000)), 1)},
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin("denomb", mathmod.NewIntFromUint64(1000)), 1)},
				},
			},
			expErr: false,
		},
		{
			desc: "invalid: nil validator fee",
			genState: &types.GenesisState{
				Params:  types.Params{ValidatorFee: mathmod.LegacyDec{}},
				Streams: []types.StreamExport{},
			},
			expErr: true,
			errSub: "validator fee cannot be nil",
		},
		{
			desc: "invalid: validator fee > MaxValidatorFee",
			genState: &types.GenesisState{
				Params:  types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(11, 2)},
				Streams: []types.StreamExport{},
			},
			expErr: true,
			errSub: "validator fee cannot exceed",
		},
		{
			desc: "invalid: negative validator fee",
			genState: &types.GenesisState{
				Params:  types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(-1, 2)},
				Streams: []types.StreamExport{},
			},
			expErr: true,
			errSub: "validator fee cannot be negative",
		},
		// Per-stream validation
		{
			desc: "invalid: malformed sender address",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: "not-a-bech32", Receiver: receiver1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(1000)), 1)},
				},
			},
			expErr: true,
			errSub: "invalid sender",
		},
		{
			desc: "invalid: malformed receiver address",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: "garbage", Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(1000)), 1)},
				},
			},
			expErr: true,
			errSub: "invalid receiver",
		},
		{
			desc: "invalid: sender == receiver",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: sender1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(1000)), 1)},
				},
			},
			expErr: true,
			errSub: "sender and receiver are the same",
		},
		{
			desc: "invalid: negative deposit amount",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: mathmod.NewInt(-1)}, 1)},
				},
			},
			expErr: true,
		},
		{
			desc: "invalid: malformed deposit denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.Coin{Denom: "1bad", Amount: mathmod.NewIntFromUint64(1000)}, 1)},
				},
			},
			expErr: true,
		},
		{
			desc: "invalid: zero flow rate",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(1000)), 0)},
				},
			},
			expErr: true,
			errSub: "flow rate must be > 0",
		},
		{
			desc: "invalid: negative flow rate",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(1000)), -1)},
				},
			},
			expErr: true,
			errSub: "flow rate must be > 0",
		},
		{
			desc: "invalid: duplicate (sender, receiver, denom)",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Streams: []types.StreamExport{
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(1000)), 1)},
					{Sender: sender1, Receiver: receiver1, Stream: validStream(sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(2000)), 1)},
				},
			},
			expErr: true,
			errSub: "duplicate",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.expErr {
				require.Error(t, err)
				if tc.errSub != "" {
					require.Contains(t, err.Error(), tc.errSub)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
