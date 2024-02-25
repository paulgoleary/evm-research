
#### Note that neither Heimdall checkpoints or milestones include the block state root in their hashing scheme.
* It is strongly assumed that we will need to hardfork to incorporate the state root into the checkpoint / milestone mechanism, since this is generally the basis for state inclusion proofs.
* GetRootHash code is here in Bor: https://github.com/maticnetwork/bor/blob/241af1fa1e74fa8ef911c4391b3e913bced04452/consensus/bor/api.go#L318
* I *believe* GetRootHash is used for both checkpoints and milestones.
* All the analysis below proceeds as if the state roots will be included in the checkpoint hash.

#### Overall (possible) Flow

* Identify the Bor block with the desired state root hash.
* I'm assuming we can base proofs off of milestones - they are voted on in the same way as checkpoints. Further, milestones will have less blocks in them and therefore should be somewhat more efficient to calculate in a ZK circuit.
* Find the transaction and its hash for the milestone that contains the target block.
  * So far I have not found a direct way to do this. The required data is emitted in events from the milestone transaction; if we find the event, we know the hash. Unless there is another mechanism, it may be necessary to create an index of `block_num->tx_hash`.
* The transaction hash can be used to retrieve both the milestone transaction *and* its associated side transaction.
  * The top-level transaction will contain the root hash that is calculated from block header data in `GetRootHash`.
  * The side transaction contains the signatures and binary data that is actually signed. The signed data includes the root hash and other data. This is also basically the same type of data that is passed to the root chain contract when submitting checkpoints.
* The milestone root hash will be used to prove that a given state root hash (once included) was present in a block.
* The signatures from the side transaction can be used to prove that a given set of validators attested to the milestone root hash.
* I assume we will need some form contract on the root chain that can be used to validate that the attesting validators had sufficient state.
  * *Lots* of important details here are TBD... 

So far there has been some preliminary POC coding on how these structures and data can be mapped into Rust and then a proving system like SP1.

Current code is here: https://github.com/paulgoleary/polygon-pos-light