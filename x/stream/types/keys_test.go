package types_test

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"

	"github.com/unification-com/x-stream/x/stream/types"
)

func TestAddressFromStreamsStore(t *testing.T) {
	receiverAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	require.Equal(t, 20, len(receiverAddr))

	senderAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	require.Equal(t, 20, len(senderAddr))

	denom := sdk.DefaultBondDenom
	key := types.GetStreamKey(receiverAddr, senderAddr, denom)

	expectedLen := len(types.StreamKeyPrefix) +
		len(address.MustLengthPrefix(receiverAddr)) +
		len(address.MustLengthPrefix(senderAddr)) +
		1 + len(denom) // 1-byte length prefix + raw denom
	require.Len(t, key, expectedLen)

	r, s, d := types.AddressesFromStreamKey(key)

	require.Equal(t, receiverAddr, r)
	require.Equal(t, senderAddr, s)
	require.Equal(t, denom, d)
}

// TestAddressFromStreamsStore_IBCDenom proves the stream store key round-trips an
// IBC voucher denom (ibc/<hash>, ~68 bytes) without loss. The denom is single-byte
// length-prefixed, and an IBC denom is well inside the 128-char sdk.Coin bound, so
// neither the primary key nor the sender index is limited to short/native denoms.
func TestAddressFromStreamsStore_IBCDenom(t *testing.T) {
	receiverAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	senderAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	const denom = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	require.NoError(t, sdk.ValidateDenom(denom)) // an IBC denom is a valid bank denom
	require.Equal(t, 68, len(denom))             // "ibc/" + 64 hex

	key := types.GetStreamKey(receiverAddr, senderAddr, denom)
	expectedLen := len(types.StreamKeyPrefix) +
		len(address.MustLengthPrefix(receiverAddr)) +
		len(address.MustLengthPrefix(senderAddr)) +
		1 + len(denom) // 1-byte length prefix + raw denom
	require.Len(t, key, expectedLen)

	r, s, d := types.AddressesFromStreamKey(key)
	require.Equal(t, receiverAddr, r)
	require.Equal(t, senderAddr, s)
	require.Equal(t, denom, d)

	// The sender secondary index round-trips the same denom too (strip its
	// 0x12 | len(sender) | sender prefix before parsing the remainder).
	idxKey := types.GetStreamBySenderKey(senderAddr, receiverAddr, denom)
	rcv, dn := types.ReceiverAndDenomFromSenderIndexRemainder(idxKey[len(types.GetStreamsBySenderPrefixKey(senderAddr)):])
	require.Equal(t, receiverAddr, rcv)
	require.Equal(t, denom, dn)
}
