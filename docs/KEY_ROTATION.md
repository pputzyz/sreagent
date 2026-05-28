# Key Rotation Guide

## Overview

SREAgent uses AES-256-GCM encryption for sensitive data (datasource credentials, API keys). The encryption key is loaded from the `SREAGENT_SECRET_KEY` environment variable at startup.

## Current Limitations

- **Single key**: Only one key is active at a time. Changing the key requires:
  1. Stop the SREAgent server
  2. Decrypt all existing encrypted data with the old key (see script below)
  3. Set the new key in `SREAGENT_SECRET_KEY`
  4. Re-encrypt all data with the new key
  5. Restart the server

- **No hot reload**: The key is loaded once at startup via `sync.Once`. Changing the env var without a restart has no effect.

- **No key versioning**: Encrypted values do not carry a key identifier. All values must use the same key.

## Key Format

- 64-character hexadecimal string (32 bytes)
- Generate with: `openssl rand -hex 32`
- Example: `0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`

## Encrypted Value Format

Encrypted values are stored with the prefix `enc:` followed by base64-encoded (nonce + ciphertext):

```
enc:<base64(12-byte-nonce + aes-gcm-ciphertext)>
```

The `IsEncrypted()` function checks for this prefix to distinguish encrypted values from legacy plaintext.

## Rotation Procedure

### Step 1: Stop the server

```bash
docker compose down   # or systemctl stop sreagent
```

### Step 2: Export current data

```bash
# Identify columns that contain encrypted values
mysql -u root -p sreagent -e "
  SELECT id, auth_config FROM data_sources WHERE auth_config LIKE 'enc:%';
" > /tmp/encrypted_export.sql
```

### Step 3: Decrypt with old key

Use a one-off script or the SREAgent CLI (if available) to decrypt each value with the old key.

### Step 4: Update the key

```bash
export SREAGENT_SECRET_KEY=$(openssl rand -hex 32)
```

Update your systemd unit, docker-compose.yml, or Kubernetes secret accordingly.

### Step 5: Re-encrypt and write back

Re-encrypt each decrypted value with the new key and UPDATE the database rows.

### Step 6: Restart

```bash
docker compose up -d
```

## Emergency: Lost Key

If `SREAGENT_SECRET_KEY` is lost, all encrypted datasource credentials become **unrecoverable**. Recovery steps:

1. Clear the encrypted columns:
   ```sql
   UPDATE data_sources SET auth_config = '', auth_type = 'none';
   ```
2. Users must re-enter credentials for all datasources through the UI.
3. Generate a new key: `openssl rand -hex 32`

## Verifying Encryption State

Check if any values are still in plaintext:

```sql
SELECT id, name FROM data_sources WHERE auth_config != '' AND auth_config NOT LIKE 'enc:%';
```

These rows were created before encryption was enabled and should be re-saved through the UI to encrypt them.

## Future Improvements

- Key versioning with envelope encryption (KMS integration)
- Hot-reload support via SIGHUP or config file watch
- Key derivation from master secret + per-value salt
- Automated rotation script with rollback support
