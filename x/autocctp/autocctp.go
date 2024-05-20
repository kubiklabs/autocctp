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

	"github.com/noble-assets/autocctp/v2/x/autocctp/types"
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
	return im.app.OnChanOpenTry(
		ctx,
		order,
		connectionHops,
		portID,
		channelID,
		chanCap,
		counterparty,
		counterpartyVersion,
	)
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
	var transferData transfertypes.FungibleTokenPacketData
	var ackErr error
	if err := cctptypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &transferData); err != nil {
		ackErr = errors.Wrapf(sdkerrors.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
		return channeltypes.NewErrorAcknowledgement(ackErr)
	}

	// _, addressBz, err := bech32.DecodeAndConvert(data.Receiver)
	var transferReceiver = transferData.Receiver
	// TODO: handle ok here or see a better way to get intgere ICS-20 amount, ref github
	var transferAmount, _ = math.NewIntFromString(transferData.Amount)

	// Pass the new packet down the middleware stack first to complete the transfer
	//  allowing everything to be executed before the CCTP message is sent.
	ack := im.app.OnRecvPacket(ctx, packet, relayer)
	if !ack.Success() {
		return ack
	}

	var memoData types.CctpMemo
	// Attempt to parse the memo field as either DepositForBurn or DepositForBurnWithCaller.
	if err := cctptypes.ModuleCdc.UnmarshalJSON([]byte(transferData.GetMemo()), &memoData); err == nil {
		depositForBurn := memoData.Circle.Cctp.GetDepositForBurn()
		depositForBurnWithCaller := memoData.Circle.Cctp.GetDepositForBurnWithCaller()
		if depositForBurn != nil {
			var memoDestinationDomain = depositForBurn.DestinationDomain
			var memoMintRecipient = depositForBurn.MintRecipient
			var memoAmount = depositForBurn.Amount

			// Check that the amount in the memo is <= the amount in the ICS20 transfer.
			if memoAmount.GT(transferAmount) {
				// TODO: return err ack here
			}

			// - Construct a CCTP transaction based on the memo field, and run ValidateBasic.
			cctpMsg := &cctptypes.MsgDepositForBurn{
				From:              transferReceiver,
				Amount:            memoAmount,
				DestinationDomain: memoDestinationDomain,
				MintRecipient:     []byte(memoMintRecipient),
				BurnToken:         "",
			}
			// if err := cctpMsg.ValidateBasic(); err != nil {
			// 	return err
			// }

			// - Execute that CCTP transaction and return the result.
			msgServer := cctpkeeper.NewMsgServerImpl(im.cctpKeeper)
			_, err := msgServer.DepositForBurn(ctx, cctpMsg)
			if err != nil {
				return channeltypes.NewErrorAcknowledgement(err)
				// TODO: implement error type and uncomment
				// return errors.Wrapf(err, "failed to cctp")
			}
		} else if depositForBurnWithCaller != nil {
			var memoDestinationDomain = depositForBurnWithCaller.DestinationDomain
			var memoMintRecipient = depositForBurnWithCaller.MintRecipient
			var memoAmount = depositForBurnWithCaller.Amount
			var memoDestinationCaller = depositForBurnWithCaller.DestinationCaller

			cctpMsg := &cctptypes.MsgDepositForBurnWithCaller{
				From:              transferReceiver,
				Amount:            memoAmount,
				DestinationDomain: memoDestinationDomain,
				MintRecipient:     []byte(memoMintRecipient),
				BurnToken:         "",
				DestinationCaller: []byte(memoDestinationCaller),
			}
			// if err := cctpMsg.ValidateBasic(); err != nil {
			// 	return err
			// }

			// - Execute that CCTP transaction and return the result.
			msgServer := cctpkeeper.NewMsgServerImpl(im.cctpKeeper)
			_, err := msgServer.DepositForBurnWithCaller(ctx, cctpMsg)
			if err != nil {
				return channeltypes.NewErrorAcknowledgement(err)
				// TODO: implement error type and uncomment
				// return errors.Wrapf(err, "failed to cctp")
			}
		} else {
			// Unable to parse, return the acknowledge received by the underlying middlwares.
			return ack
		}
	}
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
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}
