package types

import (
	"time"

	mathmod "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MaxStreamDurationSeconds caps the lifetime of any individual stream at
// 10 years (315,360,000 s). Caps two distinct risks:
//
//   - time.Duration is int64 nanoseconds (max ~292 years). A naïve duration
//     near 9.2e9 seconds would overflow the time arithmetic in
//     keeper.AddDeposit / keeper.SetNewFlowRate, producing a malformed
//     DepositZeroTime.
//   - Indefinite-duration streams are an unusual user pattern; an explicit
//     bound forces senders to choose a reasonable horizon and re-topup if
//     they want to extend.
//
// Any MsgCreateStream / MsgTopUpDeposit / MsgUpdateFlowRate whose resulting
// duration exceeds this is rejected at msg-server time.
const MaxStreamDurationSeconds int64 = 10 * 365 * 24 * 60 * 60 // 10 years

func PeriodEnumFromString(period string) StreamPeriod {
	switch period {
	case "Second", "second", "sec":
		return StreamPeriodSecond
	case "Minute", "minute", "min":
		return StreamPeriodMinute
	case "Hour", "hour":
		return StreamPeriodHour
	case "Day", "day":
		return StreamPeriodDay
	case "Week", "week":
		return StreamPeriodWeek
	case "Month", "month", "mon":
		return StreamPeriodMonth
	case "Year", "year":
		return StreamPeriodYear
	}
	return StreamPeriodUnspecified
}

func CalculateFlowRateForCoin(coin sdk.Coin, period StreamPeriod, duration uint64) (uint64, mathmod.LegacyDec, int64) {
	var baseDuration uint64
	var totalDuration uint64

	switch period {
	case StreamPeriodUnspecified:
		baseDuration = 1
	case StreamPeriodSecond:
		baseDuration = 1
	case StreamPeriodMinute:
		baseDuration = 60
	case StreamPeriodHour:
		baseDuration = 3600
	case StreamPeriodDay:
		baseDuration = 86400
	case StreamPeriodWeek:
		baseDuration = 604800
	case StreamPeriodMonth:
		baseDuration = 2628000 // (365 / 12) * 24 * 60 * 60 = 30.416666667 * 24 * 60 * 60
	case StreamPeriodYear:
		baseDuration = 31536000
	default:
		// unrecognised period — fall back to seconds (1)
		baseDuration = 1
	}

	totalDuration = baseDuration * duration

	if coin.IsNil() || coin.IsNegative() || coin.IsZero() || totalDuration == 0 {
		return totalDuration, mathmod.LegacyNewDecWithPrec(0, 0), 0
	}

	// flow rate calculation from deposit and duration
	decCoin := sdk.NewDecCoinFromCoin(coin)
	decDuration := mathmod.LegacyNewDecFromInt(mathmod.NewIntFromUint64(totalDuration))

	flowRate := decCoin.Amount.QuoTruncateMut(decDuration)

	// note: decimal values are rounded down, e.g. 8.9 to just 8.
	return totalDuration, flowRate, flowRate.TruncateInt64()
}

func CalculateDuration(deposit sdk.Coin, flowRate int64) int64 {
	// no point if flowRate is <= 0
	if flowRate <= 0 {
		return 0
	}
	// no point if the deposit value is zero - e.g. if re-calculating from a new flow rate
	// of an existing stream
	if deposit.Amount.GT(mathmod.NewIntFromUint64(0)) {
		// calculate duration in seconds
		decFlowRate := mathmod.LegacyNewDecFromInt(mathmod.NewIntFromUint64(uint64(flowRate)))
		decDeposit := sdk.NewDecCoinFromCoin(deposit)
		decDuration := decDeposit.Amount.QuoTruncateMut(decFlowRate)
		// note: decimal values are rounded down, e.g. 2628008.9 to just 2628008.
		return decDuration.TruncateInt64()
	}

	return 0
}

func CalculateAmountToClaim(
	nowTime,
	depositZeroTime,
	lastOutflowTime time.Time,
	deposit sdk.Coin,
	flowRate int64,
) (sdk.Coin, sdk.Coin) {
	var amountToClaim sdk.Coin
	var remainingDepositValue sdk.Coin

	if nowTime.After(depositZeroTime) || nowTime.Equal(depositZeroTime) {
		// now > deposit_zero_time, use all remaining deposit
		amountToClaim = deposit
		remainingDepositValue = sdk.NewCoin(deposit.Denom, mathmod.NewInt(0))
	} else {
		// Compute claim amount in arbitrary-precision math.Int rather than
		// int64 — at extreme inputs (flow_rate × elapsed near int64 max)
		// the multiplication would otherwise wrap silently.
		timeSinceLast := nowTime.Sub(lastOutflowTime)
		secondsSinceLast := int64(timeSinceLast.Seconds())
		if secondsSinceLast < 0 {
			// clock skew or invalid lastOutflowTime — treat as no time elapsed
			secondsSinceLast = 0
		}
		// flowRate is validated > 0 by msg_server before this is reached, but
		// guard anyway to keep this pure-fn defensible.
		var flowInt mathmod.Int
		if flowRate < 0 {
			flowInt = mathmod.NewInt(0)
		} else {
			flowInt = mathmod.NewInt(flowRate)
		}
		numCoinsInt := mathmod.NewInt(secondsSinceLast).Mul(flowInt)
		amountToClaim = sdk.NewCoin(deposit.Denom, numCoinsInt)
		if deposit.Amount.GT(amountToClaim.Amount) {
			remainingDepositValue = deposit.Sub(amountToClaim)
		} else {
			// computed accrual exceeds deposit — cap at deposit
			amountToClaim = deposit
			remainingDepositValue = sdk.NewCoin(deposit.Denom, mathmod.NewInt(0))
		}
	}

	return amountToClaim, remainingDepositValue
}

func CalculateValidatorFee(valFee mathmod.LegacyDec, amountToClaim sdk.Coin) (sdk.Coin, sdk.Coin) {

	var valFeeCoin sdk.Coin
	var finalClaimCoin sdk.Coin

	if valFee.GT(mathmod.LegacyNewDecFromInt(mathmod.NewIntFromUint64(0))) {
		decCoin := sdk.NewDecCoinFromCoin(amountToClaim)
		valFeeAmount := decCoin.Amount.Mul(valFee).TruncateInt64()
		valFeeCoin = sdk.NewCoin(amountToClaim.Denom, mathmod.NewIntFromUint64(uint64(valFeeAmount)))
		finalClaimCoin = amountToClaim.Sub(valFeeCoin)
	} else {
		valFeeCoin = sdk.NewCoin(amountToClaim.Denom, mathmod.NewIntFromUint64(0))
		finalClaimCoin = amountToClaim
	}

	return finalClaimCoin, valFeeCoin
}
