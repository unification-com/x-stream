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
