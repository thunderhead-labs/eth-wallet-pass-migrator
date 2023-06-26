# eth-wallet-pass-migrator

A simple tool to regenerate a wallet (only non-deterministic supported right now) with a new password. Particularly useful if you have empty password keys

## Installation

`go install github.com/thunderhead-labs/eth-wallet-pass-migrator@latest`

## Usage

`eth-wallet-pass-migrator --new-passphrase newpass --wallet-name oldwallet --new-wallet-name newwallet --store-location ./ndwallet`
