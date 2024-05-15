package autocctp

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	cctpkeeper "github.com/circlefin/noble-cctp/x/cctp/keeper"
	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ porttypes.IBCModule = &IBCMiddleware{}

// IBCMiddleware implements the tokenfactory keeper in order to check against blacklisted addresses.
type IBCMiddleware struct {
	app        porttypes.IBCModule
	cctpKeeper *cctpkeeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application.
func NewIBCMiddleware(app porttypes.IBCModule, k *cctpkeeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:        app,
		cctpKeeper: k,
	}
}

// OnChanOpenInit implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
}

// OnChanOpenTry implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (version string, err error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID, channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface.
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCModule interface.
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket intercepts the packet data and checks the sender and receiver address against
// the blacklisted addresses held in the tokenfactory keeper. If the address is found in the blacklist, an
// acknowledgment error is returned.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	var ackErr error
	if err := cctptypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ackErr = errors.Wrapf(sdkerrors.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
		return channeltypes.NewErrorAcknowledgement(ackErr)
	}

	// parse the ics-20 receiver address
	// _, addressBz, err := bech32.DecodeAndConvert(data.Receiver)
	// _, addressBz, err = bech32.DecodeAndConvert(data.Sender)
	// if err != nil {
	// 	return channeltypes.NewErrorAcknowledgement(err)
	// }

	// _, found := im.cctpKeeper.GetBlacklisted(ctx, addressBz)
	// if found {
	// 	ackErr = errors.Wrapf(sdkerrors.ErrUnauthorized, "receiver address is blacklisted")
	// 	return channeltypes.NewErrorAcknowledgement(ackErr)
	// }

	// parse the memo field in the ics-20 packet

	// get DestinationDomain, MintRecipient from memo field

	// Pass the new packet down the middleware stack first to complete the transfer
	ack := im.app.OnRecvPacket(ctx, packet, relayer)
	if !ack.Success() {
		return ack
	}

	// cctp transaction message
	msg := &cctptypes.MsgDepositForBurn{
		From:              "transferMetadata.Receiver", // TODO: check if noble address
		Amount:            math.OneInt(),
		DestinationDomain: 1, // TODO: fix this hardcode, fetch the map of domains and validate the "memo" packet domain value
		MintRecipient:     []byte("autopilotMetadata.MintRecipient"),
		BurnToken:         "",
	}

	// TODO: implement msg validatoion and uncomment
	// if err := msg.ValidateBasic(); err != nil {
	// 	return err
	// }

	msgServer := cctpkeeper.NewMsgServerImpl(im.cctpKeeper)
	_, err := msgServer.DepositForBurn(
		sdk.WrapSDKContext(ctx),
		msg,
	)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
		// TODO: implement error type and uncomment
		// return errors.Wrapf(err, "failed to cctp")
	}

	// _, addressBz, err := bech32.DecodeAndConvert(data.Receiver)
	// _, addressBz, err = bech32.DecodeAndConvert(data.Sender)
	// if err != nil {
	// 	return channeltypes.NewErrorAcknowledgement(err)
	// }

	// _, found := im.cctpKeeper.GetBlacklisted(ctx, addressBz)
	// if found {
	// 	ackErr = errors.Wrapf(sdkerrors.ErrUnauthorized, "receiver address is blacklisted")
	// 	return channeltypes.NewErrorAcknowledgement(ackErr)
	// }

	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCModule interface.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}
