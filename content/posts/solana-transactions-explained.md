---
title: "How Solana Transactions Work: A Complete Guide"
date: 2025-01-25
draft: true
description: "A deep dive into Solana's transaction model: accounts, programs, instructions, and the entire lifecycle from submission to confirmation."
---

Ethereum has contracts. Solana has accounts.

This fundamental difference confuses everyone coming from Ethereum. On Solana, *everything* is an account — your wallet, tokens, smart contracts, even the data those contracts store.

This article explains Solana from the ground up: what accounts are, how programs work, and how transactions actually flow through the network.

## The Mental Model

Before diving in, here's the key insight:

**Ethereum**: Smart contracts = code + state bundled together
**Solana**: Programs = code only. State lives in separate accounts.

```
Ethereum:
┌─────────────────────────────────┐
│      Smart Contract             │
│  ┌───────────┬───────────────┐  │
│  │   Code    │     State     │  │
│  │ (logic)   │ (balances,    │  │
│  │           │  mappings)    │  │
│  └───────────┴───────────────┘  │
└─────────────────────────────────┘

Solana:
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│    Program    │    │ Data Account  │    │ Data Account  │
│   (Code Only) │    │   (State)     │    │   (State)     │
│               │    │               │    │               │
│  Token Program│    │ Alice's       │    │ Bob's         │
│               │    │ Token Balance │    │ Token Balance │
└───────────────┘    └───────────────┘    └───────────────┘
        │                   ▲                    ▲
        │                   │                    │
        └───────────────────┴────────────────────┘
                   Program "owns" these accounts
```

This separation enables Solana's parallel execution. Different transactions can modify different accounts simultaneously.

## Part 1: Accounts — Everything is a File

Think of Solana as a giant file system. Every piece of data lives in an **account** — a file with an address.

### Account Structure

Every account has these fields:

```
┌─────────────────────────────────────────────────────────────┐
│                         Account                             │
├─────────────────────────────────────────────────────────────┤
│  Address (Public Key)     32 bytes                          │
│  ─────────────────────────────────────────────────────────  │
│  lamports                 u64        (SOL balance)          │
│  data                     [u8]       (arbitrary bytes)      │
│  owner                    Pubkey     (program that controls)│
│  executable               bool       (is this code?)        │
│  rent_epoch               u64        (rent tracking)        │
└─────────────────────────────────────────────────────────────┘
```

Let's break down each field:

### Address (Public Key)

Every account has a unique 32-byte address. This is like a file path.

```
Example addresses:
Your wallet:     7EcDhSYGxXyscszYEp35KHN8sQWD8zAqKJBJRc2n2Kqq
Token Program:   TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA
USDC Mint:       EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v
```

### Lamports (Balance)

1 SOL = 1,000,000,000 lamports (10^9)

Every account holds some lamports. This isn't just for spending — it's **rent**.

### Data

Arbitrary bytes. The meaning depends on who owns the account:
- Wallet accounts: no data (empty)
- Token accounts: balance, mint address, owner
- Program accounts: compiled bytecode

### Owner

**This is crucial**: Every account has an **owner**, and the owner is always a program.

```
┌──────────────────┐     owns      ┌──────────────────┐
│  System Program  │ ────────────► │  Your Wallet     │
│                  │               │  (System Account)│
└──────────────────┘               └──────────────────┘

┌──────────────────┐     owns      ┌──────────────────┐
│  Token Program   │ ────────────► │  Token Account   │
│                  │               │  (Your USDC)     │
└──────────────────┘               └──────────────────┘
```

Only the owner program can:
- Modify the account's data
- Deduct lamports from the account

Anyone can:
- Credit lamports to any account
- Read any account's data

### Executable

If `true`, this account contains program code. The Solana runtime will execute it when called.

### Rent

To store data on-chain, you must deposit lamports proportional to the data size. This is called **rent exemption**.

```
Rent formula:
  minimum_balance = (data_size + 128) * lamports_per_byte_year * 2

Example:
  165-byte account (token account) ≈ 0.00203 SOL
  10 KB account ≈ 0.07 SOL
```

The "2 years" is a deposit — you get it back when you close the account.

```bash
# Check rent for a given size
solana rent 165
# Output: Rent-exempt minimum: 0.00203928 SOL
```

## Part 2: Account Types

### 1. System Accounts (Wallets)

Your wallet is just a System Account — owned by the **System Program**.

```
┌─────────────────────────────────────────────────────────────┐
│  System Account (Wallet)                                    │
├─────────────────────────────────────────────────────────────┤
│  address:     7EcDhSYGxXyscszYEp35KHN8sQWD8zAqKJBJRc2n2Kqq  │
│  lamports:    5_000_000_000 (5 SOL)                         │
│  data:        [] (empty)                                    │
│  owner:       11111111111111111111111111111111 (System)     │
│  executable:  false                                         │
└─────────────────────────────────────────────────────────────┘
```

The System Program handles:
- Creating new accounts
- Transferring SOL
- Assigning ownership to other programs

### 2. Program Accounts

Programs (smart contracts) are executable accounts.

```
┌─────────────────────────────────────────────────────────────┐
│  Program Account                                            │
├─────────────────────────────────────────────────────────────┤
│  address:     TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA   │
│  lamports:    1_141_440 (rent exempt)                       │
│  data:        [ELF bytecode...]                             │
│  owner:       BPFLoader2111111111111111111111111111111111   │
│  executable:  true  ← This makes it a program               │
└─────────────────────────────────────────────────────────────┘
```

Programs are owned by a **Loader** program (BPFLoader). The loader verifies and executes the bytecode.

### 3. Data Accounts

Programs store their state in separate data accounts.

```
┌─────────────────────────────────────────────────────────────┐
│  Token Account (holds your USDC balance)                    │
├─────────────────────────────────────────────────────────────┤
│  address:     9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM  │
│  lamports:    2_039_280 (rent exempt)                       │
│  data:        [mint, owner, amount, ...]  ← 165 bytes       │
│  owner:       TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA   │
│  executable:  false                                         │
└─────────────────────────────────────────────────────────────┘

Token Account Data Layout:
┌──────────┬────────┬──────────┬───────┬─────────────────────┐
│  mint    │ owner  │ amount   │ state │ ... (other fields)  │
│ 32 bytes │32 bytes│  8 bytes │1 byte │                     │
└──────────┴────────┴──────────┴───────┴─────────────────────┘
```

### 4. Mint Accounts

A Mint represents a token type (like USDC or a memecoin).

```
┌─────────────────────────────────────────────────────────────┐
│  Mint Account (USDC)                                        │
├─────────────────────────────────────────────────────────────┤
│  address:     EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v  │
│  data:        [supply, decimals, mint_authority, ...]       │
│  owner:       TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA   │
└─────────────────────────────────────────────────────────────┘

Mint Data:
  supply:          1,000,000,000 USDC total
  decimals:        6 (1 USDC = 1,000,000 units)
  mint_authority:  Who can create more
  freeze_authority: Who can freeze accounts
```

## Part 3: Programs — Stateless Smart Contracts

### How Programs Work

Programs are **stateless**. They don't store data internally. Instead, they read and write to accounts passed to them.

```
Transaction:
  "Transfer 100 USDC from Alice to Bob"

┌─────────────────────────────────────────────────────────────┐
│                      Token Program                          │
│                     (stateless code)                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  Alice's Token  │ │  Bob's Token    │ │  Alice's Wallet │
│  Account        │ │  Account        │ │  (signer)       │
│                 │ │                 │ │                 │
│  amount: 1000   │ │  amount: 500    │ │                 │
│  ──────────     │ │  ──────────     │ │                 │
│  amount: 900    │ │  amount: 600    │ │                 │
└─────────────────┘ └─────────────────┘ └─────────────────┘
        -100              +100              (signs tx)
```

The program:
1. Receives accounts as input
2. Validates permissions (Alice signed? Accounts owned by Token Program?)
3. Modifies account data (subtract from Alice, add to Bob)
4. Returns success or error

### Why Stateless?

This enables **parallel execution**. The Solana runtime (called Sealevel) can process multiple transactions simultaneously if they touch different accounts:

```
Time ──────────────────────────────────────────────────►

Thread 1: Alice → Bob (USDC)
Thread 2: Carol → Dave (SOL)
Thread 3: Eve → Frank (BONK)
                           │
                           │ All run in parallel!
                           │ Different accounts = no conflicts
```

If transactions touch the same account, they're serialized:

```
Thread 1: Alice → Bob ─────┐
Thread 2: Bob → Carol ─────┴── Must wait (same Bob account)
```

### Built-in Programs

Solana ships with several programs:

| Program | Address | Purpose |
|---------|---------|---------|
| System Program | `1111...1111` | Create accounts, transfer SOL |
| Token Program | `Tokenkeg...` | SPL tokens (USDC, etc.) |
| Associated Token | `ATokenGP...` | Derive token account addresses |
| BPF Loader | `BPFLoader...` | Deploy and run custom programs |

## Part 4: Instructions — The Unit of Work

An **Instruction** is a single operation: "call this program with these accounts and this data."

### Instruction Structure

```
┌─────────────────────────────────────────────────────────────┐
│                       Instruction                           │
├─────────────────────────────────────────────────────────────┤
│  program_id:    Which program to call                       │
│  accounts:      List of accounts to pass                    │
│  data:          Bytes telling the program what to do        │
└─────────────────────────────────────────────────────────────┘

Example: Transfer SOL
┌─────────────────────────────────────────────────────────────┐
│  program_id:    System Program (1111...1111)                │
│  accounts:      [                                           │
│                   { pubkey: Alice, is_signer: true,         │
│                     is_writable: true },                    │
│                   { pubkey: Bob, is_signer: false,          │
│                     is_writable: true }                     │
│                 ]                                           │
│  data:          [2, 0, 0, 0,  // instruction type (transfer)│
│                  64, 66, 15, 0, 0, 0, 0, 0]  // 1M lamports │
└─────────────────────────────────────────────────────────────┘
```

### Account Permissions

Each account in an instruction has two flags:

```
is_signer:   Does this account need to sign the transaction?
is_writable: Will this instruction modify this account?

┌──────────────────┬───────────┬─────────────┐
│     Account      │ is_signer │ is_writable │
├──────────────────┼───────────┼─────────────┤
│ Fee payer        │    Yes    │     Yes     │
│ Token sender     │    Yes    │     Yes     │
│ Token receiver   │    No     │     Yes     │
│ Token program    │    No     │     No      │
│ Mint (read only) │    No     │     No      │
└──────────────────┴───────────┴─────────────┘
```

**Why declare this upfront?**

Solana's scheduler uses these flags to parallelize transactions. Two transactions that both read the same account can run in parallel. Two that write to it must be serialized.

## Part 5: Transactions — Bundling Instructions

A **Transaction** bundles one or more instructions into an atomic unit.

### Transaction Structure

```
┌─────────────────────────────────────────────────────────────┐
│                       Transaction                           │
├─────────────────────────────────────────────────────────────┤
│  signatures:     [sig1, sig2, ...]    (64 bytes each)       │
│                                                             │
│  message:                                                   │
│    ├── header:                                              │
│    │     num_required_signatures: 2                         │
│    │     num_readonly_signed: 0                             │
│    │     num_readonly_unsigned: 3                           │
│    │                                                        │
│    ├── account_keys: [                                      │
│    │     Alice,        // index 0, signer, writable         │
│    │     Bob,          // index 1, signer, writable         │
│    │     Token Account A,  // index 2, writable             │
│    │     Token Account B,  // index 3, writable             │
│    │     Token Program,    // index 4, read-only            │
│    │     System Program,   // index 5, read-only            │
│    │   ]                                                    │
│    │                                                        │
│    ├── recent_blockhash: "5eykt4UsFv8P8NJdTR..."           │
│    │                                                        │
│    └── instructions: [                                      │
│          { program_id_index: 4,                             │
│            account_indices: [2, 3, 0],                      │
│            data: [3, ...] },   // Transfer tokens           │
│          { program_id_index: 5,                             │
│            account_indices: [0, 1],                         │
│            data: [2, ...] }    // Transfer SOL              │
│        ]                                                    │
└─────────────────────────────────────────────────────────────┘
```

### Key Points

**1. Size Limit: 1232 bytes**

Transactions must fit in a single UDP packet. This limits:
- Number of accounts (~35 unique accounts max)
- Number of instructions
- Data size

**2. Multiple Signers**

Unlike Ethereum (1 signer), Solana transactions can have multiple signers:

```
Transaction: "Alice and Bob both approve this trade"
  signatures: [Alice's signature, Bob's signature]
```

**3. Recent Blockhash**

Every transaction includes a recent blockhash as a timestamp. This:
- Prevents replay attacks (same tx can't be submitted twice)
- Expires transactions (valid for ~60 seconds / 150 blocks)

```
blockhash: "5eykt4UsFv8P8NJdTR..."  // From recent block
           ↓
After ~150 blocks, this transaction is invalid
```

**4. Atomicity**

All instructions succeed or all fail. No partial execution.

```
Instructions:
  1. Create token account  ✓
  2. Transfer tokens       ✓
  3. Close old account     ✗ (fails)

Result: ALL rolled back. Token account not created.
```

### Transaction Fees

**Base fee**: 5,000 lamports (0.000005 SOL) per signature

**Priority fee**: Optional tip to validators for faster inclusion

```
Total fee = (num_signatures × 5000) + priority_fee

Example (1 signer, no priority):
  Fee = 1 × 5000 = 5000 lamports = 0.000005 SOL
```

Half the fee is burned. Half goes to the validator.

## Part 6: Program Derived Addresses (PDAs)

How do programs "own" accounts if programs can't sign transactions?

**PDAs** — addresses derived from seeds, with no private key.

### The Problem

Programs need to control accounts (hold tokens, store state). But:
- Programs can't have private keys
- Without a private key, you can't sign transactions
- Without signing, you can't authorize changes

### The Solution: PDAs

A PDA is an address mathematically derived from:
1. Seeds (arbitrary bytes)
2. Program ID
3. A "bump" value

```
PDA = hash(seeds, program_id, bump)

Example:
  seeds = ["vault", user_pubkey]
  program_id = MyProgram
  bump = 255 (or lower until we get a valid PDA)

  PDA = hash(["vault", Alice], MyProgram, 255)
      = 8xK2v9bLqNmV5...  (deterministic address)
```

### Why the Bump?

Normal addresses are on the Ed25519 elliptic curve (they have private keys). PDAs must be **off** the curve (no private key possible).

```
hash(seeds, program_id, 255) → On curve? → Has private key → BAD!
hash(seeds, program_id, 254) → On curve? → Has private key → BAD!
hash(seeds, program_id, 253) → Off curve? → No private key → GOOD! ✓

The first bump that produces an off-curve address is the "canonical bump"
```

### How Programs Sign with PDAs

Programs don't sign. Instead, the runtime grants them authority:

```rust
// In program code:
invoke_signed(
    &transfer_instruction,
    &[vault_account, destination],
    &[&[b"vault", user.key.as_ref(), &[bump]]]  // Seeds prove ownership
)?;
```

The runtime verifies:
1. The PDA can be derived from these seeds + this program
2. If yes, the program can modify this account

```
┌──────────────────────────────────────────────────────────────┐
│                        Runtime Check                         │
│                                                              │
│  Program: MyProgram (address: Abc123...)                     │
│  Trying to modify: 8xK2v9bLqNmV5... (PDA)                   │
│                                                              │
│  Verify: hash(["vault", Alice], Abc123..., 253) = 8xK2v9b?  │
│           ↓                                                  │
│          YES → Program can modify this account               │
│          NO  → Access denied                                 │
└──────────────────────────────────────────────────────────────┘
```

### PDA Example: Token Vault

```
User wants to stake tokens. Program needs to hold them.

┌─────────────────┐
│    User         │
│    Alice        │
└────────┬────────┘
         │
         │ 1. Deposit 100 tokens
         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Staking Program                          │
├─────────────────────────────────────────────────────────────┤
│  Derive vault PDA:                                          │
│    seeds = ["vault", Alice's pubkey]                        │
│    PDA = 8xK2v9bLqNmV5...                                  │
│                                                              │
│  Transfer tokens from Alice → PDA                           │
│                                                              │
│  Later, when unstaking:                                     │
│    invoke_signed() to transfer from PDA → Alice             │
└─────────────────────────────────────────────────────────────┘
         │
         │ Tokens held in PDA
         ▼
┌─────────────────┐
│  Vault (PDA)    │
│  8xK2v9bLqNmV5  │
│                 │
│  100 tokens     │
│  owner: Token   │
│  Program        │
└─────────────────┘
```

## Part 7: Token Accounts — How USDC Works

### The Token Program

All SPL tokens (USDC, BONK, etc.) use the same **Token Program**. It's like ERC-20, but as a shared program.

### Three Account Types

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Mint Account  │     │  Token Account  │     │  Token Account  │
│   (USDC)        │     │  (Alice's USDC) │     │  (Bob's USDC)   │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ supply: 1B      │     │ mint: USDC      │     │ mint: USDC      │
│ decimals: 6     │     │ owner: Alice    │     │ owner: Bob      │
│ mint_auth: ...  │     │ amount: 1000    │     │ amount: 500     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        └───────────────────────┴───────────────────────┘
                    All owned by Token Program
```

**Mint Account**: Defines the token (supply, decimals, who can mint)

**Token Account**: Holds balance for one user, for one token type

### Associated Token Accounts (ATAs)

Problem: Token account addresses are random. How does Bob know Alice's USDC address?

Solution: **Deterministic addresses** derived from (owner, mint):

```
ATA address = derive(Alice's wallet, USDC mint, ATA program)
            = deterministic!

Anyone can compute Alice's USDC address without asking Alice.
```

```
┌─────────────────────────────────────────────────────────────┐
│  Associated Token Account Derivation                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Inputs:                                                    │
│    wallet_address:  Alice (7EcDh...)                        │
│    mint_address:    USDC (EPjFW...)                         │
│    ata_program:     ATokenGPvbdGVxr...                      │
│                                                             │
│  PDA = find_program_address(                                │
│          [Alice, Token Program, USDC],                      │
│          ATA Program                                        │
│        )                                                    │
│                                                             │
│  Output:                                                    │
│    Alice's USDC ATA: 9WzDX... (deterministic!)              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Token Transfer Flow

```
Alice sends 100 USDC to Bob:

┌──────────────────────────────────────────────────────────────┐
│  Transaction                                                 │
├──────────────────────────────────────────────────────────────┤
│  Signatures: [Alice's signature]                             │
│                                                              │
│  Instruction:                                                │
│    program: Token Program                                    │
│    accounts:                                                 │
│      - Alice's USDC ATA (writable)                          │
│      - Bob's USDC ATA (writable)                            │
│      - Alice's wallet (signer)                              │
│    data: Transfer { amount: 100_000_000 }  // 100 USDC      │
│                                            // (6 decimals)   │
└──────────────────────────────────────────────────────────────┘

Token Program executes:
  1. Check: Alice signed?                    ✓
  2. Check: Alice's ATA owned by Token Prog? ✓
  3. Check: Alice's ATA owner == Alice?      ✓
  4. Check: Alice's balance >= 100?          ✓
  5. Alice's ATA: amount -= 100_000_000
  6. Bob's ATA: amount += 100_000_000
```

## Part 8: Transaction Lifecycle

What happens when you click "Send" in your wallet?

### Step 1: Build Transaction

```
Wallet builds transaction:
  - Fetches recent blockhash from RPC
  - Constructs instructions
  - Estimates fees
  - Adds accounts
```

### Step 2: Sign

```
User signs with private key:
  - Signs the message (not the whole tx)
  - Produces 64-byte signature
```

### Step 3: Submit to RPC

```
Wallet → RPC Server → Leader Validator

The RPC:
  1. Basic validation
  2. Forwards to current leader
  3. Also forwards to next 2 leaders (redundancy)
```

### Step 4: Leader Processing

```
┌─────────────────────────────────────────────────────────────┐
│                    Leader Validator                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Fetch Stage:                                               │
│    Receive transactions, batch into groups of 128           │
│                                                             │
│  SigVerify Stage:                                           │
│    Verify all signatures are valid                          │
│    Remove duplicates                                        │
│                                                             │
│  Banking Stage:                                             │
│    Execute transactions (parallel when possible)            │
│    Update account states                                    │
│    Check account balances, permissions                      │
│                                                             │
│  Broadcast Stage:                                           │
│    Share results with other validators                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Step 5: Confirmation

Solana has three commitment levels:

```
┌─────────────────────────────────────────────────────────────┐
│                   Commitment Levels                         │
├─────────────┬───────────────────────────────────────────────┤
│  processed  │ Included in a block by the leader            │
│             │ Fast (~400ms), but could be reverted          │
├─────────────┼───────────────────────────────────────────────┤
│  confirmed  │ 2/3 of validators voted for this block       │
│             │ Very unlikely to revert (~5 seconds)          │
├─────────────┼───────────────────────────────────────────────┤
│  finalized  │ 31+ blocks built on top                       │
│             │ Never reverted in Solana's history (~12 sec)  │
└─────────────┴───────────────────────────────────────────────┘
```

### Timeline

```
Time: 0ms      400ms         5s              12s
      │         │            │                │
      ▼         ▼            ▼                ▼
   Submit → Processed → Confirmed → Finalized
      │         │            │                │
      │         │            │                └─ Safe to consider permanent
      │         │            └─ Safe for most use cases
      │         └─ Visible on explorer, might revert
      └─ Transaction sent to leader
```

### What If It Fails?

**Blockhash expired**: Resubmit with fresh blockhash
**Insufficient funds**: Top up, resubmit
**Account locked**: Another tx using same account; retry
**Program error**: Fix the bug in your program

## Part 9: Practical Example

Let's trace a complete USDC transfer:

```
Alice (wallet): 7EcDhSYG...
Bob (wallet):   9BzKqR4W...
USDC Mint:      EPjFWdd5...

Alice's USDC ATA: derive(Alice, USDC) = 3xLp8mKw...
Bob's USDC ATA:   derive(Bob, USDC)   = 7yNq2vRt...
```

### The Transaction

```javascript
// Using @solana/web3.js
const transaction = new Transaction().add(
  createTransferInstruction(
    aliceUsdcAta,        // source (Alice's USDC account)
    bobUsdcAta,          // destination (Bob's USDC account)
    aliceWallet,         // owner (Alice, must sign)
    100_000_000          // amount (100 USDC, 6 decimals)
  )
);

transaction.recentBlockhash = (await connection.getLatestBlockhash()).blockhash;
transaction.feePayer = aliceWallet;

// Sign
const signed = await wallet.signTransaction(transaction);

// Send
const signature = await connection.sendRawTransaction(signed.serialize());

// Wait for confirmation
await connection.confirmTransaction(signature, 'confirmed');
```

### What Gets Sent

```
┌─────────────────────────────────────────────────────────────┐
│  Raw Transaction (serialized)                               │
├─────────────────────────────────────────────────────────────┤
│  01                        // 1 signature                   │
│  [64 bytes: Alice's sig]                                    │
│                                                             │
│  01                        // 1 signer required             │
│  00                        // 0 read-only signers           │
│  02                        // 2 read-only non-signers       │
│                                                             │
│  05                        // 5 accounts                    │
│  [32 bytes: Alice wallet]  // index 0, signer, writable     │
│  [32 bytes: Alice ATA]     // index 1, writable             │
│  [32 bytes: Bob ATA]       // index 2, writable             │
│  [32 bytes: Token Program] // index 3, read-only            │
│  [32 bytes: USDC Mint]     // index 4, read-only            │
│                                                             │
│  [32 bytes: recent blockhash]                               │
│                                                             │
│  01                        // 1 instruction                 │
│  03                        // program index (Token Program) │
│  04                        // 4 accounts in instruction     │
│  01 02 00 04               // account indices               │
│  09                        // data length                   │
│  03                        // instruction type (Transfer)   │
│  [8 bytes: 100_000_000]    // amount                        │
└─────────────────────────────────────────────────────────────┘
```

## Summary Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SOLANA ARCHITECTURE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ACCOUNTS (Data Storage)                                                   │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│   │   Wallet    │  │   Program   │  │    Data     │  │    Mint     │       │
│   │  (System)   │  │ (Executable)│  │  (State)    │  │  (Token)    │       │
│   │             │  │             │  │             │  │             │       │
│   │ holds SOL   │  │ holds code  │  │ holds state │  │ token config│       │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘       │
│                                                                             │
│   PROGRAMS (Code Execution)                                                 │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  Stateless │ Read/write accounts │ Can invoke other programs        │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   TRANSACTIONS (Atomic Operations)                                          │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  Signatures │ Recent Blockhash │ Instructions │ Account List        │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   INSTRUCTIONS (Single Operations)                                          │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  Program ID │ Account Indices │ Data (what to do)                   │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   LIFECYCLE                                                                 │
│   Build → Sign → Submit → Process → Confirm → Finalize                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Takeaways

1. **Everything is an account** — wallets, programs, tokens, state
2. **Programs are stateless** — they read/write to accounts passed to them
3. **Accounts have owners** — only the owner program can modify data
4. **PDAs enable program-owned accounts** — addresses without private keys
5. **Transactions are atomic** — all instructions succeed or all fail
6. **Parallelism from explicit accounts** — declare what you'll read/write

## References

- [Solana Docs: Transactions](https://solana.com/docs/core/transactions)
- [Solana Docs: Accounts](https://solana.com/docs/core/accounts)
- [Solana Docs: Programs](https://solana.com/docs/core/programs)
- [Lifecycle of a Solana Transaction - Umbra Research](https://www.umbraresearch.xyz/writings/lifecycle-of-a-solana-transaction)
- [Transaction Anatomy - Solana Handbook](https://ackee.xyz/solana/book/latest/chapter3/transaction-anatomy/)
- [Program Derived Addresses - Solana Docs](https://solana.com/docs/core/pda)
- [SPL Token Program](https://spl.solana.com/token)
- [Solana Account Model - QuickNode](https://www.quicknode.com/guides/solana-development/getting-started/an-introduction-to-the-solana-account-model)
