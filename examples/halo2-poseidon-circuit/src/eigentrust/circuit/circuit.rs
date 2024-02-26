use eth_types::Field;
use halo2_proofs::{
    circuit::{Layouter, SimpleFloorPlanner, Value},
    plonk::{Advice, Circuit, Column, ConstraintSystem, Error, Selector},
    poly::Rotation,
};
use primitive_types::U256;

use crate::eigentrust::{
    gadgets::absorb::AbsorbChip,
    params::hasher::poseidon_bn254_5x5::Params,
    poseidon::{
        sponge::PoseidonSpongeChipset, sponge::PoseidonSpongeConfig, FullRoundChip,
        PartialRoundChip, PoseidonConfig,
    },
    Chip, Chipset, CommonConfig, FieldExt, RegionCtx, ADVICE,
};

const WIDTH: usize = 5;

/// IntegratedCircuit is to check the rewards are expected with three tiers
#[derive(Default, Clone)]
pub struct IntegratedCircuit<F: FieldExt> {
    pub input: Vec<Value<F>>,
    pub diff: Value<F>,
    pub _marker: std::marker::PhantomData<F>,
}

#[derive(Clone)]
pub struct IntegratedConfig {
    advice: Column<Advice>,
    enable: Selector,
    poseidon_common_cfg: CommonConfig,
    poseidon_sponge_cfg: PoseidonSpongeConfig,
}

impl<F: FieldExt> Circuit<F> for IntegratedCircuit<F> {
    type Config = IntegratedConfig;
    type FloorPlanner = SimpleFloorPlanner;

    fn without_witnesses(&self) -> Self {
        Self::default()
    }

    fn configure(meta: &mut ConstraintSystem<F>) -> Self::Config {
        let advice = meta.advice_column();
        meta.enable_equality(advice);

        let enable = meta.complex_selector();
        meta.create_gate("verify", |meta| {
            let enable = meta.query_selector(enable);
            let input = meta.query_advice(advice, Rotation::cur());
            let diff = meta.query_advice(advice, Rotation::next());
            let output = meta.query_advice(advice, Rotation(2));

            vec![enable * (input + diff - output)]
        });

        // poseidon
        let common = CommonConfig::new(meta);
        let absorb_selector = AbsorbChip::<_, WIDTH>::configure(&common, meta);
        let pr_selector = PartialRoundChip::<F, WIDTH, Params>::configure(&common, meta);
        let fr_selector = FullRoundChip::<F, WIDTH, Params>::configure(&common, meta);
        let poseidon = PoseidonConfig::new(fr_selector, pr_selector);
        let sponge = PoseidonSpongeConfig::new(poseidon, absorb_selector);

        IntegratedConfig {
            advice,
            enable,
            poseidon_common_cfg: common,
            poseidon_sponge_cfg: sponge,
        }
    }

    fn synthesize(
        &self,
        config: Self::Config,
        mut layouter: impl Layouter<F>,
    ) -> Result<(), Error> {
        let (inputs, zero) = layouter.assign_region(
            || "load_inputs",
            |region| {
                let mut ctx = RegionCtx::new(region, 0);

                let mut advice_i = 0;
                let mut assigned_inputs = Vec::new();
                for inp in &self.input {
                    let assn_inp =
                        ctx.assign_advice(config.poseidon_common_cfg.advice[advice_i], *inp)?;
                    assigned_inputs.push(assn_inp);

                    advice_i += 1;
                    if advice_i % ADVICE == 0 {
                        advice_i = 0;
                        ctx.next();
                    }
                }

                let zero =
                    ctx.assign_from_constant(config.poseidon_common_cfg.advice[advice_i], F::ZERO)?;
                Ok((assigned_inputs, zero))
            },
        )?;

        let zero_state = [(); WIDTH].map(|_| zero.clone());
        let mut poseidon_sponge =
            PoseidonSpongeChipset::<F, WIDTH, Params>::new(zero_state, zero.clone());
        poseidon_sponge.update(&inputs);
        let result_state = poseidon_sponge.synthesize(
            &config.poseidon_common_cfg,
            &config.poseidon_sponge_cfg,
            layouter.namespace(|| "poseidon_sponge"),
        )?;

        layouter.assign_region(
            || "verify",
            |mut region| {
                config.enable.enable(&mut region, 0)?;
                result_state[0].copy_advice(|| "input", &mut region, config.advice, 0)?;
                region.assign_advice(|| "diff", config.advice, 1, || self.diff)?;
                region.assign_advice_from_instance(
                    || "output",
                    config.poseidon_common_cfg.instance,
                    0,
                    config.advice,
                    2,
                )?;

                Ok(())
            },
        )?;

        Ok(())
    }
}

/// Returns congruent field element for the given hex string.
pub fn hex_to_field<F: Field>(s: &str) -> F {
    let s = &s[2..];
    let mut bytes = hex::decode(s).expect("Invalid params");
    bytes.reverse();
    let mut bytes_wide: [u8; 64] = [0; 64];
    bytes_wide[..bytes.len()].copy_from_slice(&bytes[..]);
    F::from_uniform_bytes(&bytes_wide)
}

pub fn u256_to_field<F: Field>(num: &U256) -> F {
    let mut bytes_wide: [u8; 64] = [0; 64];
    num.to_little_endian(&mut bytes_wide[0..32]);
    F::from_uniform_bytes(&bytes_wide)
}

#[cfg(test)]
mod tests {
    use crate::eigentrust::circuit::test::init_intput;

    use super::*;

    use halo2_curves::bn256::Fr;
    use halo2_proofs::dev::MockProver;

    #[test]
    fn test_ciruict_success() {
        let k = 8;

        let (inputs, diff, difficulty) = init_intput();

        // Instantiate the circuit with the private inputs.
        let circuit = IntegratedCircuit::<Fr> {
            input: inputs.iter().map(|x| Value::known(*x)).collect(),
            diff: Value::known(diff),
            _marker: Default::default(),
        };

        // Given the correct public input, our circuit will verify.
        let prover = MockProver::run(k, &circuit, vec![vec![difficulty]]).unwrap();

        prover.assert_satisfied();
    }
}
