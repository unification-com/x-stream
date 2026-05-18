package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/kv"
)

const (
	// ModuleName defines the module name
	ModuleName = "stream"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_stream"
)

// MaxStreamsPerSender caps the number of concurrent open streams a single
// sender address can hold. Defends against state-bloat / storage DoS where
// an attacker creates many small streams to grow the KV store. 1000 is
// generous for ordinary use (a typical sender will have a handful at most)
// while still bounding the worst case at O(N_addresses × 1000) entries.
const MaxStreamsPerSender = 1000

var (
	// ParamsKey is the prefix for the params store
	ParamsKey = []byte{0x01}

	// StreamKeyPrefix prefix for the primary Stream store. Full key shape is
	//   0x11 | len(receiver) | receiver | len(sender) | sender | len(denom) | denom
	StreamKeyPrefix = []byte{0x11}

	// StreamBySenderKeyPrefix is a secondary index keyed by sender, so that
	// AllStreamsForSender doesn't have to scan the entire stream store.
	// Full key shape is
	//   0x12 | len(sender) | sender | len(receiver) | receiver | len(denom) | denom
	// The value is empty — the index is purely a presence-marker. The full
	// Stream record lives under StreamKeyPrefix.
	StreamBySenderKeyPrefix = []byte{0x12}
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// GetStreamKey creates the key for the (receiver, sender, denom) triple
// 0x11 | len(receiver) | receiver | len(sender) | sender | len(denom) | denom
func GetStreamKey(receiverAddr sdk.AccAddress, senderAddr sdk.AccAddress, denom string) []byte {
	return append(GetStreamsByPairKey(receiverAddr, senderAddr), lengthPrefixedDenom(denom)...)
}

// GetStreamsByReceiverKey is the prefix for "all streams to receiver"
// 0x11 | len(receiver) | receiver
func GetStreamsByReceiverKey(receiverAddr sdk.AccAddress) []byte {
	return append(StreamKeyPrefix, address.MustLengthPrefix(receiverAddr)...)
}

// GetStreamsByPairKey is the prefix for "all streams between this (receiver, sender) pair, one entry per denom"
// 0x11 | len(receiver) | receiver | len(sender) | sender
func GetStreamsByPairKey(receiverAddr sdk.AccAddress, senderAddr sdk.AccAddress) []byte {
	return append(GetStreamsByReceiverKey(receiverAddr), address.MustLengthPrefix(senderAddr)...)
}

// AddressesFromStreamKey returns (receiver, sender, denom) from a full stream store key.
func AddressesFromStreamKey(key []byte) (sdk.AccAddress, sdk.AccAddress, string) {
	// key is of format:
	// 0x11<receiverAddrLen (1 Byte)><receiverAddr><senderAddrLen (1 Byte)><senderAddr><denomLen (1 Byte)><denom>

	receiverAddrLen, receiverAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 1, 1) // ignore key[0] since it is a prefix key
	receiverAddr, receiverAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, receiverAddrLenEndIndex+1, int(receiverAddrLen[0]))

	senderAddrLen, senderAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, receiverAddrEndIndex+1, 1)
	senderAddr, senderAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, senderAddrLenEndIndex+1, int(senderAddrLen[0]))

	denomLen, denomLenEndIndex := sdk.ParseLengthPrefixedBytes(key, senderAddrEndIndex+1, 1)
	denomBytes, denomEndIndex := sdk.ParseLengthPrefixedBytes(key, denomLenEndIndex+1, int(denomLen[0]))

	kv.AssertKeyAtLeastLength(key, denomEndIndex+1)
	return receiverAddr, senderAddr, string(denomBytes)
}

// FirstAddressFromStreamStoreKey parses the first address only
func FirstAddressFromStreamStoreKey(key []byte) sdk.AccAddress {
	addrLen := key[0]
	return sdk.AccAddress(key[1 : 1+addrLen])
}

// lengthPrefixedDenom returns a single-byte length prefix followed by the denom bytes.
// The denom length is bounded by sdk.Coin validation (≤128 chars) so a uint8 prefix is sufficient.
func lengthPrefixedDenom(denom string) []byte {
	b := []byte(denom)
	if len(b) > 255 {
		panic("denom length must be <= 255")
	}
	return append([]byte{byte(len(b))}, b...)
}

// --- Sender secondary index ---

// GetStreamBySenderKey constructs the full secondary-index key for a single
// stream identified by (sender, receiver, denom). The key alone is the marker;
// the value stored against it is empty.
func GetStreamBySenderKey(senderAddr sdk.AccAddress, receiverAddr sdk.AccAddress, denom string) []byte {
	out := append([]byte{}, StreamBySenderKeyPrefix...)
	out = append(out, address.MustLengthPrefix(senderAddr)...)
	out = append(out, address.MustLengthPrefix(receiverAddr)...)
	out = append(out, lengthPrefixedDenom(denom)...)
	return out
}

// GetStreamsBySenderPrefixKey is the prefix that selects every secondary-index
// entry for a given sender.
//
//	0x12 | len(sender) | sender
func GetStreamsBySenderPrefixKey(senderAddr sdk.AccAddress) []byte {
	return append(StreamBySenderKeyPrefix, address.MustLengthPrefix(senderAddr)...)
}

// ReceiverAndDenomFromSenderIndexRemainder parses (receiver, denom) out of a
// secondary-index key that has had its 0x12 | len(sender) | sender prefix
// stripped — i.e. just len(receiver) | receiver | len(denom) | denom.
//
// Returns nil/"" on malformed input rather than panicking, so iterators that
// encounter unexpected bytes can skip the entry instead of halting.
func ReceiverAndDenomFromSenderIndexRemainder(key []byte) (sdk.AccAddress, string) {
	if len(key) == 0 {
		return nil, ""
	}
	recvLen := int(key[0])
	if len(key) < 1+recvLen+1 {
		return nil, ""
	}
	receiver := sdk.AccAddress(key[1 : 1+recvLen])
	denomLen := int(key[1+recvLen])
	if len(key) < 1+recvLen+1+denomLen {
		return receiver, ""
	}
	denom := string(key[1+recvLen+1 : 1+recvLen+1+denomLen])
	return receiver, denom
}
