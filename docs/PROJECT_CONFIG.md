# Project Configuration File

The Project Configuration file is a JSON array. The fields in the JSON include `code`, `codeExpParam`, `vmType`, `output`, `aggregation`, and `version`.

``` json
[
  {
    "code": "",
    "codeExpParam": "",
    "vmType": "",
    "output": {},
    "aggregation": {},
    "version": ""
  }
]
```

* `code` is a hex string converted from a circuit compressed by zlib.
* `codeExpParam` is the parameter used in the process of loading the circuit by zk-vm.
* `vmType` is the type of zk-vm, currently supporting halo2, risc0, and zkwasm.
* `output` is the output method of the proof, the default output is the terminal, but it can also output to smart contracts.
* `aggregation` can package multiple messages sent to sprout into one task, and this task will generate a proof.
* `version` indicates the version information of the current project.

[more project configuration file →](../test/project/10000)
