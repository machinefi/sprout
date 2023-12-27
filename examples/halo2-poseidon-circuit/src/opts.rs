use clap::{Parser, Subcommand};

#[derive(Debug, Parser)]
#[clap(name = "halo2-circuit", version = "0.1.0")]
pub struct Opts {
    #[clap(subcommand)]
    pub sub: Subcommands,
}

#[derive(Debug, Subcommand)]
pub enum Subcommands {
    #[clap(name = "solidity")]
    #[clap(about = "Generate verifier solidity contract.")]
    Solidity {
        #[clap(
            long,
            short,
            value_name = "file",
            default_value = "Verifier.sol"
        )]
        file: String,
    },

    #[clap(name = "verify")]
    #[clap(about = "Local verify proof.")]
    Verfiy {
        #[clap(long, value_name = "proof-file")]
        proof: String,
        #[clap(long, value_name = "public-input")]
        public: u64,
    },

    #[clap(name = "proof")]
    #[clap(about = "gen proof.")]
    Proof {
        #[clap(
            long,
            short,
            value_name = "file",
            default_value = "proof.json"
        )]
        file: String,
    },
}
