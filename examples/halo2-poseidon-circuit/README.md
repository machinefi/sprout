Halo2 wasm template
==================

## Preparation
install `wasm-pack`
``` shell
curl https://rustwasm.github.io/wasm-pack/installer/init.sh -sSf | sh
```

install `solc`
Reference [install guid](https://docs.soliditylang.org/en/v0.8.9/installing-solidity.html)

## Build halo2 wasm program
1. get template 

``` shell
git clone git@github.com:machinefi/sprout.git && cd examples/halo2-circuit
```

2. build wasm

``` shell
wasm-pack build --target nodejs --out-dir pkg
```

you will find `halo2_simple_bg.wasm` in the `pkg` folder.

## Advanced
You can also develop your own halo2 circuit program.

1. Write a circuit according to the [halo2 development documentation](https://zcash.github.io/halo2/user/simple-example.html), and put the circuit file in `src/circuits`.
2. Replace the `TODO` in `src/lib.rs`.
3. Build wasm with `wasm-pack build --target nodejs --out-dir pkg`.

## Build executable file

``` shell
cargo build --release
```

After this command is successful, a `halo2-simple-circuit` executable file(executable file corresponding to the [simple circuit](./src/circuits/simple.rs)) will be generated in the `target/release` directory.

> **_NOTE:_**
> If you want to build an executable file corresponding to your own circuit, you need to replace the `TODO` in `src/main.rs`.

## Generate verify smart contract

``` shell
target/release/halo2-simple-circuit solidity
```
You will find `Verifier.sol` under the current folder. Or you can run `target/release/halo2-simple-circuit solidity -f path/filename.sol`.
Then you can deploy the smart contract to IoTeX chain or other ETH-compatible chains.

## Local verify proof
1. Get halo2 proof 
if you can send messages to znode successfully, then you can execute `ioctl ws message send --project-id 10001 --project-version "0.1" --data "{\"private_a\": 3, \"private_b\": 4}"` to obtain a halo2 proof, then put it in a file, like `halo2-simple-proof.json`.

2. verify
`--proof` is proof file, and `--public` is the public input

``` shell
target/release/halo2-simple-circuit verify --proof halo2-simple-proof.json --public 900
```
