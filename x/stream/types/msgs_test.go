package types_test

import (
	"fmt"
	"testing"

	mathmod "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/unification-com/x-stream/x/stream/types"
)

// TestValidateBasic_IBCDenom proves the stateless message validation accepts an IBC
// voucher denom (ibc/<hash>) on every stream message — nothing restricts streams to the
// bond/native denom. The full end-to-end escrow/claim/refund proof lives in the keeper
// test TestMsgServerStreamLifecycle_IBCDenom.
func TestValidateBasic_IBCDenom(t *testing.T) {
	r := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	s := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	// A real-shape IBC voucher denom: "ibc/" + 64 uppercase hex (the denom-trace hash).
	const ibcDenom = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	require.NoError(t, sdk.ValidateDenom(ibcDenom))

	// deposit/flow = 10000/100 = 100s, comfortably inside the 60s..10y duration bounds.
	require.NoError(t, types.MsgCreateStream{Sender: s, Receiver: r, Deposit: sdk.NewInt64Coin(ibcDenom, 10000), FlowRate: 100}.ValidateBasic())
	require.NoError(t, types.MsgClaimStream{Sender: s, Receiver: r, Denom: ibcDenom}.ValidateBasic())
	require.NoError(t, types.MsgCancelStream{Sender: s, Receiver: r, Denom: ibcDenom}.ValidateBasic())
	require.NoError(t, types.MsgUpdateFlowRate{Sender: s, Receiver: r, FlowRate: 100, Denom: ibcDenom}.ValidateBasic())
	require.NoError(t, types.MsgTopUpDeposit{Sender: s, Receiver: r, Deposit: sdk.NewInt64Coin(ibcDenom, 10000)}.ValidateBasic())
}

//	MsgCreateStream{}

func TestMsgCreateStream_Route(t *testing.T) {
	msg := types.MsgCreateStream{}
	require.Equal(t, types.ModuleName, msg.Route())
}

func TestMsgCreateStream_Type(t *testing.T) {
	msg := types.MsgCreateStream{}
	require.Equal(t, types.CreateStreamAction, msg.Type())
}

func TestMsgCreateStream_ValidateBasic(t *testing.T) {
	s := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	r := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	tests := []struct {
		deposit    sdk.Coin
		flowRate   int64
		receiver   sdk.AccAddress
		sender     sdk.AccAddress
		expectPass bool
	}{
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(10000)), 100, r, s, true},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(0)), 100, r, s, false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(10000)), 0, r, s, false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(10000)), 100, sdk.AccAddress{}, s, false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(10000)), 100, r, sdk.AccAddress{}, false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(100)), 100, r, s, false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(10000)), 100, r, r, false},
		// Malformed denoms rejected at ValidateBasic
		{sdk.Coin{Denom: "1invalid", Amount: mathmod.NewIntFromUint64(10000)}, 100, r, s, false},  // starts with digit
		{sdk.Coin{Denom: "x", Amount: mathmod.NewIntFromUint64(10000)}, 100, r, s, false},         // too short
		{sdk.Coin{Denom: "", Amount: mathmod.NewIntFromUint64(10000)}, 100, r, s, false},          // empty
		{sdk.Coin{Denom: "has space", Amount: mathmod.NewIntFromUint64(10000)}, 100, r, s, false}, // space not allowed
	}

	for i, tc := range tests {
		msg := types.NewMsgCreateStream(
			tc.deposit,
			tc.flowRate,
			tc.receiver,
			tc.sender,
		)

		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

//	MsgClaimStream{}

func TestMsgClaimStream_Route(t *testing.T) {
	msg := types.MsgClaimStream{}
	require.Equal(t, types.ModuleName, msg.Route())
}

func TestMsgClaimStream_Type(t *testing.T) {
	msg := types.MsgClaimStream{}
	require.Equal(t, types.ClaimStreamAction, msg.Type())
}

func TestMsgClaimStream_ValidateBasic(t *testing.T) {
	tests := []struct {
		sender     sdk.AccAddress
		receiver   sdk.AccAddress
		expectPass bool
	}{
		{sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), true},
		{sdk.AccAddress{}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		{sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress{}, false},
	}

	for i, tc := range tests {
		msg := types.NewMsgClaimStream(
			tc.receiver,
			tc.sender,
			sdk.DefaultBondDenom,
		)

		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

//	MsgTopUpDeposit{}

func TestMsgTopUpDeposit_Route(t *testing.T) {
	msg := types.MsgTopUpDeposit{}
	require.Equal(t, types.ModuleName, msg.Route())
}

func TestMsgTopUpDeposit_Type(t *testing.T) {
	msg := types.MsgTopUpDeposit{}
	require.Equal(t, types.TopUpDepositAction, msg.Type())
}

func TestMsgTopUpDeposit_ValidateBasic(t *testing.T) {
	tests := []struct {
		deposit    sdk.Coin
		sender     sdk.AccAddress
		receiver   sdk.AccAddress
		expectPass bool
	}{
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(100)), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), true},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(100)), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress{}, false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(100)), sdk.AccAddress{}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		{sdk.NewCoin(sdk.DefaultBondDenom, mathmod.NewIntFromUint64(0)), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		// Malformed denoms rejected at ValidateBasic
		{sdk.Coin{Denom: "1invalid", Amount: mathmod.NewIntFromUint64(100)}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		{sdk.Coin{Denom: "", Amount: mathmod.NewIntFromUint64(100)}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		{sdk.Coin{Denom: "has space", Amount: mathmod.NewIntFromUint64(100)}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
	}

	for i, tc := range tests {
		msg := types.NewMsgTopUpDeposit(
			tc.receiver,
			tc.sender,
			tc.deposit,
		)

		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

//	MsgUpdateFlowRate{}

func TestMsgUpdateFlowRate_Route(t *testing.T) {
	msg := types.MsgUpdateFlowRate{}
	require.Equal(t, types.ModuleName, msg.Route())
}

func TestMsgUpdateFlowRate_Type(t *testing.T) {
	msg := types.MsgUpdateFlowRate{}
	require.Equal(t, types.UpdateFlowRateAction, msg.Type())
}

func TestMsgUpdateFlowRate_ValidateBasic(t *testing.T) {
	tests := []struct {
		flowRate   int64
		sender     sdk.AccAddress
		receiver   sdk.AccAddress
		expectPass bool
	}{
		{1, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), true},
		{1, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress{}, false},
		{1, sdk.AccAddress{}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		{0, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
	}

	for i, tc := range tests {
		msg := types.NewMsgUpdateFlowRate(
			tc.receiver,
			tc.sender,
			tc.flowRate,
			sdk.DefaultBondDenom,
		)

		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

//	MsgCancelStream{}

func TestMsgCancelStream_Route(t *testing.T) {
	msg := types.MsgCancelStream{}
	require.Equal(t, types.ModuleName, msg.Route())
}

func TestMsgCancelStream_Type(t *testing.T) {
	msg := types.MsgCancelStream{}
	require.Equal(t, types.CancelStreamAction, msg.Type())
}

func TestMsgCancelStream_ValidateBasic(t *testing.T) {
	tests := []struct {
		receiver   sdk.AccAddress
		sender     sdk.AccAddress
		expectPass bool
	}{
		{sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), true},
		{sdk.AccAddress{}, sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), false},
		{sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()), sdk.AccAddress{}, false},
	}

	for i, tc := range tests {
		msg := types.NewMsgCancelStream(
			tc.receiver,
			tc.sender,
			sdk.DefaultBondDenom,
		)

		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

// MsgUpdateParams{}

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		name            string
		msgUpdateParams types.MsgUpdateParams
		expFail         bool
		expError        string
	}{
		{
			"valid msg",
			types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    types.DefaultParams(),
			},
			false,
			"",
		},
		{
			"negative validator fee",
			types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params: types.Params{
					ValidatorFee: mathmod.LegacyNewDecWithPrec(-1, 2),
				},
			},
			true,
			"validator fee cannot be negative:",
		},
		{
			"validator fee > MaxValidatorFee (10%)",
			types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params: types.Params{
					ValidatorFee: mathmod.LegacyNewDecWithPrec(11, 2),
				},
			},
			true,
			"validator fee cannot exceed",
		},
		{
			"validator fee = 100% rejected (above MaxValidatorFee cap)",
			types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params: types.Params{
					ValidatorFee: mathmod.LegacyOneDec(),
				},
			},
			true,
			"validator fee cannot exceed",
		},
		{
			"validator fee exactly at cap accepted",
			types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params: types.Params{
					ValidatorFee: types.MaxValidatorFee,
				},
			},
			false,
			"",
		},
		{
			"nil validator fee",
			types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params: types.Params{
					ValidatorFee: mathmod.LegacyDec{},
				},
			},
			true,
			"validator fee cannot be nil",
		},
		{
			"Invalid authority",
			types.MsgUpdateParams{
				Authority: "invalid",
				Params:    types.DefaultParams(),
			},
			true,
			"invalid authority address",
		},
	}

	for _, tc := range tests {
		err := tc.msgUpdateParams.ValidateBasic()
		if tc.expFail {
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expError)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestMsgCreateStreamGetSignBytes(t *testing.T) {
	sender := sdk.AccAddress("addr1")
	receiver := sdk.AccAddress("addr2")
	deposit := sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000)
	msg := types.NewMsgCreateStream(deposit, 1, receiver, sender)
	pc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	res, err := pc.MarshalAminoJSON(msg)
	require.NoError(t, err)
	expected := fmt.Sprintf(
		`{"type":"stream/MsgCreateStream","value":{"deposit":{"amount":"1000","denom":%q},"flow_rate":"1","receiver":%q,"sender":%q}}`,
		sdk.DefaultBondDenom, receiver.String(), sender.String())
	require.Equal(t, expected, string(res))
}

func TestMsgClaimStreamGetSignBytes(t *testing.T) {
	sender := sdk.AccAddress("addr1")
	receiver := sdk.AccAddress("addr2")
	msg := types.NewMsgClaimStream(receiver, sender, sdk.DefaultBondDenom)
	pc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	res, err := pc.MarshalAminoJSON(msg)
	require.NoError(t, err)
	expected := fmt.Sprintf(
		`{"type":"stream/MsgClaimStream","value":{"denom":%q,"receiver":%q,"sender":%q}}`,
		sdk.DefaultBondDenom, receiver.String(), sender.String())
	require.Equal(t, expected, string(res))
}

func TestMsgTopUpDepositGetSignBytes(t *testing.T) {
	sender := sdk.AccAddress("addr1")
	receiver := sdk.AccAddress("addr2")
	deposit := sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000)
	msg := types.NewMsgTopUpDeposit(receiver, sender, deposit)
	pc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	res, err := pc.MarshalAminoJSON(msg)
	require.NoError(t, err)
	expected := fmt.Sprintf(
		`{"type":"stream/MsgTopUpDeposit","value":{"deposit":{"amount":"1000","denom":%q},"receiver":%q,"sender":%q}}`,
		sdk.DefaultBondDenom, receiver.String(), sender.String())
	require.Equal(t, expected, string(res))
}

func TestMsgUpdateFlowRateGetSignBytes(t *testing.T) {
	sender := sdk.AccAddress("addr1")
	receiver := sdk.AccAddress("addr2")
	msg := types.NewMsgUpdateFlowRate(receiver, sender, 1, sdk.DefaultBondDenom)
	pc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	res, err := pc.MarshalAminoJSON(msg)
	require.NoError(t, err)
	expected := fmt.Sprintf(
		`{"type":"stream/MsgUpdateFlowRate","value":{"denom":%q,"flow_rate":"1","receiver":%q,"sender":%q}}`,
		sdk.DefaultBondDenom, receiver.String(), sender.String())
	require.Equal(t, expected, string(res))
}

func TestMsgCancelStreamGetSignBytes(t *testing.T) {
	sender := sdk.AccAddress("addr1")
	receiver := sdk.AccAddress("addr2")
	msg := types.NewMsgCancelStream(receiver, sender, sdk.DefaultBondDenom)
	pc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	res, err := pc.MarshalAminoJSON(msg)
	require.NoError(t, err)
	expected := fmt.Sprintf(
		`{"type":"stream/MsgCancelStream","value":{"denom":%q,"receiver":%q,"sender":%q}}`,
		sdk.DefaultBondDenom, receiver.String(), sender.String())
	require.Equal(t, expected, string(res))
}
