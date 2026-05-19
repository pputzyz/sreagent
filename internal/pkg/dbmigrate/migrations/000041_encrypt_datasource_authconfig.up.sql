-- Migration: Encrypt DataSource AuthConfig at rest
-- The actual encryption is handled lazily in the service layer (AES-256-GCM).
-- Existing plaintext AuthConfig values will be encrypted on next Update through the API.
-- This migration is a no-op marker for tracking purposes.
SELECT 1;
