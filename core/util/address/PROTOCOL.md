# Protocols

## Address derivation

Public keys are created using `ed25519` elliptic curve algorithm. An address is a hash of the public key to obtain a smaller length representation of it. To derive the address, we:

1- Take the SHA3-256 hash of the public key (end up with 32 bytes string).

2- Drop the first 14 bytes of the hash

3- Convert the remaining 18 bytes to hex and prepend `0x`. (38 characters in total)

