package cmd

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/ququzone/ckb-sdk-go/crypto/blake2b"
	"github.com/ququzone/ckb-sdk-go/crypto/secp256k1"
	"github.com/ququzone/ckb-sdk-go/rpc"
	"github.com/ququzone/ckb-sdk-go/transaction"
	"github.com/ququzone/ckb-sdk-go/types"
	"github.com/ququzone/ckb-sdk-go/utils"
	"github.com/spf13/cobra"
)

var (
	deployURL    *string
	deployKey    *string
	deployFile   *string
	deployMethod *string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy script",
	Long:  `Deploy CKB script.`,
	Run: func(cmd *cobra.Command, args []string) {
		dataFile, err := os.Open(*deployFile)
		if err != nil {
			Fatalf("open script binary file error: %v", err)
		}
		defer dataFile.Close()

		dataInfo, err := dataFile.Stat()
		if err != nil {
			Fatalf("load script binary info error: %v", err)
		}

		data := make([]byte, dataInfo.Size())
		_, err = dataFile.Read(data)
		if err != nil {
			Fatalf("read script binary  error: %v", err)
		}

		client, err := rpc.Dial(*deployURL)
		if err != nil {
			Fatalf("create rpc client error: %v", err)
		}

		key, err := secp256k1.HexToKey(*deployKey)
		if err != nil {
			Fatalf("import private key error: %v", err)
		}

		scripts, err := utils.NewSystemScripts(client)
		if err != nil {
			Fatalf("load system script error: %v", err)
		}

		var codeHash types.Hash

		change, err := key.Script(scripts)

		capacity := uint64(dataInfo.Size()+61+65) * uint64(math.Pow10(8))

		cellCollector := utils.NewCellCollector(client, change, utils.NewCapacityCellProcessor(capacity+100000000))
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

		if *deployMethod == "typeID" {
			typeIdScript, err := BuildTypeIdScript(&types.CellInput{
				Since:          0,
				PreviousOutput: cells.Cells[0].OutPoint,
			}, 0)
			if err != nil {
				Fatalf("build typeId script error: %v", err)
			}
			tx.Outputs[0].Type = typeIdScript
			codeHash, err = typeIdScript.Hash()
			if err != nil {
				Fatalf("CodeHash error")
			}
		} else {
			bytes, err := blake2b.Blake256(data)
			if err != nil {
				Fatalf("CodeHash error")
			}
			codeHash = types.BytesToHash(bytes)
		}

		tx.OutputsData = append(tx.OutputsData, data)

		if cells.Capacity-capacity-100000000 > 61 {
			tx.Outputs = append(tx.Outputs, &types.CellOutput{
				Capacity: 0,
				Lock:     change,
			})
			tx.OutputsData = append(tx.OutputsData, []byte{})
		}

		group, witnessArgs, err := transaction.AddInputsForTransaction(tx, cells.Cells)
		if err != nil {
			Fatalf("add inputs to transaction error: %v", err)
		}

		fee, err := transaction.CalculateTransactionFee(tx, 1100)
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

		fmt.Printf(`Deployed script info:
	txHash: %s
	index: 0
	CodeHash: %s
`, hash.String(), codeHash)
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployURL = deployCmd.Flags().StringP("url", "u", "http://localhost:8114", "RPC API server url")
	deployKey = deployCmd.Flags().StringP("key", "k", "", "Deploy private key")
	deployFile = deployCmd.Flags().StringP("binary", "b", "", "Compiled script binary file path")
	deployMethod = deployCmd.Flags().StringP("method", "m", "", "Deploy method data or typeID")
	_ = deployCmd.MarkFlagRequired("key")
	_ = deployCmd.MarkFlagRequired("binary")
}
