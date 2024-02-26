# DePIN SandBox Contracts

## Design

### NodeRegistry

Register node and operator contract, one node can only register once and will get an NFT, the NFT tokenId is node id. One operator address can only bind to one node, and through operator address can query node info.

### FleetManager

The contract that manage node and project relationship.

### W3bstreamRouter

The router that route node message to project recevier contract.

## Deployment

### Testnet

```
PROJECT_REGISTRY: 0x02feBE78F3A740b3e9a1CaFAA1b23a2ac0793D26
NodeRegistry: 0x16ca331641a9537e346e12C7403fDA014Da72F16
FleetManager: 0x8D3c113805f970839940546D5ef88afE98Ba76E4
W3bstreamRouter: 0x1BFf17c79b5fa910cC77e95Ca82C7De26fC3C3b0
```