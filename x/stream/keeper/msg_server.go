package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	mathmod "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/unification-com/x-stream/x/stream/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateStream creates a new stream
func (k msgServer) CreateStream(goCtx context.Context, msg *types.MsgCreateStream) (*types.MsgCreateStreamResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, accErr := sdk.AccAddressFromBech32(msg.Sender)
	if accErr != nil {
		return nil, accErr
	}
	receiverAddr, accErr := sdk.AccAddressFromBech32(msg.Receiver)
	if accErr != nil {
		return nil, accErr
	}

	if k.bankKeeper.BlockedAddr(receiverAddr) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.Receiver)
	}

	if msg.Sender == msg.Receiver {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "sender and receiver cannot be same address")
	}

	if msg.Deposit.IsNil() || msg.Deposit.IsNegative() || msg.Deposit.IsZero() {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "deposit must be > zero")
	}

	if k.IsStream(ctx, receiverAddr, senderAddr, msg.Deposit.Denom) {
		return nil, errorsmod.Wrap(types.ErrStreamExists, "use update stream msg to modify existing stream")
	}

	// Per-sender stream cap: deter state-bloat. Enforced here rather than in
	// ValidateBasic because it needs stream-count state.
	if count := k.CountStreamsForSender(ctx, senderAddr); count >= types.MaxStreamsPerSender {
		return nil, errorsmod.Wrapf(types.ErrInvalidData,
			"sender %s already has %d open streams (max %d)",
			senderAddr.String(), count, types.MaxStreamsPerSender)
	}

	if msg.FlowRate <= 0 {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "flow rate must be > zero")
	}

	duration := types.CalculateDuration(msg.Deposit, msg.FlowRate)

	if duration < 60 {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "calculated duration too short. Must be > 1 minute")
	}

	// Cap stream duration to prevent time.Duration overflow in the keeper
	// math (nowTime.Add(time.Second * time.Duration(N)) wraps once N
	// nanoseconds exceeds ~292 years).
	if duration > types.MaxStreamDurationSeconds {
		return nil, errorsmod.Wrapf(types.ErrInvalidData,
			"calculated duration %d exceeds max %d seconds (10 years)", duration, types.MaxStreamDurationSeconds)
	}

	// create the "empty" stream
	_, err := k.CreateNewStream(ctx, receiverAddr, senderAddr, msg.Deposit, msg.FlowRate)

	if err != nil {
		return nil, err
	}

	// add the deposit
	_, err = k.AddDeposit(ctx, receiverAddr, senderAddr, msg.Deposit)
	if err != nil {
		return nil, err
	}

	defer telemetry.IncrCounter(1, types.ModuleName, types.EventTypeCreateStreamAction)

	return &types.MsgCreateStreamResponse{
		Receiver: msg.Receiver,
		Sender:   msg.Sender,
		Deposit:  msg.Deposit,
		FlowRate: msg.FlowRate,
	}, nil

}

// ClaimStream claims from a stream using sender, receiver and denom as inputs
func (k msgServer) ClaimStream(goCtx context.Context, msg *types.MsgClaimStream) (*types.MsgClaimStreamResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, accErr := sdk.AccAddressFromBech32(msg.Sender)
	if accErr != nil {
		return nil, accErr
	}
	receiverAddr, accErr := sdk.AccAddressFromBech32(msg.Receiver)
	if accErr != nil {
		return nil, accErr
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidData, err.Error())
	}

	ok := k.IsStream(ctx, receiverAddr, senderAddr, msg.Denom)

	if !ok {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "stream not found")
	}

	finalClaimCoin, valFeeCoin, totalClaimValue, remainingDeposit, err := k.ClaimFromStream(ctx, receiverAddr, senderAddr, msg.Denom)

	if err != nil {
		return nil, err
	}

	return &types.MsgClaimStreamResponse{
		TotalClaimed:     totalClaimValue,
		StreamPayment:    finalClaimCoin,
		ValidatorFee:     valFeeCoin,
		RemainingDeposit: remainingDeposit,
	}, nil
}

// TopUpDeposit adds more deposit to a stream. The denom is carried inside msg.Deposit;
// if no stream exists for (sender, receiver, msg.Deposit.Denom), the call errors with
// "stream not found" rather than the old "denom mismatch" — under multi-denom, an absent
// stream for a denom is genuinely not-found rather than a mismatched-denom condition.
func (k msgServer) TopUpDeposit(goCtx context.Context, msg *types.MsgTopUpDeposit) (*types.MsgTopUpDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, accErr := sdk.AccAddressFromBech32(msg.Sender)
	if accErr != nil {
		return nil, accErr
	}

	receiverAddr, accErr := sdk.AccAddressFromBech32(msg.Receiver)
	if accErr != nil {
		return nil, accErr
	}

	if msg.Deposit.IsNil() || msg.Deposit.IsNegative() || msg.Deposit.IsZero() {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "deposit must be > zero")
	}

	stream, ok := k.GetStream(ctx, receiverAddr, senderAddr, msg.Deposit.Denom)
	if !ok {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "stream not found")
	}

	// Cap the prospective total duration. The keeper's AddDeposit adds the
	// top-up's duration extension to the existing DepositZeroTime; compute
	// the same extension here and reject if the total would breach the cap
	// before any state mutates.
	extensionSecs := types.CalculateDuration(msg.Deposit, stream.FlowRate)
	nowTime := ctx.BlockTime()
	var existingRemaining int64
	if stream.DepositZeroTime.After(nowTime) {
		existingRemaining = int64(stream.DepositZeroTime.Sub(nowTime).Seconds())
	}
	if existingRemaining+extensionSecs > types.MaxStreamDurationSeconds {
		return nil, errorsmod.Wrapf(types.ErrInvalidData,
			"top-up would extend stream duration beyond max %d seconds (10 years); current remaining=%d, extension=%d",
			types.MaxStreamDurationSeconds, existingRemaining, extensionSecs)
	}

	// Add the requested deposit
	_, err := k.AddDeposit(ctx, receiverAddr, senderAddr, msg.Deposit)

	if err != nil {
		return nil, err
	}

	// get updated stream data
	stream, _ = k.GetStream(ctx, receiverAddr, senderAddr, msg.Deposit.Denom)

	return &types.MsgTopUpDepositResponse{
		DepositAmount:   msg.Deposit,
		CurrentDeposit:  stream.Deposit,
		DepositZeroTime: stream.DepositZeroTime,
	}, nil

}

// UpdateFlowRate updates the flow rate on a stream identified by (sender, receiver, denom)
func (k msgServer) UpdateFlowRate(goCtx context.Context, msg *types.MsgUpdateFlowRate) (*types.MsgUpdateFlowRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, accErr := sdk.AccAddressFromBech32(msg.Sender)
	if accErr != nil {
		return nil, accErr
	}

	receiverAddr, accErr := sdk.AccAddressFromBech32(msg.Receiver)
	if accErr != nil {
		return nil, accErr
	}

	if msg.FlowRate <= 0 {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "flow rate must be > zero")
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidData, err.Error())
	}

	stream, ok := k.GetStream(ctx, receiverAddr, senderAddr, msg.Denom)
	if !ok {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "stream not found")
	}

	// Cap the prospective new duration. SetNewFlowRate effectively resets
	// the stream's deposit-zero time to (now + remaining_deposit/new_rate);
	// reject if that exceeds the cap.
	if stream.Deposit.Amount.GT(mathmod.NewIntFromUint64(0)) {
		newDuration := types.CalculateDuration(stream.Deposit, msg.FlowRate)
		if newDuration > types.MaxStreamDurationSeconds {
			return nil, errorsmod.Wrapf(types.ErrInvalidData,
				"new flow rate would extend duration beyond max %d seconds (10 years); calculated=%d",
				types.MaxStreamDurationSeconds, newDuration)
		}
	}

	// update the flow rate
	err := k.SetNewFlowRate(ctx, receiverAddr, senderAddr, msg.Denom, msg.FlowRate)

	if err != nil {
		return nil, err
	}

	return &types.MsgUpdateFlowRateResponse{
		FlowRate: msg.FlowRate,
	}, nil
}

func (k msgServer) CancelStream(goCtx context.Context, msg *types.MsgCancelStream) (*types.MsgCancelStreamResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, accErr := sdk.AccAddressFromBech32(msg.Sender)
	if accErr != nil {
		return nil, accErr
	}

	receiverAddr, accErr := sdk.AccAddressFromBech32(msg.Receiver)
	if accErr != nil {
		return nil, accErr
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidData, err.Error())
	}

	stream, ok := k.GetStream(ctx, receiverAddr, senderAddr, msg.Denom)

	if !ok {
		return nil, errorsmod.Wrap(types.ErrInvalidData, "stream not found")
	}

	if !stream.Cancellable {
		return nil, errorsmod.Wrap(types.ErrStreamNotCancellable, "cannot be cancelled")
	}

	// cancel stream
	err := k.CancelStreamBySenderReceiver(ctx, receiverAddr, senderAddr, msg.Denom)

	if err != nil {
		return nil, err
	}

	return &types.MsgCancelStreamResponse{}, nil
}

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
