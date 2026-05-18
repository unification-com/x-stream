package keeper_test

import (
	mathmod "cosmossdk.io/math"

	"github.com/unification-com/x-stream/simapp"
	"github.com/unification-com/x-stream/x/stream/types"
)

func (s *KeeperTestSuite) TestParamsQuery() {
	defaultFee := simapp.SimTestDefaultStreamValFee
	// Use a fee within MaxValidatorFee (10%).
	newFee := mathmod.LegacyNewDecWithPrec(5, 2) // 5%

	req1 := &types.QueryParamsRequest{}
	expRes1 := &types.QueryParamsResponse{Params: types.DefaultParams()}

	res1, err1 := s.app.StreamKeeper.Params(s.ctx, req1)

	s.Require().NoError(err1)
	s.Require().Equal(expRes1, res1)

	req2 := &types.QueryParamsRequest{}
	expRes2 := &types.QueryParamsResponse{Params: types.Params{ValidatorFee: defaultFee}}

	res2, err2 := s.app.StreamKeeper.Params(s.ctx, req2)

	s.Require().NoError(err2)
	s.Require().Equal(expRes2, res2)

	err := s.app.StreamKeeper.SetParams(s.ctx, types.NewParams(newFee))
	s.Require().NoError(err)

	req3 := &types.QueryParamsRequest{}
	expRes3 := &types.QueryParamsResponse{Params: types.Params{ValidatorFee: newFee}}

	res3, err3 := s.app.StreamKeeper.Params(s.ctx, req3)

	s.Require().NoError(err3)
	s.Require().Equal(expRes3, res3)

	// above-cap fee rejected by SetParams
	aboveCap := mathmod.LegacyNewDecWithPrec(11, 2) // 11%
	err = s.app.StreamKeeper.SetParams(s.ctx, types.NewParams(aboveCap))
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "validator fee cannot exceed")
}
