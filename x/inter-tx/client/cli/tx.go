package cli

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/interchain-accounts/x/inter-tx/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagAbsoluteTimeouts       = "absolute-timeouts"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// this line is used by starport scaffolding # 1
	cmd.AddCommand(
		getRegisterAccountCmd(),
		getSendTxCmd(),
	)

	return cmd
}

func getRegisterAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "register --connection-id --counterparty-connection-id",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			connectionId := viper.GetString(FlagConnectionId)
			counterpartyConnectionId := viper.GetString(FlagCounterpartyConnectionId)

			msg := types.NewMsgRegisterAccount(
				clientCtx.GetFromAddress().String(),
				connectionId,
				counterpartyConnectionId,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(fsConnectionId)
	_ = cmd.MarkFlagRequired(FlagConnectionId)
	_ = cmd.MarkFlagRequired(FlagCounterpartyConnectionId)

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func getSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "send [interchain_account_address] --connection-id --counterparty-connection-id",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ownerAddr := clientCtx.GetFromAddress()
			interchainAccountAddr := args[0]

			connectionId := viper.GetString(FlagConnectionId)
			counterpartyConnectionId := viper.GetString(FlagCounterpartyConnectionId)

			amount, err := sdk.ParseCoinsNormalized("1uatom,1uphoton")
			if err != nil {
				return err
			}
			interchainMsg := bank.NewMsgSend(ownerAddr, ownerAddr, amount)

			msg := types.NewMsgSend(
				interchainAccountAddr,
				ownerAddr,
				interchainMsg,
				connectionId,
				counterpartyConnectionId,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(fsConnectionId)

	_ = cmd.MarkFlagRequired(FlagConnectionId)
	_ = cmd.MarkFlagRequired(FlagCounterpartyConnectionId)

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
