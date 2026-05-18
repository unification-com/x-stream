package keeper_test

import (
	mathmod "cosmossdk.io/math"

	"github.com/unification-com/x-stream/x/stream/types"
)

func (s *KeeperTestSuite) TestParams() {
	testCases := []struct {
		name      string
		input     types.Params
		expectErr bool
	}{
		{
			name: "set valid params (5% within cap)",
			input: types.Params{
				ValidatorFee: mathmod.LegacyNewDecWithPrec(5, 2),
			},
			expectErr: false,
		},
		{
			name: "set valid params exactly at cap (10%)",
			input: types.Params{
				ValidatorFee: types.MaxValidatorFee,
			},
			expectErr: false,
		},
		{
			name: "> MaxValidatorFee (11%) rejected",
			input: types.Params{
				ValidatorFee: mathmod.LegacyNewDecWithPrec(11, 2),
			},
			expectErr: true,
		},
		{
			name: "100% rejected",
			input: types.Params{
				ValidatorFee: mathmod.LegacyOneDec(),
			},
			expectErr: true,
		},
		{
			name: "set invalid params > 100%",
			input: types.Params{
				ValidatorFee: mathmod.LegacyNewDecWithPrec(101, 2),
			},
			expectErr: true,
		},
		{
			name: "set invalid params negative value",
			input: types.Params{
				ValidatorFee: mathmod.LegacyNewDecWithPrec(-1, 2),
			},
			expectErr: true,
		},
		{
			name: "set invalid params nil value",
			input: types.Params{
				ValidatorFee: mathmod.LegacyDec{},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			expected := s.app.StreamKeeper.GetParams(s.ctx)
			err := s.app.StreamKeeper.SetParams(s.ctx, tc.input)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				expected = tc.input
				s.Require().NoError(err)
			}

			p := s.app.StreamKeeper.GetParams(s.ctx)
			s.Require().Equal(expected, p)
		})
	}
}
