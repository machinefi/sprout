use std::fmt::Debug;
use std::hash::Hash;

use halo2_proofs::{
    circuit::{AssignedCell, Layouter, Region, Value},
    halo2curves::{
        bn256::{Fq as BnBase, Fr as BnScalar},
        ff::{FromUniformBytes, PrimeField},
        secp256k1::{Fp as SecpBase, Fq as SecpScalar},
    },
    plonk::{Advice, Column, ConstraintSystem, Error, Fixed, Instance, Selector, TableColumn},
};

pub mod circuit;
pub mod gadgets;
pub mod integer;
pub mod params;
pub mod poseidon;
pub mod utils;

/// Extention to the traits provided by halo2
pub trait FieldExt: PrimeField + FromUniformBytes<64> + Hash {}
impl FieldExt for BnBase {}
impl FieldExt for BnScalar {}
impl FieldExt for SecpBase {}
impl FieldExt for SecpScalar {}

/// Hasher trait
pub trait Hasher<F: FieldExt, const WIDTH: usize> {
    /// Creates a new hasher
    fn new(inputs: [F; WIDTH]) -> Self;
    /// Finalize the hasher
    fn finalize(&self) -> [F; WIDTH];
}

/// Sponge Hasher trait
pub trait SpongeHasher<F: FieldExt>: Clone {
    /// Creates a new sponge hasher
    fn new() -> Self;
    /// Update current sponge state
    fn update(&mut self, inputs: &[F]);
    /// Finalize the sponge hasher
    fn squeeze(&mut self) -> F;
}

/// Hasher chipset trait
pub trait HasherChipset<F: FieldExt, const WIDTH: usize>: Chipset<F> + Clone {
    /// Creates a new hasher chipset
    fn new(inputs: [AssignedCell<F, F>; WIDTH]) -> Self;
    /// Configure
    fn configure(common: &CommonConfig, meta: &mut ConstraintSystem<F>) -> Self::Config;
    /// Finalize the hasher
    fn finalize(
        self,
        common: &CommonConfig,
        config: &Self::Config,
        layouter: impl Layouter<F>,
    ) -> Result<[AssignedCell<F, F>; WIDTH], Error>;
}

/// Sponge Hasher chipset trait
pub trait SpongeHasherChipset<F: FieldExt>: Clone + Debug {
    /// Config selectors for the sponge
    type Config: Clone + Debug;
    /// Creates a new sponge hasher chipset
    fn init(common: &CommonConfig, layouter: impl Layouter<F>) -> Result<Self, Error>;
    /// Configure
    fn configure(common: &CommonConfig, meta: &mut ConstraintSystem<F>) -> Self::Config;
    /// Update current sponge chipset state
    fn update(&mut self, inputs: &[AssignedCell<F, F>]);
    /// Finalize the sponge hasher
    fn squeeze(
        &mut self,
        common: &CommonConfig,
        config: &Self::Config,
        layouter: impl Layouter<F>,
    ) -> Result<AssignedCell<F, F>, Error>;
}

#[derive(Debug)]
/// Region Context struct for managing region assignments
pub struct RegionCtx<'a, F: FieldExt> {
    /// Region struct
    region: Region<'a, F>,
    /// Current row offset
    offset: usize,
}

impl<'a, F: FieldExt> RegionCtx<'a, F> {
    /// Construct new Region Context
    pub fn new(region: Region<'a, F>, offset: usize) -> RegionCtx<'a, F> {
        RegionCtx { region, offset }
    }

    /// Return current row offset
    pub fn offset(&self) -> usize {
        self.offset
    }

    /// Turn into region struct
    pub fn into_region(self) -> Region<'a, F> {
        self.region
    }

    /// Assign value to a fixed column
    pub fn assign_fixed(
        &mut self,
        column: Column<Fixed>,
        value: F,
    ) -> Result<AssignedCell<F, F>, Error> {
        self.region.assign_fixed(
            || format!("fixed_{}", self.offset),
            column,
            self.offset,
            || Value::known(value),
        )
    }

    /// Assign to advice column from an instance column
    pub fn assign_from_instance(
        &mut self,
        advice: Column<Advice>,
        instance: Column<Instance>,
        index: usize,
    ) -> Result<AssignedCell<F, F>, Error> {
        self.region.assign_advice_from_instance(
            || format!("advice_{}", self.offset),
            instance,
            index,
            advice,
            self.offset,
        )
    }

    /// Assign to advice column from an fixed column
    pub fn assign_from_constant(
        &mut self,
        advice: Column<Advice>,
        constant: F,
    ) -> Result<AssignedCell<F, F>, Error> {
        self.region.assign_advice_from_constant(
            || format!("advice_{}", self.offset),
            advice,
            self.offset,
            constant,
        )
    }

    /// Assign value to an advice column
    pub fn assign_advice(
        &mut self,
        column: Column<Advice>,
        value: Value<F>,
    ) -> Result<AssignedCell<F, F>, Error> {
        self.region.assign_advice(
            || format!("advice_{}", self.offset),
            column,
            self.offset,
            || value,
        )
    }

    /// Copy value from passed assigned cell into an advice column
    pub fn copy_assign(
        &mut self,
        column: Column<Advice>,
        value: AssignedCell<F, F>,
    ) -> Result<AssignedCell<F, F>, Error> {
        value.copy_advice(
            || format!("advice_{}", self.offset),
            &mut self.region,
            column,
            self.offset,
        )
    }

    /// Constrain two cells to be equal
    pub fn constrain_equal(
        &mut self,
        a_cell: AssignedCell<F, F>,
        b_cell: AssignedCell<F, F>,
    ) -> Result<(), Error> {
        self.region.constrain_equal(a_cell.cell(), b_cell.cell())
    }

    /// Constrain a cell to be equal to a constant
    pub fn constrain_to_constant(
        &mut self,
        a_cell: AssignedCell<F, F>,
        constant: F,
    ) -> Result<(), Error> {
        self.region.constrain_constant(a_cell.cell(), constant)
    }

    /// Enable selector at the current row offset
    pub fn enable(&mut self, selector: Selector) -> Result<(), Error> {
        selector.enable(&mut self.region, self.offset)
    }

    /// Increment the row offset
    pub fn next(&mut self) {
        self.offset += 1
    }
}

/// Number of advice columns in common config
pub const ADVICE: usize = 20;
/// Number of fixed columns in common config
pub const FIXED: usize = 10;

/// Common config for the whole circuit
#[derive(Clone, Debug)]
pub struct CommonConfig {
    /// Advice columns
    pub advice: [Column<Advice>; ADVICE],
    /// Fixed columns
    fixed: [Column<Fixed>; FIXED],
    /// Table column
    #[allow(dead_code)]
    table: TableColumn,
    /// Instance column
    pub instance: Column<Instance>,
}

impl CommonConfig {
    /// Create a new `CommonConfig` columns
    pub fn new<F: FieldExt>(meta: &mut ConstraintSystem<F>) -> Self {
        let advice = [(); ADVICE].map(|_| meta.advice_column());
        let fixed = [(); FIXED].map(|_| meta.fixed_column());
        let table = meta.lookup_table_column();
        let instance = meta.instance_column();

        advice.map(|c| meta.enable_equality(c));
        fixed.map(|c| meta.enable_constant(c));
        meta.enable_equality(instance);

        Self {
            advice,
            fixed,
            table,
            instance,
        }
    }
}

/// Trait for an atomic chip implementation
/// Each chip uses common config columns, but has its own selector
pub trait Chip<F: FieldExt> {
    /// Output of the synthesis
    type Output: Clone;
    /// Gate configuration, using common config columns
    fn configure(common: &CommonConfig, meta: &mut ConstraintSystem<F>) -> Selector;
    /// Chip synthesis. This function can return an assigned cell to be used
    /// elsewhere in the circuit
    fn synthesize(
        self,
        common: &CommonConfig,
        selector: &Selector,
        layouter: impl Layouter<F>,
    ) -> Result<Self::Output, Error>;
}

/// Chipset uses a collection of chips as primitives to build more abstract
/// circuits
pub trait Chipset<F: FieldExt> {
    /// Config used for synthesis
    type Config: Clone;
    /// Output of the synthesis
    type Output: Clone;
    /// Chipset synthesis. This function can have multiple smaller chips
    /// synthesised inside. Also can returns an assigned cell.
    fn synthesize(
        self,
        common: &CommonConfig,
        config: &Self::Config,
        layouter: impl Layouter<F>,
    ) -> Result<Self::Output, Error>;
}

/// UnassignedValue Trait
pub trait UnassignedValue {
    /// Returns unknown value type
    fn without_witnesses(&self) -> Self;
}
