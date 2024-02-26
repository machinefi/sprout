use std::rc::Rc;

use halo2_curves::bn256::{Bn256, Fq, Fr, G1Affine};
use halo2_proofs::{
    dev::MockProver,
    plonk::{create_proof, keygen_pk, keygen_vk, verify_proof, Circuit, ProvingKey},
    poly::{
        commitment::{Params, ParamsProver},
        kzg::{
            commitment::{KZGCommitmentScheme, ParamsKZG},
            multiopen::{ProverGWC, VerifierGWC},
            strategy::AccumulatorStrategy,
        },
        VerificationStrategy,
    },
    transcript::{TranscriptReadBuffer, TranscriptWriterBuffer},
};
use itertools::Itertools;
use rand::rngs::OsRng;
use snark_verifier::{
    loader::evm::EvmLoader,
    pcs::kzg::{Gwc19, KzgAs},
    system::halo2::{compile, transcript::evm::EvmTranscript, Config},
    verifier::{self, SnarkVerifier},
};

#[derive(Debug)]
pub enum Error {
    Proof(String),
    Verify(String),
}

type PlonkVerifier = verifier::plonk::PlonkVerifier<KzgAs<Bn256, Gwc19>>;

pub fn gen_srs(k: u32) -> ParamsKZG<Bn256> {
    ParamsKZG::<Bn256>::setup(k, OsRng)
}

pub fn gen_pk<C: Circuit<Fr>>(params: &ParamsKZG<Bn256>, circuit: &C) -> ProvingKey<G1Affine> {
    let vk = keygen_vk(params, circuit).unwrap();
    keygen_pk(params, vk, circuit).unwrap()
}

pub fn gen_sol_verifier<C: Circuit<Fr>>(
    params: &ParamsKZG<Bn256>,
    empty_circuit: C,
    num_instance: Vec<usize>,
) -> Result<String, Error> {
    let pk = gen_pk(params, &empty_circuit);

    let protocol = compile(
        params,
        pk.get_vk(),
        Config::kzg().with_num_instance(num_instance.clone()),
    );
    let vk = (params.get_g()[0], params.g2(), params.s_g2()).into();

    let loader = EvmLoader::new::<Fq, Fr>();
    let protocol = protocol.loaded(&loader);
    let mut transcript = EvmTranscript::<_, Rc<EvmLoader>, _, _>::new(&loader);

    let instances = transcript.load_instances(num_instance);
    let proof = match PlonkVerifier::read_proof(&vk, &protocol, &instances, &mut transcript) {
        Ok(proof) => proof,
        Err(_) => return Err(Error::Proof(String::from("Generate proof error"))),
    };

    if PlonkVerifier::verify(&vk, &protocol, &instances, &proof).is_err() {
        return Err(Error::Verify(String::from("Verify proof error")));
    }

    Ok(loader.solidity_code())
}

pub fn gen_proof<C: Circuit<Fr>>(
    params: &ParamsKZG<Bn256>,
    pk: &ProvingKey<G1Affine>,
    circuit: C,
    instances: &[Vec<Fr>],
) -> Vec<u8> {
    MockProver::run(params.k(), &circuit, instances.to_vec())
        .unwrap()
        .assert_satisfied();

    let instances = instances
        .iter()
        .map(|instances| instances.as_slice())
        .collect_vec();
    let proof = {
        let mut transcript = TranscriptWriterBuffer::<_, G1Affine, _>::init(Vec::new());
        create_proof::<KZGCommitmentScheme<Bn256>, ProverGWC<_>, _, _, EvmTranscript<_, _, _, _>, _>(
            params,
            pk,
            &[circuit],
            &[instances.as_slice()],
            OsRng,
            &mut transcript,
        )
        .unwrap();
        transcript.finalize()
    };

    let accept = {
        let mut transcript = TranscriptReadBuffer::<_, G1Affine, _>::init(proof.as_slice());
        VerificationStrategy::<_, VerifierGWC<_>>::finalize(
            verify_proof::<_, VerifierGWC<_>, _, EvmTranscript<_, _, _, _>, _>(
                params.verifier_params(),
                pk.get_vk(),
                AccumulatorStrategy::new(params.verifier_params()),
                &[instances.as_slice()],
                &mut transcript,
            )
            .unwrap(),
        )
    };
    assert!(accept);

    proof
}
