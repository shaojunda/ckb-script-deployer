package cmd

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/ququzone/ckb-sdk-go/crypto/blake2b"
	"github.com/ququzone/ckb-sdk-go/types"
)

const (
	TypeIdCodeHash = "0x00000000000000000000000000000000000000000000000000545950455f4944"
)

func Fatalf(format string, v ...interface{}) {
	fmt.Printf(format, v)
	os.Exit(1)
}

func BuildTypeIdScript(input *types.CellInput, index uint64) (*types.Script, error) {
	data, err := input.Serialize()
	if err != nil {
		return nil, err
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, index)
	data = append(data, b...)

	data, err = blake2b.Blake256(data)
	if err != nil {
		return nil, err
	}

	return &types.Script{
		CodeHash: types.HexToHash(TypeIdCodeHash),
		HashType: types.HashTypeType,
		Args:     data,
	}, nil
}
