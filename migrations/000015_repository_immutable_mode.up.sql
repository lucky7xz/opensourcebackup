-- B_IMM: Add immutable_mode to repositories table.
--
-- immutable_mode documents the write-protection mechanism of the repository.
-- This is a declaration by the operator — the Control Plane does not
-- automatically verify the storage-level configuration.
--
-- Values:
--   none         No write protection configured (default, highest risk)
--   object_lock  S3 Object Lock / MinIO Object Lock (WORM at object level)
--   worm         Hardware or storage-level WORM (NAS, tape, etc.)
--   append_only  Append-only access (e.g. restic --append-only sftp backend)
--   unknown      Protection status not verified
--
-- The existing object_lock_enabled boolean is kept for backward compatibility
-- but new code should prefer immutable_mode.

ALTER TABLE repositories
    ADD COLUMN IF NOT EXISTS immutable_mode TEXT NOT NULL DEFAULT 'none';

-- Back-fill: existing rows with object_lock_enabled=true → 'object_lock'
UPDATE repositories SET immutable_mode = 'object_lock' WHERE object_lock_enabled = true;

COMMENT ON COLUMN repositories.immutable_mode IS
    'Write-protection mechanism: none | object_lock | worm | append_only | unknown';
