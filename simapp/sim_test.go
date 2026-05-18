//go:build sims

package simapp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/simsx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

func init() {
	simcli.GetSimulatorFlags()
}

func setupStateFactory(app *SimApp) simsx.SimStateFactory {
	return simsx.SimStateFactory{
		Codec:         app.AppCodec(),
		AppStateFn:    simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
		BlockedAddr:   BlockedAddresses(),
		AccountSource: app.AccountKeeper,
		BalanceSource: app.BankKeeper,
	}
}

// isEmptyValidatorSetErr lets us skip-not-fail when the simulator ends with zero
// validators.
func isEmptyValidatorSetErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "validator set is empty after InitGenesis")
}

func assertEqualStores(t testing.TB, a, b *SimApp, ctxA, ctxB sdk.Context) {
	t.Helper()

	skipPrefixes := map[string][][]byte{
		stakingtypes.StoreKey: {
			stakingtypes.UnbondingQueueKey,
			stakingtypes.RedelegationQueueKey,
			stakingtypes.ValidatorQueueKey,
			stakingtypes.HistoricalInfoKey,
			stakingtypes.UnbondingIDKey,
			stakingtypes.UnbondingIndexKey,
			stakingtypes.UnbondingTypeKey,
			stakingtypes.ValidatorUpdatesKey,
		},
		authzkeeper.StoreKey:   {authzkeeper.GrantQueuePrefix},
		feegrant.StoreKey:      {feegrant.FeeAllowanceQueueKeyPrefix.Bytes()},
		slashingtypes.StoreKey: {slashingtypes.ValidatorMissedBlockBitmapKeyPrefix},
		// upgrade module persists operational history that isn't round-tripped
		// through genesis. Empty prefix skips every key in the store.
		upgradetypes.StoreKey: {{}},
	}

	storeKeys := a.GetStoreKeys()
	require.NotEmpty(t, storeKeys)

	for _, keyA := range storeKeys {
		if _, ok := keyA.(*storetypes.KVStoreKey); !ok {
			continue
		}
		name := keyA.Name()
		keyB := b.GetKey(name)

		storeA := ctxA.KVStore(keyA)
		storeB := ctxB.KVStore(keyB)

		failedA, failedB := simtestutil.DiffKVStores(storeA, storeB, skipPrefixes[name])
		require.Equal(t, len(failedA), len(failedB), "unequal sets of key-values to compare %s", name)
		fmt.Printf("compared %d different key/value pairs between %s and %s\n", len(failedA), keyA, keyB)
		require.Equal(t, 0, len(failedA),
			simtestutil.GetSimulationLog(name, a.SimulationManager().StoreDecoders, failedA, failedB))
	}
}

// newImportApp builds a fresh *SimApp for the import side of an import/export test.
func newImportApp(t testing.TB, chainID string) *SimApp {
	t.Helper()
	workDir := t.TempDir()
	appOpts := make(simtestutil.AppOptionsMap)
	appOpts[flags.FlagHome] = workDir
	return NewSimApp(log.NewNopLogger(), dbm.NewMemDB(), true, appOpts, baseapp.SetChainID(chainID))
}

// TestFullAppSimulation runs the simulator end-to-end against a single seed.
func TestFullAppSimulation(t *testing.T) {
	cfg := simcli.NewConfigFromFlags()
	cfg.ChainID = simsx.SimAppChainID
	simsx.RunWithSeed(t, cfg, NewSimApp, setupStateFactory, cfg.Seed, nil)
}

// TestAppImportExport runs a simulation, exports genesis, re-imports into a
// fresh app, and asserts every KV store is byte-identical (modulo intentionally
// non-deterministic queue/bitmap prefixes).
func TestAppImportExport(t *testing.T) {
	importExport := func(tb testing.TB, simApp simsx.TestInstance[*SimApp], _ []simtypes.Account) {
		app := simApp.App
		exported, err := app.ExportAppStateAndValidators(false, []string{}, []string{})
		require.NoError(tb, err)

		newApp := newImportApp(tb, simApp.Cfg.ChainID)
		defer func() { _ = newApp.Close() }()

		var genesisState GenesisState
		require.NoError(tb, json.Unmarshal(exported.AppState, &genesisState))

		ctxA := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
		ctxB := newApp.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

		_, err = newApp.ModuleManager.InitGenesis(ctxB, app.AppCodec(), genesisState)
		if isEmptyValidatorSetErr(err) {
			tb.Skip("validator set is empty after InitGenesis, skipping import/export comparison")
			return
		}
		require.NoError(tb, err)
		require.NoError(tb, newApp.StoreConsensusParams(ctxB, exported.ConsensusParams))

		assertEqualStores(tb, app, newApp, ctxA, ctxB)
	}

	cfg := simcli.NewConfigFromFlags()
	cfg.ChainID = simsx.SimAppChainID
	simsx.RunWithSeed(t, cfg, NewSimApp, setupStateFactory, cfg.Seed, nil, importExport)
}

// TestAppSimulationAfterImport runs a simulation, exports the resulting genesis,
// reboots a fresh app from that export, and runs a second simulation on top.
func TestAppSimulationAfterImport(t *testing.T) {
	cfg := simcli.NewConfigFromFlags()
	cfg.ChainID = simsx.SimAppChainID

	afterImport := func(tb testing.TB, simApp simsx.TestInstance[*SimApp], accs []simtypes.Account) {
		app := simApp.App
		exported, err := app.ExportAppStateAndValidators(true, []string{}, []string{})
		require.NoError(tb, err)

		newApp := newImportApp(tb, simApp.Cfg.ChainID)
		defer func() { _ = newApp.Close() }()

		_, err = newApp.InitChain(&abci.RequestInitChain{
			AppStateBytes: exported.AppState,
			ChainId:       simApp.Cfg.ChainID,
		})
		if isEmptyValidatorSetErr(err) {
			tb.Skip("validator set is empty after InitChain, skipping post-import simulation")
			return
		}
		require.NoError(tb, err)
	}

	simsx.RunWithSeed(t, cfg, NewSimApp, setupStateFactory, cfg.Seed, nil, afterImport)
}

// TestAppStateDeterminism runs the same seed multiple times and asserts the
// final app hash is identical across runs.
func TestAppStateDeterminism(t *testing.T) {
	cfg := simcli.NewConfigFromFlags()
	cfg.InitialBlockHeight = 1
	cfg.ExportParamsPath = ""
	cfg.OnOperation = false
	cfg.ChainID = simsx.SimAppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 3
	if cfg.Seed != simcli.DefaultSeedValue {
		numSeeds = 1
	}

	for i := 0; i < numSeeds; i++ {
		var seed int64
		if cfg.Seed != simcli.DefaultSeedValue {
			seed = cfg.Seed
		} else {
			seed = rand.Int63()
		}
		t.Logf("seed %d/%d: %d", i+1, numSeeds, seed)

		hashes := make([][]byte, numTimesToRunPerSeed)
		for j := 0; j < numTimesToRunPerSeed; j++ {
			j := j
			captureHash := func(_ testing.TB, app simsx.TestInstance[*SimApp], _ []simtypes.Account) {
				hashes[j] = app.App.LastCommitID().Hash
			}
			simsx.RunWithSeed(t, cfg, NewSimApp, setupStateFactory, seed, nil, captureHash)
			if j > 0 {
				require.Equal(t, hashes[0], hashes[j],
					"non-determinism on seed %d: run %d/%d differs from run 1", seed, j+1, numTimesToRunPerSeed)
			}
		}
	}
}
