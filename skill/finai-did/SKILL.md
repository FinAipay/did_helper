---
name: did-helper
description: CLI tool for FinAI DID, keys, tickets, API keys. Use for decentralized identity operations.
license: MIT
compatibility: |
  Requires 'did_helper' CLI in PATH.
  Install: 
    Release：
        #download newest version
        https://github.com/FinAipay/did_helper/releases
    Manual:
        git clone https://github.com/finai-network/did_helper.git
        cd did_helper && make build && sudo cp did_helper /usr/local/bin/
  Verify: did_helper help
  OS: Linux, macOS, Windows (PowerShell/CMD)
---

# FinAI DID Helper

## HARD RULES (MUST FOLLOW)

### RULE 1: did_helper ONLY
- ✅ ALLOWED: `did_helper key generate --type ethereum --password "xxx"`
- ❌ FORBIDDEN: Any command NOT starting with `did_helper` (ls, cat, find, grep, which, stat, etc.)
- ❌ FORBIDDEN: Any filesystem probing

### RULE 2: NO PATH GUESSING
- ❌ NEVER guess/assume file paths, directories, key locations
- ❌ NEVER run: `ls`, `find`, `which`, `cat`, `read`, `stat`, `locate`
- Let `did_helper` manage its own storage
- If "key not found" → tell user to generate key first

### RULE 3: NO PARAMETER GUESSING
- Use EXACT parameter names from this doc
- **Required** params MUST come from user
- NEVER add undocumented params
- NEVER assume defaults (require explicit user input)

### RULE 4: FORBIDDEN — API Key Commands

**Agent MUST NOT execute:**

| Command                    | Reason                                         |
| -------------------------- | ---------------------------------------------- |
| `did_helper apikey create` | Creates live API keys — user must run manually |
| `did_helper apikey list`   | Exposes sensitive keys                         |
| `did_helper apikey revoke` | Irreversible                                   |

**If user asks for API keys → Reply:**
```
API key commands are FORBIDDEN. Please run manually:
did_helper apikey <command>
```
**DO NOT execute on behalf of user.**

### RULE 5: TICKET AUTH — Dynamic Error Detection

**CORE PRINCIPLE: No need to know which commands require tickets. Check output after execution.**

After ANY `did_helper` command, MUST check output:

```
IF output CONTAINS "missing authorization header" OR similar auth error:
    → Command requires ticket (no valid ticket exists)
    → Auto-trigger Workflow C (Ticket Authentication)
    → After success, ask user: "Retry original command?"
```

**Auth error patterns to detect (case-insensitive):**
- `missing authorization header`
- `unauthorized`
- `ticket not found`
- `authentication required`
- `invalid ticket`
- `ticket expired`
- `wallet not found`                    # x402: ETH wallet missing
- `eth wallet required`                 # x402: Only ETH supported
- `solana not supported`                # x402: Solana wallets rejected
- `insufficient balance`                # x402: Not enough funds
- `invalid signature`                   # x402: EIP-712 signature failed
- `eip-712 signing failed`             # x402: Signing process error
- `chain not supported`                 # x402: Network not configured
- `payment timeout`                     # x402: Transaction timeout

**Agent behavior:**
1. Execute user command
2. Capture stdout + stderr
3. If auth error detected → DO NOT report as failure → Auto-run Workflow C
4. After ticket success → Ask user: "Ticket acquired. Retry previous command? (yes/no)"
5. If yes → Re-execute original command

---

## WORKFLOWS

### Step 0: Pre-Execution Validation (MANDATORY)

```python
def validate_command(cmd_parts):
    # Check 1: Must start with did_helper
    if cmd_parts[0] not in ["did_helper", "did_helper.exe"]:
        raise ValidationError(f"Forbidden: {cmd_parts[0]} - only did_helper allowed")
    
    # Check 2: No path separators or dangerous chars
    dangerous = ["/", "\\", "..", "~", "$", "`", ";", "|", "&", ">", "<"]
    for part in cmd_parts:
        for d in dangerous:
            if d in part:
                raise ValidationError(f"Forbidden char '{d}' in: {part}")
    
    # Check 3: No apikey commands
    if len(cmd_parts) >= 2 and cmd_parts[1] == "apikey":
        raise ValidationError("apikey commands forbidden. Tell user to run manually.")
    
    return True
```

**If validation fails** → Output error + Terminate. NEVER "fix" command.

---

### Step 1: Classify User Intent

| Category           | Keywords                                                   | Action                                  |
| ------------------ | ---------------------------------------------------------- | --------------------------------------- |
| **Key generate**   | "generate key", "create key", "new ethereum/solana/x25519" | Workflow A                              |
| **DID create**     | "create DID", "new DID", "register identity"               | Workflow B                              |
| **Ticket auth**    | "get ticket", "authenticate", "login", "challenge"         | Workflow C                              |
| **DID query**      | "query DID", "lookup DID", "get DID info"                  | Workflow D                              |
| **DID update**     | "update DID", "add key", "add service", "modify DID"       | Workflow E (auto-detect ticket)         |
| **Reputation**     | "reputation", "score", "trust score"                       | Workflow F (auto-detect ticket)         |
| **DID deactivate** | "deactivate DID", "disable DID", "delete DID"              | Workflow G (auto-detect + irreversible) |
| **x402支付**       | "payment", "pay", "x402", "order", "create payment"        | Workflow H (auto-detect + wallet check) |
| **apikey** (any)   | "API key", "create API key", "list API keys", "revoke"     | **FORBIDDEN** → manual instruction      |
| **Other**          | Not matching above                                         | Ask user to clarify                     |

**If confidence <80%** → Ask: "Please specify operation. Example: 'generate Ethereum key' or 'create DID for 0x...'"

---

### Workflow A: Key Generation (No Ticket)

**Step A1: Extract required params**

| Param               | Required       | Source                              | Guessing forbidden |
| ------------------- | -------------- | ----------------------------------- | ------------------ |
| `--type`            | ✅ YES          | User: ethereum, solana, x25519      | ❌ NO default       |
| `--password`        | ✅ for ethereum | User provide or `--prompt-password` | ❌ NO generation    |
| `--prompt-password` | Optional       | Use for interactive                 | -                  |

**Step A2: If missing → Ask user**

```
Missing required: --type
Specify: ethereum, solana, or x25519

Example: did_helper key generate --type ethereum --password "your_password"
```

**Step A3: Execute EXACT format**

```bash
# Ethereum
did_helper key generate --type ethereum --password "<password>"
did_helper key generate --type ethereum --prompt-password

# Solana
did_helper key generate --type solana

# X25519
did_helper key generate --type x25519
```

**Step A4: Show output** (public key/address only)

**Step A5: Error handling**

| Error                | Action                        |
| -------------------- | ----------------------------- |
| "key already exists" | Inform user, ask if overwrite |
| Invalid type         | Show allowed types, retry     |
| Other                | Show raw error, retry         |

**Max retries: 2** → After 2 failures, terminate, suggest manual.

---

### Workflow B: DID Creation (No Ticket)

**Step B1: Extract required params**

| Param           | Required | Source                                         | Guessing forbidden |
| --------------- | -------- | ---------------------------------------------- | ------------------ |
| `--entity-type` | ✅ YES    | users, agents, devices, services, orgs, assets | ❌ NO guessing      |
| `--entity-id`   | ✅ YES    | User provide address/pubkey                    | ❌ NO assumption    |

**Step B2: Validate entity-id format**
- Ethereum: `0x` + 40 hex chars
- Solana: Base58, 32-44 chars

**Step B3: Execute**

```bash
did_helper did create --entity-type <type> --entity-id <address>
```

**Step B4: Show output**

```
✅ DID created: did:finai:users:0x936...
```

**Step B5: Error handling**

| Error                 | Action                           |
| --------------------- | -------------------------------- |
| "key not found"       | Run Workflow A first, then retry |
| "invalid entity type" | Show allowed types               |
| Other                 | Show error, terminate            |

**Max retries: 2**

---

### Workflow C: Ticket Authentication (CORE)

**TRIGGERS:**
1. User explicitly requests ticket
2. Command output contains auth error (e.g., "missing authorization header")

**Step C1: Get DID**

**Case 1 (auth error trigger)**: Extract DID from failed command's `--did` param. If not found → ask user.

**Case 2 (user request)**: User provides DID.

If no DID → Ask: "Provide DID to authenticate. Example: did:finai:users:0x..."

**Step C2: Execute challenge**

```bash
did_helper ticket challenge --did <DID>
```

**Step C3: Parse output**

Two possible outcomes:
- **Auto mode**: Private key local → auto-signed
- **Manual mode**: JS code for MetaMask/Phantom

**Step C4: Route based on output**

```
IF output contains "auto-signed" or no manual JS:
    → Ticket acquired automatically
    → Output: "✅ Ticket acquired automatically"
    
ELSE IF output contains "Please sign with MetaMask/Phantom":
    → Output to user:
      "⚠️ Manual signature required. Steps:
       1. Open browser console (F12)
       2. Copy/paste the JavaScript code below
       3. Sign with MetaMask/Phantom
       4. Provide challenge AND signature"
    
    WAIT for user to provide challenge + signature
    DO NOT auto-sign or bypass
```

**Step C5: Verify ticket**

For manual mode:
```bash
did_helper ticket verify --did <DID> --challenge "<challenge>" --signature "<signature>"
```

For auto mode:
```bash
did_helper ticket show --did <DID>
```

**Step C6: Parse verification output**

Expected output:
```
✅ Verification successful!
🎫 Ticket: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
⏰ Expire: 2026-10-26T13:13:28Z
```

**Step C7: Expiry warning**

If `Expire` < 15 days from now → Output:
```
⚠️ Ticket expires in <X> days. Please renew.
```

**Step C8: Record state & ask retry**

```
✅ Ticket acquired for DID: <DID> (expires: <date>)

Previous command failed due to missing ticket. Retry? (yes/no)
```

If yes → Re-execute original command.

---

### Workflow D: DID Query (No Ticket)

**Step D1: Extract DID**

User MUST provide DID. If missing → Ask.

**Step D2: Execute**

```bash
did_helper did query --did <DID>
```

**Step D3: Show output**

Add `--original` flag for raw JSON:
```bash
did_helper did query --did <DID> --original
```

**Error handling**: 
- DID not found → Inform user may not exist or deactivated
- Auth error → Auto-trigger Workflow C

---

### Workflow E: DID Update (Auto-detect Ticket)

**Step E1: Extract params**

User MUST provide:
- DID
- Update operation: `--add-key`, `--revoke-key`, `--add-service`, `--remove-service`, `--update-metadata`, or `--request-file`

**Step E2: If `--request-file`**

- User MUST provide absolute path (e.g., `/home/user/update.json`)
- Agent MUST NOT guess path
- Agent MUST NOT run `ls` or `find`
- If file not found → Show error, ask for correct path

**Step E3: Execute**

```bash
did_helper did update --did <DID> --add-key <key_id>
```

**Step E4: Check output & handle**

```
CASE output contains "missing authorization header" or auth error:
    → Auto-trigger Workflow C (with this DID)
    → After success, ask: "Retry update?"
    
CASE output contains success:
    → Show success message
    
CASE other error:
    → Show error, ask user to check params
```

**Max retries: 2** (excluding ticket-triggered retry)

---

### Workflow F: Reputation (Auto-detect Ticket)

**Step F1: Extract DID**

User MUST provide DID. If missing → Ask.

**Step F2: Execute**

```bash
did_helper did reputation --did <DID>
```

**Step F3: Check output & handle**

```
CASE output contains "missing authorization header" or auth error:
    → Auto-trigger Workflow C (with this DID)
    → After success, ask: "Retry reputation query?"
    
CASE output contains reputation info:
    → Show formatted:
      📊 DID Reputation
      ========================
      DID: <DID>
      Level: <level>
      Total Score: <score>
    
CASE other error:
    → Show error
```

---

### Workflow G: DID Deactivation (Auto-detect + Irreversible)

```
🚨 IRREVERSIBLE: This action CANNOT be undone.
```

**Step G1: Extract DID**

User MUST provide DID.

**Step G2: Show WARNING & require explicit confirmation**

```
⚠️⚠️⚠️ IRREVERSIBLE ACTION WARNING ⚠️⚠️⚠️

You are about to deactivate DID: <DID>

This action:
- CANNOT be reversed
- CANNOT be recovered
- DID will be permanently disabled

Type EXACTLY: "CONFIRM DEACTIVATE <DID>"
Type 'cancel' to abort.
```

**Step G3: Wait for user response**

```
if input == f"CONFIRM DEACTIVATE {DID}":
    proceed to Step G4
elif input == "cancel":
    abort
else:
    repeat warning (max 3 times), then abort
```

**Step G4: Execute**

```bash
did_helper did deactivate --did <DID>
```

Optional `--force` (only if user explicitly requests automation):
```bash
did_helper did deactivate --did <DID> --force
```

**Step G5: Check output & handle**

```
CASE output contains "missing authorization header" or auth error:
    → Auto-trigger Workflow C (with this DID)
    → After success, ask: "Retry deactivation? (requires reconfirmation)"
    
CASE output contains success:
    → Output: "✅ DID deactivated: <DID>"
    → Output: "⚠️ This action is permanent and cannot be undone."
    
CASE other error:
    → Show error
```

---

### Workflow H: x402 Payment (Auto-detect Ticket + Wallet Validation)

**PREREQUISITES:**
- Valid DID must exist
- ETH wallet must be imported (Solana NOT supported)
- Valid ticket (auto-detected)
- Payment confirmation for high-value transactions (configurable threshold)

**TRIGGERS:**
- "create payment", "pay", "x402", "order list", "process payment"

**Step H1: Validate Prerequisites**

```python
def validate_x402_prerequisites(did):
    # Check DID exists
    if not did_exists(did):
        raise Error("DID not found. Please create DID first.")
    
    # Extract entity ID from DID
    entity_id = extract_entity_id(did)
    
    # Check ETH wallet exists in import directory
    if not eth_wallet_exists(entity_id):
        raise Error(f"ETH wallet not found for {entity_id}. Run: did_helper key generate --type ethereum")
    
    # Verify it's ETH type (not Solana/X25519)
    if not is_eth_wallet(entity_id):
        raise Error("Only ETH wallets supported for x402. Solana wallets cannot be used.")
```

**Step H2: Execute Command & Auto-detect Auth**

```python
result = execute_did_helper_command(command, did)

# Check for auth errors
if contains_auth_error(result):
    workflow_c_authenticate(did)
    retry_command()
```

**Step H3: Handle Specific Operations**

| Operation | Command | Special Handling |
|-----------|---------|------------------|
| Create Payment | `x402 payment create --did <DID> --amount <AMT> --recipient <ADDR>` | Save order to local storage |
| List Orders | `x402 order list --did <DID> [--status pending] [--limit 10]` | Format table output |
| Show Order | `x402 order show --did <DID> --order-id <ID>` | Display full details |
| Pay | `x402 pay --did <DID> --order-id <ID>` | EIP-712 signing + payment confirmation |
| Status | `x402 order status --did <DID> --order-id <ID>` | Query current status |
| Retry | `x402 order retry --did <DID> --order-id <ID>` | Retry failed payment |
| Cancel | `x402 order cancel --did <DID> --order-id <ID>` | Requires confirmation |

**Step H4: EIP-712 Signing Flow (for `x402 pay`)**

```
1. Get signing requirements from API: GET /order/{orderId}/signing-requirements
2. Load ETH private key from import directory (keystore.json + password.txt)
3. Sign using EIP-712 standard with configured network domain
4. Submit signature: POST /order/{orderId}/process-payment
5. Verify transaction status and update local order cache
```

**Step H5: Payment Confirmation for High-Value Transactions**

```
IF config.payment_confirmation.enabled == true:
    Parse order amount
    IF amount > config.payment_confirmation.usdc_threshold:
        Output: "⚠️ HIGH VALUE PAYMENT WARNING"
        Output: "Amount: {amount} USDC (threshold: {threshold})"
        Output: "Type EXACTLY: CONFIRM PAYMENT {order_id} to proceed"
        
        WAIT for user input
        
        IF input != f"CONFIRM PAYMENT {order_id}":
            Abort payment
        ELSE:
            Continue with EIP-712 signing
```

**Error Patterns to Detect (case-insensitive):**
- `missing authorization header`
- `unauthorized`
- `ticket not found`
- `wallet not found`
- `eth wallet required`
- `solana not supported`
- `insufficient balance`
- `invalid signature`
- `eip-712 signing failed`
- `chain not supported`
- `payment timeout`
- `authentication required`
- `invalid ticket`
- `ticket expired`

**Max retries: 2** (excluding ticket-triggered retry)

**Configuration Notes:**
- Default x402 API: `https://x402.finai.network/testnet/base-sepolia`
- Default network: `base-sepolia` (Chain ID: 84532)
- Payment confirmation threshold: `100.00 USDC` (configurable)
- Supported networks: `base-sepolia`, `ethereum`

---

## Quick Reference: x402 Payment Examples

```bash
# Create payment order
did_helper x402 payment create \
  --did did:finai:users:0x123... \
  --amount "1.00" \
  --recipient "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"

# List orders
did_helper x402 order list --did did:finai:users:0x123...

# Show order details
did_helper x402 order show \
  --did did:finai:users:0x123... \
  --order-id 550e8400-e29b-41d4-a716-446655440000

# Execute payment (with auto EIP-712 signing)
did_helper x402 pay \
  --did did:finai:users:0x123... \
  --order-id 550e8400-e29b-41d4-a716-446655440000

# Check order status
did_helper x402 order status \
  --did did:finai:users:0x123... \
  --order-id 550e8400-e29b-41d4-a716-446655440000

# Retry failed order
did_helper x402 order retry \
  --did did:finai:users:0x123... \
  --order-id 550e8400-e29b-41d4-a716-446655440000

# Cancel pending order
did_helper x402 order cancel \
  --did did:finai:users:0x123... \
  --order-id 550e8400-e29b-41d4-a716-446655440000
```

---

## Generic Command Execution Template

**MUST use this pattern for ALL did_helper executions:**

```python
def execute_did_helper_command(command, did=None):
    # 1. Execute
    result = subprocess.run(command, capture_output=True, text=True)
    
    # 2. Check for auth errors (case-insensitive)
    auth_errors = [
        "missing authorization header",
        "unauthorized", 
        "ticket not found",
        "authentication required",
        "invalid ticket",
        "ticket expired",
        "wallet not found",
        "eth wallet required",
        "solana not supported",
        "insufficient balance",
        "invalid signature",
        "eip-712 signing failed",
        "chain not supported",
        "payment timeout"
    ]
    
    output_lower = (result.stdout + result.stderr).lower()
    
    for error in auth_errors:
        if error in output_lower:
            # Ticket required
            print("⚠️ Ticket authentication required")
            
            # Extract DID or use provided
            target_did = did or extract_did_from_command(command)
            
            if target_did:
                success = workflow_c_authenticate(target_did)
                if success:
                    if ask_user("Ticket acquired. Retry original command? (yes/no)"):
                        return execute_did_helper_command(command, target_did)
                else:
                    print("❌ Ticket authentication failed")
                    return None
            else:
                print("❌ Cannot determine DID for authentication")
                return None
    
    # 3. Return result
    return result
```

---

## Agent Internal State

```python
state = {
    "pending_retry_command": None,  # Original command that failed due to missing ticket
    "pending_retry_did": None,      # Corresponding DID
    "authenticated_dids": {         # Optional optimization
        "did:finai:users:0x...": {
            "expiry": "2026-10-26T13:13:28Z"
        }
    }
}
```

**State rules:**
- Command fails (auth error) → Save `pending_retry_command` + `pending_retry_did`
- Ticket success → Ask user to retry
- User confirms → Re-execute saved command
- After retry or user declines → Clear pending state

**FORBIDDEN:**
- Persist state across sessions (no file writes)
- Store actual ticket strings

---

## Quick Reference: Command Flow

```
User Request
    │
    ├─ "key generate" ──→ Workflow A (direct)
    │
    ├─ "did create" ──→ Workflow B (direct)
    │
    ├─ "did query" ──→ Workflow D (direct)
    │
    ├─ "did update / reputation / deactivate" ──→ Execute
    │                                              │
    │                                              ▼
    │                                      Check Output
    │                                              │
    │                          ┌───────────────────┴───────────────────┐
    │                          │                                       │
    │                   Contains auth error                      Success
    │                          │                                       │
    │                          ▼                                       ▼
    │                  Auto-trigger Workflow C                   Show Result
    │                          │
    │                          ▼
    │                  Ask: "Retry?"
    │                          │
    │                          ▼
    │                  Re-execute original
    │
    ├─ "apikey *" ──→ FORBIDDEN → Manual instruction
    │
    └─ Other ──→ Ask clarification
```

---

## Error Response Templates

### Missing Required Parameter
```
❌ Missing required: {param}

Please provide: {description}

Example: {example}
```

### No Ticket (detected from output)
```
⚠️ Command requires ticket (detected: {auth_error})

Auto-triggering Workflow C...
```

### apikey Forbidden
```
I cannot execute apikey commands per skill rules.

Please run manually:
did_helper apikey {command}
```
