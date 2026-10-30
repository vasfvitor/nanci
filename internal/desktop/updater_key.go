package main

// updaterPublicKey is the Ed25519 public key every update must be signed
// with, as standard base64 of the raw 32-byte key: the line that
// `wails3 updater genkey` prints under "Or inline as base64 of the raw key".
// While it is empty, automatic updates are off, and the release workflow
// refuses to publish a tag. docs/WAILS_V3.md, section "Atualização
// automática", has the steps to create the key pair.
//
// It is a var, not a const, so a local test build can pin a throwaway key
// with -ldflags "-X main.updaterPublicKey=...".
var updaterPublicKey = "BEVh/oI0xlNQRudPKXw7Y4TA/5DcM96SoMuYoov0Kqo="
