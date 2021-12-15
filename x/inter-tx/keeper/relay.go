package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	icatypes "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// TrySend builds a banktypes.NewMsgSend and uses the ibc-account module keeper to send the message to another chain
func (keeper Keeper) TrySend(
	ctx sdk.Context,
	owner sdk.AccAddress,
	msg sdk.Msg,
	connectionId string,
	counterpartyConnectionId string,
) error {
	portId, err := icatypes.GeneratePortID(owner.String(), connectionId, counterpartyConnectionId)
	if err != nil {
		return err
	}

	// var fromAddr string
	// fromAddr, errBool := keeper.icaControllerKeeper.GetInterchainAccountAddress(ctx, portId)
	// if errBool {
	// 	// TODO: replace with real error
	// 	return errors.New("No Intechain Account Registered for this Owner on this connection")
	// }

	chanId, found := keeper.icaControllerKeeper.GetActiveChannelID(ctx, portId)
	if !found {
		return sdkerrors.Wrapf(icatypes.ErrActiveChannelNotFound, "failed to retrieve active channel for port %s", portId)
	}

	chanCap, found := keeper.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(portId, chanId))
	if !found {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	// msg := &banktypes.MsgSend{FromAddress: fromAddr, ToAddress: toAddr, Amount: amt}
	data, err := icatypes.SerializeCosmosTx(keeper.cdc, []sdk.Msg{msg})
	if err != nil {
		return err
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	_, err = keeper.icaControllerKeeper.TrySendTx(ctx, chanCap, portId, packetData)
	return err
}
