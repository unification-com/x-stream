package simulation

import (
	"encoding/json"
	"fmt"
	"math/rand"

	mathmod "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/unification-com/x-stream/x/stream/types"
)

// Simulation parameter constants
const (
	ValidatorFee = "validator_fee"
)

// GenValidatorFee randomized ValidatorFee — 0% to 10% inclusive, matching the
// hard cap enforced by types.MaxValidatorFee (post-audit).
func GenValidatorFee(r *rand.Rand) mathmod.LegacyDec {
	return mathmod.LegacyNewDecWithPrec(int64(r.Intn(11)), 2)
}

// RandomizedGenState generates a random GenesisState for the stream module.
func RandomizedGenState(simState *module.SimulationState) {
	var streams []types.StreamExport

	var validatorFee mathmod.LegacyDec
	simState.AppParams.GetOrGenerate(
		ValidatorFee, &validatorFee, simState.Rand,
		func(r *rand.Rand) { validatorFee = GenValidatorFee(r) },
	)

	params := types.NewParams(validatorFee)

	streamGenState := types.NewGenesisState(streams, params)
	bz, err := json.MarshalIndent(&streamGenState, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated stream parameters:\n%s\n", bz)

	simState.GenState[types.ModuleName] = bz
}
