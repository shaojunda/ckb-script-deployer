package cmd

import (
	"context"
	"fmt"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"io/ioutil"
	"math"

	"github.com/nervosnetwork/ckb-sdk-go/crypto/secp256k1"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/nervosnetwork/ckb-sdk-go/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	depURL        *string
	depKey        *string
	depConfigFile *string
	indexURL      *string
)

type DepGroupConfig struct {
	TxHash string `yaml:"txHash"`
	Index  uint   `yaml:"index"`
}

var depCmd = &cobra.Command{
	Use:   "dep_group",
	Short: "create dep_group",
	Long:  `Create dep_group transaction.`,
	Run: func(cmd *cobra.Command, args []string) {
		var c []DepGroupConfig

		file, err := ioutil.ReadFile(*depConfigFile)
		if err != nil {
			Fatalf("read %s error: %v", *depConfigFile, err)
		}
		err = yaml.Unmarshal(file, &c)
		if err != nil {
			Fatalf("decode %s error: %v", *depConfigFile, err)
		}

		var deps [][]byte

		for _, item := range c {
			dep := &types.OutPoint{
				TxHash: types.HexToHash(item.TxHash),
				Index:  item.Index,
			}
			depBytes, err := dep.Serialize()
			if err != nil {
				Fatalf("serialize dep error: %v", err)
			}
			deps = append(deps, depBytes)
		}
		data := types.SerializeFixVec(deps)

		client, err := rpc.DialWithIndexer(*depURL, *indexURL)
		if err != nil {
			Fatalf("create rpc client error: %v", err)
		}

		key, err := secp256k1.HexToKey(*depKey)
		if err != nil {
			Fatalf("import private key error: %v", err)
		}

		scripts, err := utils.NewSystemScripts(client)
		if err != nil {
			Fatalf("load system script error: %v", err)
		}

		change, err := key.Script(scripts)

		capacity := uint64(len(data)+61) * uint64(math.Pow10(8))
		searchKey := &indexer.SearchKey{
			Script:     change,
			ScriptType: indexer.ScriptTypeLock,
		}

		cellCollector := utils.NewLiveCellCollector(client, searchKey, indexer.SearchOrderAsc, 1000, "", utils.NewCapacityLiveCellProcessor(capacity+100000000))
		cells, err := cellCollector.Collect()
		if err != nil {
			Fatalf("collect cell error: %v", err)
		}
		if cells.Capacity < capacity+100000000 {
			Fatalf("insufficient capacity: %d < %d", cells.Capacity, capacity+100000000)
		}

		tx := transaction.NewSecp256k1SingleSigTx(scripts)
		tx.Outputs = append(tx.Outputs, &types.CellOutput{
			Capacity: uint64(capacity),
			Lock:     change,
		})
		tx.OutputsData = append(tx.OutputsData, data)

		if cells.Capacity-capacity-100000000 > 61 {
			tx.Outputs = append(tx.Outputs, &types.CellOutput{
				Capacity: 0,
				Lock:     change,
			})
			tx.OutputsData = append(tx.OutputsData, []byte{})
		}
		var inputs []*types.CellInput
		for _, cell := range cells.LiveCells {
			inputs = append(inputs, &types.CellInput{
				Since:          0,
				PreviousOutput: cell.OutPoint,
			})
		}
		group, witnessArgs, err := transaction.AddInputsForTransaction(tx, inputs)
		if err != nil {
			Fatalf("add inputs to transaction error: %v", err)
		}

		fee, err := transaction.CalculateTransactionFee(tx, 1000)
		if err != nil {
			Fatalf("calculate transaction fee error: %v", err)
		}

		if len(tx.Outputs) > 1 {
			tx.Outputs[1].Capacity = cells.Capacity - capacity - fee
		} else {
			tx.Outputs[0].Capacity = cells.Capacity - fee
		}

		err = transaction.SingleSignTransaction(tx, group, witnessArgs, key)
		if err != nil {
			Fatalf("sign transaction error: %v", err)
		}

		hash, err := client.SendTransaction(context.Background(), tx)
		if err != nil {
			Fatalf("send transaction error: %v", err)
		}

		fmt.Printf(`Create dep_group info:
	txHash: %s
	index: 0
`, hash.String())
	},
}

func init() {
	rootCmd.AddCommand(depCmd)

	depURL = depCmd.Flags().StringP("url", "u", "http://localhost:8114", "RPC API server url")
	indexURL = depCmd.Flags().StringP("indexUrl", "i", "http://localhost:8116", "ckb-indexer url")
	depKey = depCmd.Flags().StringP("key", "k", "", "private key")
	depConfigFile = depCmd.Flags().StringP("file", "f", "dep_group.yaml", "dep_group config file path")
	_ = depCmd.MarkFlagRequired("key")
}
