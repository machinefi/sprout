// Copyright 2023 RISC Zero, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#![no_main]
// #![no_std]

use risc0_zkvm::guest::env;
use serde_json::Value as JsonValue;

risc0_zkvm::guest::entry!(main);

pub fn main() {
    let input: String = env::read();

    // the string is a json array
    let input_v: JsonValue = serde_json::from_str(&input).unwrap();
    let item_str = input_v.as_array().unwrap()[0].as_str().unwrap();
    let v: JsonValue = serde_json::from_str(item_str).unwrap();
 
    // Load the first number from the host, is a private key
    let a: String = v["private_input"].as_str().unwrap().to_string();
    // Load the second number from the host, is a public key
    let b: String = v["public_input"].as_str().unwrap().to_string();

    let pri_a = a.trim().parse::<u64>().unwrap();
    let mut pub_b: u64 = 0;
    let mut pub_c: u64 = 0;

    let pub_ver: Result<Vec<u64>, _> = b.split(",").map(|s| s.trim().parse::<u64>()).collect();
    match pub_ver {
        Ok(v) => (pub_b, pub_c) = (v[0], v[1]),
        Err(e) => {
            env::log(&format!("public input parse error, Error: {:?}", e));
        }
    };

    if pri_a > pub_b && pri_a < pub_c {
        let s = format!(
            "I know your private input is greater than {} and less than {}, and I can prove it!",
            pub_b, pub_c
        );
        env::commit(&s);
    } else {
        let s = format!(
            "I know your private input is not greater than {} or less than {}, and I can not prove it!",
            pub_b, pub_c
        );
        env::commit(&s);
    }
}
