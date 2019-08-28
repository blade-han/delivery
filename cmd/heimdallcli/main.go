package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	tmTypes "github.com/tendermint/tendermint/types"

	"github.com/maticnetwork/heimdall/app"
	authCli "github.com/maticnetwork/heimdall/auth/client/cli"
	authTypes "github.com/maticnetwork/heimdall/auth/types"
	bank "github.com/maticnetwork/heimdall/bank/client/cli"
	bor "github.com/maticnetwork/heimdall/bor/cli"
	checkpoint "github.com/maticnetwork/heimdall/checkpoint/cli"
	hmTxCli "github.com/maticnetwork/heimdall/client/tx"
	"github.com/maticnetwork/heimdall/helper"
	staking "github.com/maticnetwork/heimdall/staking/cli"
	supply "github.com/maticnetwork/heimdall/supply/cli"
	hmTypes "github.com/maticnetwork/heimdall/types"
)

// rootCmd is the entry point for this binary
var (
	rootCmd = &cobra.Command{
		Use:   "heimdallcli",
		Short: "Heimdall light-client",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// initialise config
			initTendermintViperConfig(cmd)
			return nil
		},
	}
)

func initTendermintViperConfig(cmd *cobra.Command) {
	tendermintNode, _ := cmd.Flags().GetString(helper.NodeFlag)
	homeValue, _ := cmd.Flags().GetString(helper.HomeFlag)
	withHeimdallConfigValue, _ := cmd.Flags().GetString(helper.WithHeimdallConfigFlag)

	// set to viper
	viper.Set(helper.NodeFlag, tendermintNode)
	viper.Set(helper.HomeFlag, homeValue)
	viper.Set(helper.WithHeimdallConfigFlag, withHeimdallConfigValue)

	// start heimdall config
	helper.InitHeimdallConfig("")

	// set prefix
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(hmTypes.PrefixAccAddr, hmTypes.PrefixAccPub)
	config.SetBech32PrefixForValidator(hmTypes.PrefixValAddr, hmTypes.PrefixValPub)
	config.SetBech32PrefixForConsensusNode(hmTypes.PrefixConsAddr, hmTypes.PrefixConsPub)
	// config.Seal()
}

func main() {
	// disable sorting
	cobra.EnableCommandSorting = false

	// get the codec
	cdc := app.MakeCodec()
	ctx := server.NewDefaultContext()

	// TODO: Setup keybase, viper object, etc. to be passed into
	// the below functions and eliminate global vars, like we do
	// with the cdc.

	// chain id
	rootCmd.PersistentFlags().String(client.FlagChainID, "", "Chain ID of tendermint node")

	// add query/post commands (custom to binary)
	rootCmd.AddCommand(
		client.LineBreak,
		queryCmd(cdc),
		txCmd(cdc),
		client.LineBreak,
		keys.Commands(),
		exportCmd(ctx, cdc),
		convertAddressToHexCmd(cdc),
		convertHexToAddressCmd(cdc),
		client.LineBreak,
		version.VersionCmd,
	)

	// bind with-heimdall-config config with root cmd
	viper.BindPFlag(
		helper.WithHeimdallConfigFlag,
		rootCmd.Flags().Lookup(helper.WithHeimdallConfigFlag),
	)

	// prepare and add flags
	executor := cli.PrepareMainCmd(rootCmd, "HD", os.ExpandEnv("$HOME/.heimdalld"))
	err := executor.Execute()
	if err != nil {
		// Note: Handle with #870
		panic(err)
	}
}

func queryCmd(cdc *amino.Codec) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "Querying subcommands",
	}

	queryCmd.AddCommand(
		rpc.ValidatorCommand(cdc),
		rpc.BlockCommand(),
		tx.SearchTxCmd(cdc),
		tx.QueryTxCmd(cdc),
		client.LineBreak,
		authCli.GetAccountCmd(authTypes.StoreKey, cdc),

		// supply related queries
		supply.GetQueryCmd(cdc),
		// checkpoint related queries
		checkpoint.GetQueryCmd(cdc),
		// staking related get commands
		staking.GetQueryCmd(cdc),
	)

	return queryCmd
}

func txCmd(cdc *amino.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:   "tx",
		Short: "Transactions subcommands",
	}

	txCmd.AddCommand(
		authCli.GetSignCommand(cdc),
		hmTxCli.GetBroadcastCommand(cdc),
		hmTxCli.GetEncodeCommand(cdc),
		client.LineBreak,

		// get bank tx commands
		bank.GetTxCmd(cdc),
		// get bor tx commands
		bor.GetTxCmd(cdc),
		// get checkpoint tx commands
		checkpoint.GetTxCmd(cdc),
		// get staking tx commands
		staking.GetTxCmd(cdc),
	)

	return txCmd
}

func convertAddressToHexCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address-to-hex [address]",
		Short: "Convert address to hex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			fmt.Println("Hex:", ethCommon.BytesToAddress(key).String())
			return nil
		},
	}
	return client.GetCommands(cmd)[0]
}

func convertHexToAddressCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hex-to-address [hex]",
		Short: "Convert hex to address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := ethCommon.HexToAddress(args[0])
			fmt.Println("Address:", sdk.AccAddress(address.Bytes()).String())
			return nil
		},
	}
	return client.GetCommands(cmd)[0]
}

// exportCmd a state dump file
func exportCmd(ctx *server.Context, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-heimdall",
		Short: "Export genesis file with state-dump",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {

			// cliCtx := context.NewCLIContext().WithCodec(cdc)
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))

			// create chain id
			chainID := viper.GetString(client.FlagChainID)
			if chainID == "" {
				chainID = fmt.Sprintf("heimdall-%v", common.RandStr(6))
			}

			dataDir := path.Join(viper.GetString(cli.HomeFlag), "data")
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			db, err := sdk.NewLevelDB("application", dataDir)
			if err != nil {
				panic(err)
			}

			happ := app.NewHeimdallApp(logger, db)
			appState, _, err := happ.ExportAppStateAndValidators()
			if err != nil {
				panic(err)
			}

			err = writeGenesisFile(rootify("config/dump-genesis.json", config.RootDir), chainID, appState)
			if err == nil {
				fmt.Println("New genesis json file created:", rootify("config/dump-genesis.json", config.RootDir))
			}
			return err
		},
	}
	cmd.Flags().String(cli.HomeFlag, helper.DefaultNodeHome, "node's home directory")
	cmd.Flags().String(helper.FlagClientHome, helper.DefaultCLIHome, "client's home directory")
	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	return cmd
}

//
// Internal functions
//

func writeGenesisFile(genesisFile, chainID string, appState json.RawMessage) error {
	genDoc := tmTypes.GenesisDoc{
		ChainID:  chainID,
		AppState: appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genesisFile)
}

func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
