# Vault

![banner](https://github.com/FleekHQ/space-daemon/raw/vault/docs/crypto/mario-crypto.jpg)

A vault is used to securely store user private keys based on a master password. It is hosted on the cloud and uses the cryptography described here to assure it can't access the data directly. It works very similarly to password managers.

## Vault data model

### Map uuids to public keys (uses address book service for this)

| uuid | public key |
|------|------------|
| 1    | 0xa        |
| 2    | 0xb        |
| 3    | 0xc        |

### Maps uuids to vaults

| uuid | vault                                                                            | vskHash  |
|------|----------------------------------------------------------------------------------|----------|
| 1    | Encrypted({   a: somePrivateKey,   b: otherPrivateKey,   c: anotherPrivateKey }) | 0xabc... |

## Vault Flows

### Storing private keys

The client needs to complete a challenge to prove they have access to a given public key. Once they have proven access, the server allows replacing the vault file for a new one.

#### Private key signing challenge flow:

1. Client sends to the server their public key
2. Server issues a challenge
3. Client signs the challenge using its private key
4. Server verifies signature matches the public key, returning a JSON Web Token (JWT)

#### Storing the private key

1. Client creates the vault file (`vf`), which is a JSON document that maps public keys to their private keys, but can also contain anything we want to store.
2. Client computes its vault key (`vk`). To do this, it runs `PBKDF2(password, salt, iterations, hashingFn)`, where `password` is the master password, `salt` is the user's `uuid`, `iterations` is a high number to prevent brute force (set to 100.000 as of now), and `hashingFn` is SHA512 which is the industry standard for a secure hashing function.
3. Using `vk`, client encrypts `vf` using AES, obtaining `vk(vf)`.
4. Client computes the vault service key (`vsk`) by doing key derivation again: `PBKDF2(vk, password, iterations, hashingFn)`, where `password` is the master password.
5. Client submits `vk(vf)`, `vsk` and the JWT back to the server.
6. Server verifies the JWT and successfully stores `vk(vf)` for the user with the given uuid.
7. Server stores `vskHash = PBKDF2(vsk, iterations, hashingFn)` using a really high value for `iterations`.

#### Retrieving the private key

1. Client computes `vk` and `vsk` again as in step (2) and (4) of the previous section.
2. Client sends a retrieve request to the server with `vsk` and `uuid` as the params.
3. Server computes `vskHash` as in (7) of the previous section.
4. Server checks `vskHash` matches the one stored. If it does, it returns `vk(vf)`. If not, returns a "Wrong password" error.
5. Client decrypts `vk(vf)` using `vk`, obtaining `vf` back and getting access to its private keys.

## Takeaways

- The client only needs to remember the master password and the uuid (which is obtained through a username, so it needs to remember the username).
- The server only receives `vsk` and therefore cannot decrypt `vk(vf)` from it alone. It can bruteforce `vsk` to obtain `vk`, but given `vk` is a SHA512 hash already, it'd take a billion years.
- If a middleman intercepts the client->server message, and somehow gets to decrypt the first layer of protection which is TLS, it can't decrypt `vk(vf)` without `vk`.
- The server should implement rate-limitting to protect weak master passwords from being cracked.
