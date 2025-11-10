# Ride Hailing application policy for HashiCorp Vault.
# Grants least-privilege access to required secret paths and enables credential rotation/auditing.

path "kv/data/ride-hailing/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "kv/metadata/ride-hailing/*" {
  capabilities = ["read", "list"]
}

# Allow destroying specific secret versions when rotating credentials.
path "kv/destroy/ride-hailing/*" {
  capabilities = ["update"]
}

# Database credentials
path "kv/data/ride-hailing/database" {
  capabilities = ["create", "read", "update", "list"]
}

# Stripe API keys
path "kv/data/ride-hailing/stripe" {
  capabilities = ["create", "read", "update", "list"]
}

# Firebase / Google service accounts
path "kv/data/ride-hailing/firebase" {
  capabilities = ["create", "read", "update", "list"]
}

# Twilio credentials
path "kv/data/ride-hailing/twilio" {
  capabilities = ["create", "read", "update", "list"]
}

# SMTP credentials
path "kv/data/ride-hailing/smtp" {
  capabilities = ["create", "read", "update", "list"]
}

# JWT signing keys managed by jwtkeys package
path "kv/data/ride-hailing/jwt-keys" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Allow reading audit devices so applications can confirm logging is enabled.
path "sys/audit" {
  capabilities = ["read", "list"]
}
