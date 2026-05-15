package keeper_test

import (
	"fmt"
	"testing"

	mathmod "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"

	"github.com/unification-com/x-stream/simapp"
	"github.com/unification-com/x-stream/x/stream/keeper"
	"github.com/unification-com/x-stream/x/stream/types"
)

type KeeperTestSuite struct {
	suite.Suite

	app         *simapp.SimApp
	ctx         sdk.Context
	queryClient types.QueryClient
	addrs       []sdk.AccAddress
	msgServer   types.MsgServer
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	app := simapp.Setup(s.T(), false)
	ctx := app.BaseApp.NewContext(false)

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, app.StreamKeeper)
	queryClient := types.NewQueryClient(queryHelper)

	s.app = app
	s.ctx = ctx
	s.queryClient = queryClient
	s.addrs = simapp.AddTestAddrsIncremental(app, ctx, 100, mathmod.NewInt(1000000000000000000))
	s.msgServer = keeper.NewMsgServerImpl(s.app.StreamKeeper)
}

func (s *KeeperTestSuite) TestGetAuthority() {
	// Authority is the gov module account address; deterministic from the bech32 prefix
	// configured at the application level (SDK simapp uses the default "cosmos" prefix).
	authority := s.app.StreamKeeper.GetAuthority()
	expected := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	s.Equal(expected, authority)
}

// TestGetStream_CorruptEntryHandledGracefully writes garbage bytes under a
// well-formed stream key and verifies that GetStream returns (zero, false)
// instead of panicking — the chain must keep running even if a single store
// entry becomes unparseable.
func (s *KeeperTestSuite) TestGetStream_CorruptEntryHandledGracefully() {
	receiver := s.addrs[0]
	sender := s.addrs[1]
	denom := sdk.DefaultBondDenom

	// Write malformed bytes at a real stream key
	key := types.GetStreamKey(receiver, sender, denom)
	store := s.ctx.KVStore(s.app.StreamKeeper.GetStoreKey())
	store.Set(key, []byte{0xff, 0xff, 0xff}) // not a valid Stream proto

	_, ok := s.app.StreamKeeper.GetStream(s.ctx, receiver, sender, denom)
	s.Require().False(ok, "corrupt entry must surface as 'not found' rather than panic")
}

// TestPerSenderStreamCap verifies that MsgCreateStream is rejected once the
// sender already has MaxStreamsPerSender streams open. We populate the
// secondary index directly with stub entries to avoid creating 1000 real
// streams (which would balloon test runtime).
func (s *KeeperTestSuite) TestPerSenderStreamCap() {
	sender := s.addrs[20]
	receiver := s.addrs[21]
	store := s.ctx.KVStore(s.app.StreamKeeper.GetStoreKey())

	// Pre-populate the sender index with MaxStreamsPerSender stub entries
	// using fake denom strings to make them unique under the index key.
	for i := 0; i < types.MaxStreamsPerSender; i++ {
		denom := fmt.Sprintf("stub%04d", i)
		store.Set(types.GetStreamBySenderKey(sender, receiver, denom), []byte{})
	}
	s.Require().Equal(types.MaxStreamsPerSender, s.app.StreamKeeper.CountStreamsForSender(s.ctx, sender))

	// Attempt to create a new stream — should be rejected
	msg := &types.MsgCreateStream{
		Sender:   sender.String(),
		Receiver: receiver.String(),
		Deposit:  sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000_000),
		FlowRate: 1,
	}
	_, err := s.msgServer.CreateStream(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "already has")
	s.Require().ErrorContains(err, "max 1000")

	// A different sender is unaffected
	otherSender := s.addrs[22]
	otherMsg := &types.MsgCreateStream{
		Sender:   otherSender.String(),
		Receiver: receiver.String(),
		Deposit:  sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000_000),
		FlowRate: 1,
	}
	_, err = s.msgServer.CreateStream(s.ctx, otherMsg)
	s.Require().NoError(err)
}

// TestSenderIndexLifecycle covers the secondary-index invariants:
//   - SetStream writes both primary and index entries
//   - DeleteStream removes both
//   - CountStreamsForSender reports the right number across multiple denoms
//   - The index distinguishes streams by denom (multi-denom support)
func (s *KeeperTestSuite) TestSenderIndexLifecycle() {
	receiver := s.addrs[10]
	sender := s.addrs[11]
	stream := types.Stream{
		Deposit:  sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000_000),
		FlowRate: 1,
	}

	// Initially zero
	s.Require().Equal(0, s.app.StreamKeeper.CountStreamsForSender(s.ctx, sender))

	// First stream
	s.Require().NoError(s.app.StreamKeeper.SetStream(s.ctx, receiver, sender, sdk.DefaultBondDenom, stream))
	s.Require().Equal(1, s.app.StreamKeeper.CountStreamsForSender(s.ctx, sender))

	// Index entry is present
	store := s.ctx.KVStore(s.app.StreamKeeper.GetStoreKey())
	s.Require().NotNil(store.Get(types.GetStreamBySenderKey(sender, receiver, sdk.DefaultBondDenom)))

	// Second stream under different denom — counts separately
	streamB := types.Stream{Deposit: sdk.NewInt64Coin("denomb", 1_000_000), FlowRate: 1}
	s.Require().NoError(s.app.StreamKeeper.SetStream(s.ctx, receiver, sender, "denomb", streamB))
	s.Require().Equal(2, s.app.StreamKeeper.CountStreamsForSender(s.ctx, sender))

	// Delete first stream — index entry goes too, count drops
	s.app.StreamKeeper.DeleteStream(s.ctx, receiver, sender, sdk.DefaultBondDenom)
	s.Require().Equal(1, s.app.StreamKeeper.CountStreamsForSender(s.ctx, sender))
	s.Require().Nil(store.Get(types.GetStreamBySenderKey(sender, receiver, sdk.DefaultBondDenom)))
	// Other denom's index entry remains
	s.Require().NotNil(store.Get(types.GetStreamBySenderKey(sender, receiver, "denomb")))

	// Other senders' counts unaffected
	otherSender := s.addrs[12]
	s.Require().Equal(0, s.app.StreamKeeper.CountStreamsForSender(s.ctx, otherSender))
}
