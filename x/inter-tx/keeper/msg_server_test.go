package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"

	"github.com/cosmos/interchain-accounts/x/inter-tx/keeper"
	"github.com/cosmos/interchain-accounts/x/inter-tx/types"
)

func (suite *KeeperTestSuite) TestRegisterInterchainAccount() {
	var (
		owner string
		path  *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"port is already bound",
			func() {
				suite.GetICAApp(suite.chainA).IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), TestPortID)
			},
			false,
		},
		{
			"fails to generate port-id",
			func() {
				owner = ""
			},
			false,
		},
		{
			"MsgChanOpenInit fails - channel is already active",
			func() {
				portID, err := icatypes.GeneratePortID(owner, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
				suite.Require().NoError(err)

				suite.GetICAApp(suite.chainA).ICAControllerKeeper.SetActiveChannelID(suite.chainA.GetContext(), portID, path.EndpointA.ChannelID)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			owner = TestOwnerAddress // must be explicitly changed

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			tc.malleate() // malleate mutates test data

			msgSrv := keeper.NewMsgServerImpl(suite.GetICAApp(suite.chainA).InterTxKeeper)
			msg := types.NewMsgRegisterAccount(owner, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)

			res, err := msgSrv.RegisterAccount(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}

		})
	}
}

func (suite *KeeperTestSuite) TestSubmitTx() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			icaAppA := suite.GetICAApp(suite.chainA)
			icaAppB := suite.GetICAApp(suite.chainB)

			owner := TestOwnerAddress // must be explicitly changed
			addr, err := sdk.AccAddressFromBech32(owner)
			suite.Require().NoError(err)

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err = SetupICAPath(path, icaAppA, owner)
			suite.Require().NoError(err)

			portID, err := icatypes.GeneratePortID(TestOwnerAddress, ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
			suite.Require().NoError(err)

			// Get the address of the interchain account stored in state during handshake step
			interchainAccountAddr, found := suite.GetICAApp(suite.chainA).ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), portID)
			suite.Require().True(found)

			icaAddr, err := sdk.AccAddressFromBech32(interchainAccountAddr)
			suite.Require().NoError(err)

			// Check if account is created
			interchainAccount := icaAppB.AccountKeeper.GetAccount(suite.chainB.GetContext(), icaAddr)
			suite.Require().Equal(interchainAccount.GetAddress().String(), interchainAccountAddr)

			// Fund interchain account
			suite.fundICAWallet(interchainAccountAddr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10000))))

			// Create bank transfer message to execute on the host
			icaMsg := &banktypes.MsgSend{
				FromAddress: interchainAccountAddr,
				ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
				Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			}

			// Get initial balance of recipient account on host chain
			initialBalance := icaAppB.BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			msgSrv := keeper.NewMsgServerImpl(icaAppA.InterTxKeeper)
			msg, err := types.NewMsgSubmitTx(addr, icaMsg, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
			suite.Require().NoError(err)

			res, err := msgSrv.SubmitTx(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)
			suite.Require().NoError(err)

			// TODO

			balance := icaAppB.BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
			suite.Require().Equal(sdk.NewInt(100).Int64(), balance.Amount.Sub(initialBalance.Amount).Int64())

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *KeeperTestSuite) fundICAWallet(interchainAccountAddr string, amount sdk.Coins) {
	msgBankSend := &banktypes.MsgSend{
		FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      amount,
	}

	res, err := suite.chainB.SendMsgs(msgBankSend)
	suite.Require().NotEmpty(res)
	suite.Require().NoError(err)
}
