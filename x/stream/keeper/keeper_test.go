package keeper_test

import (
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
