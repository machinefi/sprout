use std::ops::{Shr, Sub};

use halo2_curves::{bn256::Fr, ff::PrimeField};
use primitive_types::U256;

use crate::eigentrust::{
    params::hasher::poseidon_bn254_5x5::Params, poseidon::native::sponge::PoseidonSponge,
};

const WIDTH: usize = 5;

pub fn init_intput() -> (Vec<Fr>, Fr, Fr) {
    let inputs: Vec<Fr> = [
        "0x0000000000000000000000000000000000000000000000000000000000000000",
        "0x0000000000000000000000000000000000000000000000000000000000000001",
        "0x0000000000000000000000000000000000000000000000000000000000000002",
        "0x0000000000000000000000000000000000000000000000000000000000000003",
        "0x0000000000000000000000000000000000000000000000000000000000000004",
        "0x0000000000000000000000000000000000000000000000000000000000000005",
        "0x0000000000000000000000000000000000000000000000000000000000000006",
        "0x0000000000000000000000000000000000000000000000000000000000000007",
        "0x0000000000000000000000000000000000000000000000000000000000000008",
        "0x0000000000000000000000000000000000000000000000000000000000000009",
    ]
    .iter()
    .map(|n| super::circuit::hex_to_field(n))
    .collect();

    let mut native_poseidon_sponge = PoseidonSponge::<Fr, WIDTH, Params>::new();
    native_poseidon_sponge.update(&inputs);
    let hash_result = native_poseidon_sponge.squeeze();

    let difficulty_shift: i32 = 2; // =  powerc contract 
    let difficulty = (!U256::from(0)).shr(difficulty_shift); // = circuit  public input

    let hash_result_num = U256::from_little_endian(&hash_result.to_repr()[..]);

    if hash_result_num > difficulty {
        panic!("hash_result_num > difficulty");
    }

    (
        inputs,
        super::circuit::u256_to_field(&difficulty.sub(hash_result_num)),
        super::circuit::u256_to_field(&difficulty),
    )
}
