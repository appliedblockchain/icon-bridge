use test_helper::actions::create_account;
use test_helper::types::{Context, Signer};

// * * * * * * * * * * * * * *
// * * * * * * * * * * * * * *
// *   Create Account  * * * *
// * * * * * * * * * * * * * *
// * * * * * * * * * * * * * *

pub static CHARLIES_ACCOUNT_IS_CREATED: fn(Context) -> Context = |mut context: Context| {
    let charlie = Signer::default();
    create_account(&charlie);
    context.accounts_mut().add("charlie", &charlie);
    context
};

pub static CHUCKS_ACCOUNT_IS_CREATED: fn(Context) -> Context = |mut context: Context| {
    let chuck = Signer::default();
    create_account(&chuck);
    context.accounts_mut().add("chuck", &chuck);
    context
};

pub static BOBS_ACCOUNT_IS_CREATED: fn(Context) -> Context = |mut context: Context| {
    let bob = Signer::default();
    create_account(&bob);
    context.accounts_mut().add("bob", &bob);
    context
};

// * * * * * * * * * * * * * *
// * * * * * * * * * * * * * *
// * * * *  Set Signer * * * *
// * * * * * * * * * * * * * *
// * * * * * * * * * * * * * *

pub static BMC_OWNER_IS_THE_SIGNER: fn(Context) -> Context = |mut context: Context| {
    let signer = context.contracts().get("bmc").to_owned();
    context.set_signer(&signer);
    context
};

pub static ALICE_IS_THE_SIGNER: fn(Context) -> Context = |mut context: Context| {
    let signer = context.accounts().get("alice").to_owned();
    context.set_signer(&signer);
    context
};

pub static BOB_IS_THE_SIGNER: fn(Context) -> Context = |mut context: Context| {
    let signer = context.accounts().get("bob").to_owned();
    context.set_signer(&signer);
    context
};

pub static CHUCK_IS_THE_SIGNER: fn(Context) -> Context = |mut context: Context| {
    let signer = context.accounts().get("chuck").to_owned();
    context.set_signer(&signer);
    context
};
