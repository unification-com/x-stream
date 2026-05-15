package types_test

import (
	"testing"

	mathmod "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/unification-com/x-stream/x/stream/types"
)

func TestParamsValidate(t *testing.T) {
	// Default (1%) — accepted
	require.NoError(t, types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(1, 2)}.Validate())

	// Zero — accepted (no validator fee)
	require.NoError(t, types.Params{ValidatorFee: mathmod.LegacyZeroDec()}.Validate())

	// Negative — rejected
	err := types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(-1, 2)}.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator fee cannot be negative:")

	// Exactly at MaxValidatorFee (10%) — accepted (boundary case)
	require.NoError(t, types.Params{ValidatorFee: types.MaxValidatorFee}.Validate())

	// Just above MaxValidatorFee — rejected
	err = types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(11, 2)}.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator fee cannot exceed")

	// 100% — rejected
	err = types.Params{ValidatorFee: mathmod.LegacyOneDec()}.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator fee cannot exceed")

	// >100% — rejected
	err = types.Params{ValidatorFee: mathmod.LegacyNewDecWithPrec(101, 2)}.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator fee cannot exceed")

	// Nil — rejected
	err = types.Params{ValidatorFee: mathmod.LegacyDec{}}.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator fee cannot be nil")
}
