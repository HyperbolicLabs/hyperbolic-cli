# Hyperbolic CLI

A command-line interface for managing GPU instances on [Hyperbolic](https://app.hyperbolic.ai/). The Hyperbolic CLI allows you to rent remote GPU instances, manage your rentals, and monitor your usage - all from the comfort of your terminal.

## Features

- **Spot Instances**: Rent GPU instances at discounted rates
- **On-Demand Instances**: Rent VM and bare-metal GPU instances with guaranteed availability
- **Instance Management**: View, monitor, and terminate your active instances
- **Account Management**: Check your balance and manage authentication
- **SSH Integration**: Quick access to SSH connection details
- **JSON Output**: Machine-readable output for automation

## Prerequisites

- Go 1.24.3 or later
- A Hyperbolic account at [app.hyperbolic.ai](https://app.hyperbolic.ai/)
- SSH public key uploaded to your Hyperbolic account settings

## Installation

### From Source

```bash
git clone https://github.com/HyperbolicLabs/hyperbolic-cli.git
cd hyperbolic-cli
go build -o hyperbolic
```

### Using Go Install

```bash
go install github.com/kaihuang724/hyperbolic-cli@latest
```

### Getting a Hyperbolic Account and API Token

![Hyperbolic Website](./images/hyperbolicwebsite.png)

1. Register for a Hyperbolic account:
   - Visit [https://app.hyperbolic.xyz/](https://app.hyperbolic.xyz/)
   - Create an account or log in to your existing account
   - Verify your email address

2. Deposit funds into your account:
   - Log in to your Hyperbolic application
   - Navigate to the "Billing" tab
   - Select how much you want to deposit (we suggest starting with $25)
   - Click Pay Now
   - Follow the instructions to add funds to your account
   - Note that you will need sufficient funds to rent GPU instances

3. Generate an API token:
   - In your Hyperbolic dashboard, navigate to "Settings" 
   - Navigate to the API Key section
   - Copy the generated token and keep it secure

4. Add your SSH public key:
   - Generate an SSH key pair if you don't already have one
   - In [https://app.hyperbolic.xyz/](https://app.hyperbolic.xyz/), navigate to the "Settings" section
   - Scroll down to the SSH Public Key section
   - Paste your public key (usually from ~/.ssh/id_rsa.pub or similar)
   - Click the save icon


## Usage

### Authentication

```bash
# Set your API key
hyperbolic auth YOUR_API_KEY

# View current configuration
hyperbolic config
```

### Managing Instances

```bash
# View all your instances
hyperbolic instances

# View detailed information about a specific instance
hyperbolic instances INSTANCE_ID

# Get JSON output for automation
hyperbolic instances --json
```

### Spot Instances

```bash
# Rent a spot instance
hyperbolic spot

# View available spot instances
hyperbolic rent
```

### On-Demand Instances

```bash
# Rent an on-demand instance
hyperbolic ondemand
```

### Instance Termination

```bash
# Terminate an instance
hyperbolic terminate INSTANCE_ID
```

### Account Management

```bash
# Check your account balance
hyperbolic balance
```

## Commands

To see all available commands and their descriptions, run:

```bash
hyperbolic --help
```

You can also get help for any specific command:

```bash
hyperbolic [command] --help
```

## Configuration

The CLI stores your API key and configuration in your home directory. The configuration file is typically located at `~/.hyperbolic-cli.yaml`.

## Output Formats

Most commands support both human-readable table output and JSON output for automation:

```bash
# Human-readable output (default)
hyperbolic instances

# JSON output for scripts
hyperbolic instances --json
```

## Examples

### Rent a Spot Instance

```bash
# Browse available spot instances
hyperbolic rent

# Rent a specific spot instance
hyperbolic spot
```

### Monitor Your Instances

```bash
# View all instances with status and pricing
hyperbolic instances

# Get detailed information about a specific instance
hyperbolic instances abc123def456

# Monitor instances in a script
hyperbolic instances --json | jq '.spot_instances.instances[] | select(.instance.status == "running")'
```

### Terminate an Instance

```bash
# Terminate a specific instance
hyperbolic terminate abc123def456
```

## Error Handling

If you encounter authentication errors, make sure:
1. Your API key is correctly set: `hyperbolic auth YOUR_API_KEY`
2. Your API key is valid and not expired
3. Your account is in good standing

For instance-related errors, verify:
1. The instance ID is correct
2. You have uploaded your public key to 
3. The instance is still active

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [Hyperbolic Documentation](https://docs.hyperbolic.ai/)
- **Dashboard**: [app.hyperbolic.ai](https://app.hyperbolic.ai/)

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Tablewriter](https://github.com/olekukonko/tablewriter) - Table formatting