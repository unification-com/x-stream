package simapp

import (
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

// createRandomAccounts generates accNum random addresses from fresh ed25519 keys.
func createRandomAccounts(accNum int) []sdk.AccAddress {
	addrs := make([]sdk.AccAddress, accNum)
	for i := 0; i < accNum; i++ {
		privK := mock.NewPV()
		pubK := privK.PrivKey.PubKey()
		addrs[i] = sdk.AccAddress(pubK.Address())
	}
	return addrs
}

// SimTestDefaultStreamValFee is the default validator fee used in stream
// keeper tests. Matches the upstream module's DefaultValidatorFee of 1%.
var SimTestDefaultStreamValFee = sdkmath.LegacyNewDecWithPrec(1, 2)

// AddTestAddrsWithExtraNonBondCoin constructs and returns accNum accounts,
// each funded with accAmt of the bond denomination and an additional extraCoin.
// Used by stream keeper tests that exercise non-bond-denom streams.
func AddTestAddrsWithExtraNonBondCoin(app *SimApp, ctx sdk.Context, accNum int, accAmt sdkmath.Int, extraCoin sdk.Coin) []sdk.AccAddress {
	testAddrs := createRandomAccounts(accNum)
	bondDenom, err := app.StakingKeeper.BondDenom(ctx)
	if err != nil {
		panic(err)
	}

	initCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, accAmt), extraCoin)

	for _, addr := range testAddrs {
		if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, initCoins); err != nil {
			panic(err)
		}
		if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, initCoins); err != nil {
			panic(err)
		}
	}

	return testAddrs
}
