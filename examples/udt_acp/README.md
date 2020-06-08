Deploy sUDT and Anyone can pay
==============================

## 1. Deploy sUDT

```bash
./ckb-script-deployer deploy -u http://localhost:8114 -k PRIVATE_KEY -b udt
```

## 2. Deploy anyone can pay

### 2.1 Deploy anyone can pay script

```bash
./ckb-script-deployer deploy -u http://localhost:8114 -k PRIVATE_KEY -b anyone_can_pay
```

### 2.2 Create dep_groups

> Replace anyone can pay transaction hash in `dep_group.yaml`

```bash
./ckb-script-deployer dep_group -u http://localhost:8114 -k PRIVATE_KEY 
```
