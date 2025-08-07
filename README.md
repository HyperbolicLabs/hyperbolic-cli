# Hyperbolic CLI

Developer-first CLI to browse, rent, and manage Hyperbolic GPUs. Rent on-demand GPU VMs and bare‑metal clusters from $1.49 per hour.

## Install

```bash
brew install HyperbolicLabs/hyperbolic/hyperbolic
```

## Quick start

```bash
hyperbolic auth login
hyperbolic --help
```

## Account setup

1. Create an account: `https://app.hyperbolic.ai/` and verify your email.
2. Add funds: go to Billing and deposit funds (we suggest starting with $25).
3. Add SSH public key: Settings → SSH Public Key → paste your `~/.ssh/id_rsa.pub` (or `ed25519.pub`) and save.


## Links

- Docs: https://docs.hyperbolic.xyz
- Dashboard: https://app.hyperbolic.ai/
- License: MIT

## Examples

```bash
# Browse on-demand marketplace
hyperbolic ondemand

# Rent an on-demand VM (1, 2, 4, 8 GPUs depending on availability)
hyperbolic rent ondemand --instance-type virtual-machine --gpu-count 1

#Rent and on-demand Bare Metal Cluster (Multiples of 8 GPUs depending on availability)
hyperbolic rent ondemand --instance-type bare-metal --network-type infiniband --gpu-count 16

# List and then terminate an instance
hyperbolic instances
hyperbolic terminate <instance-id>
```