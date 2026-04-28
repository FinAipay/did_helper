# FinAI DID Helper



### Overview

FinAI DID Helper is a powerful CLI tool for managing Decentralized Identifiers (DIDs), cryptographic keys, and authentication tickets on the FinAI network. It provides seamless integration with blockchain wallets (Ethereum, Solana) and supports W3C DID Core specification.

---

### Features

- **DID Management**: Create, query, update, and deactivate DID documents
- **Key Generation**: Generate Ethereum, Solana, and X25519 key pairs
- **Wallet Integration**: Support MetaMask (Ethereum) and Phantom (Solana) browser extensions
- **Ticket System**: Challenge-response authentication with automatic ticket management
- **API Key Management**: Create, list, and revoke API keys for service access
- **Secure Storage**: Local encrypted storage with DID-based directory structure
- **Hashlink Verification**: Document integrity verification using content hashing

### Installation

#### Prerequisites

- Go 1.21 or higher
- Git

#### Build from Source

```bash
# Clone repository
git clone https://github.com/finai-network/did-helper.git
cd did-helper

# Build binary
make build

# Or manually
go build -o did_helper .
```

#### Install Globally

```bash
sudo cp did_helper /usr/local/bin/
```
Or 

```bash
make install
# Or
go install .
```

### Quick Start

#### 1. Generate a Key Pair

```bash
# Generate Ethereum key
./did_helper key generate --type ethereum --password "your_secure_password"

# Generate Solana key
./did_helper key generate --type solana

# Generate X25519 key (for encryption)
./did_helper key generate --type x25519
```

#### 2. Create DID Document

```bash
# Create DID for an agent
./did_helper did create --entity-type agents --entity-id 0xYourAddress

# Create DID for a user
./did_helper did create --entity-type users --entity-id 0xYourAddress
```

#### 3. Authenticate & Get Ticket

```bash
# Generate challenge and sign (automatic if key exists locally)
./did_helper ticket challenge --did did:finai:agents:0xYourAddress

# If using browser wallet, open the generated HTML file on your desktop

# Verify signature and obtain ticket
./did_helper ticket verify --did did:finai:agents:0xYourAddress \
  --challenge "challenge_string" \
  --signature "0xsignature"
```

#### 4. Manage API Keys

```bash
# Create API key
./did_helper apikey create --did did:finai:agents:0xYourAddress \
  --service-name my-service

# List API keys
./did_helper apikey list --did did:finai:agents:0xYourAddress

# Revoke API key
./did_helper apikey revoke --did did:finai:agents:0xYourAddress \
  --api-key pk_live_xxx
```

### Command Reference

#### Key Management

```bash
# Generate key
did_helper key generate --type <ethereum|solana|x25519> [--password <pwd>]

# Show key info
did_helper key show --address <address>

# List all keys
did_helper key list
```

#### DID Operations

```bash
# Create DID
did_helper did create --entity-type <users|agents|devices|services|orgs|assets> \
  --entity-id <id>

# Query DID
did_helper did query --did <did>

# List DIDs
did_helper did list

# Show DID details
did_helper did show --did <did>

# Verify document integrity
did_helper did verify --did <did> [--hl <hashlink>]

# Update DID
did_helper did update --did <did> [--add-key <json>] [--revoke-key <id>]

# Deactivate DID
did_helper did deactivate --did <did> [--force]

# Check reputation
did_helper did reputation --did <did>
```

#### Ticket Authentication

```bash
# Generate and sign challenge
did_helper ticket challenge --did <did>

# Verify and get ticket
did_helper ticket verify --did <did> --challenge <challenge> --signature <sig>

# Show ticket info
did_helper ticket show --did <did>
```

#### API Key Management

```bash
# Create API key
did_helper apikey create --did <did> [--service-name <name>]

# List API keys
did_helper apikey list --did <did>

# Revoke API key
did_helper apikey revoke --did <did> --api-key <key>
```

#### Utility Commands

```bash
# Set default DID
did_helper use <did>

# Send HTTP requests
did_helper api get/post/put/delete --url <url> [--body <json>] [--header <key:value>]
```

### Browser Wallet Signing

When private keys are not stored locally, the tool generates an HTML file on your desktop:

1. Open the HTML file in your browser
2. The page automatically detects MetaMask/Phantom extension
3. Connect your wallet and sign the challenge
4. Copy the signature and use it with the `ticket verify` command

### Storage Structure

All data is stored in `~/.did_helper/`:

```
~/.did_helper/
├── config.json              # Global configuration
├── import/                  # Imported keys/wallets
│   ├── 0xabc123.../        # ETH wallet directory
│   │   ├── keystore.json
│   │   ├── wallet.json
│   │   └── password.txt
│   └── xyz789.../          # Solana/X25519 key directory
│       ├── keypair.json
│       └── metadata.json
└── did-finai-agents-0x.../ # DID-specific directory
    ├── did-finai-agents-0x....json  # DID document
    ├── ticket.json          # Authentication ticket
    └── apikey.json          # API keys
```

### Security Notes

⚠️ **Important Security Practices:**

- Always use strong passwords (8+ characters, letters + digits)
- Backup your keystore files and passwords securely
- Never share private keys or passwords
- The `password.txt` file is for local convenience only - never commit to version control
- Browser wallet signing is recommended for production use

### Compatible Versions

- **Go**: 1.21+
- **Operating Systems**: Linux, macOS, Windows
- **Browser Extensions**: 
  - MetaMask (Ethereum)
  - Phantom (Solana)

### License

This project is licensed under the MIT License 
