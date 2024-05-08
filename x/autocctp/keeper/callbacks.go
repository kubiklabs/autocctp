package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func (k Keeper) IBCSendPacketCallback(
	cachedCtx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	packetData []byte,
	contractAddress,
	packetSenderAddress string,
) error {
	// no-op, since we are not interested in this callback
	return nil
}

func (k Keeper) IBCOnAcknowledgementPacketCallback(
	cachedCtx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	// no-op, since we are not interested in this callback
	return nil
}

func (k Keeper) IBCOnTimeoutPacketCallback(
	cachedCtx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	// no-op, since we are not interested in this callback
	return nil
}

func (k Keeper) IBCReceivePacketCallback(
	cachedCtx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
	contractAddress string,
) error {
	// sender validation makes no sense here, as the receiver is never the sender
	_, err := sdk.AccAddressFromBech32(contractAddress)
	if err != nil {
		return err
	}

	// only execute callback if x/autocctp is active
	// params := k.GetParams(cachedCtx)
	// TODO: add later when errors.go is imported
	// if !params.AutocctpActive {
	// 	return errorsmod.Wrapf(types.ErrAutoCctpInactive, "x/autocctp cctp routing is inactive")
	// }

	// cctp transaction message
	msg := &cctptypes.MsgDepositForBurn{
		From:              "transferMetadata.Receiver", // TODO: check if noble address
		Amount:            math.OneInt(),
		DestinationDomain: 1, // TODO: fix this hardcode, fetch the map of domains and validate the "memo" packet domain value
		MintRecipient:     []byte("autopilotMetadata.MintRecipient"),
		BurnToken:         "",
	}

	// TODO: uncomment later upon getting more details
	// if err := msg.ValidateBasic(); err != nil {
	// 	return err
	// }

	_, err = k.cctpKeeper.DepositForBurn(cachedCtx, msg)
	if err != nil {
		return errorsmod.Wrap(err, "on destination chain callback")
	}

	return nil
}
