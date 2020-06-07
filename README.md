ckb-script-deployer
===================

## Build

```bash
go mod download
go build .
```

## Usage

### Deploy

```bash
./ckb-script-deployer deploy -u http://localhost:8114 -k YOUR_PRIVATE_KEY -b COMPILED_BINARY_FILE
```

### Create Dep Group

```bash
./ckb-script-deployer dep_group -u http://localhost:8114 -k YOUR_PRIVATE_KEY -f DEP_GROUP_FILE
```
