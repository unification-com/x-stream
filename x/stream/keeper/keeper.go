package keeper

import (
	"fmt"

	"cosmossdk.io/log/v2"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/unification-com/x-stream/x/stream/types"
)

type (
	Keeper struct {
		cdc              codec.BinaryCodec
		storeKey         storetypes.StoreKey
		bankKeeper       types.BankKeeper
		accKeeper        types.AccountKeeper
		feeCollectorName string
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
	}
)

func NewKeeper(
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	accKeeper types.AccountKeeper,
	cdc codec.BinaryCodec,
	feeCollectorName string,
	authority string,
) Keeper {

	// ensure module account is set in SupplyKeeper
	if addr := accKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// Validate authority at construction so a misconfigured app fails at
	// boot rather than later when a MsgUpdateParams is first submitted.
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %q: %s", authority, err))
	}

	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		authority:        authority,
		bankKeeper:       bankKeeper,
		accKeeper:        accKeeper,
		feeCollectorName: feeCollectorName,
	}
}

// GetAuthority returns the x/stream module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) Cdc() codec.BinaryCodec {
	return k.cdc
}

// GetStoreKey returns the module's store key. Used by migration handlers.
func (k Keeper) GetStoreKey() storetypes.StoreKey {
	return k.storeKey
}

// GetStreamModuleAccount returns the stream ModuleAccount
func (k Keeper) GetStreamModuleAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.accKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// GetStreamModuleAccountBalances returns the stream ModuleAccount's balances from the bank keeper
func (k Keeper) GetStreamModuleAccountBalances(ctx sdk.Context) sdk.Coins {
	return k.bankKeeper.GetAllBalances(ctx, k.GetStreamModuleAccount(ctx).GetAddress())
}
