-- Add CASCADE to foreign keys so deleting a system cleans up tokens automatically
ALTER TABLE agent_enrollment_tokens
  DROP CONSTRAINT agent_enrollment_tokens_system_id_fkey,
  ADD CONSTRAINT agent_enrollment_tokens_system_id_fkey
    FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;

ALTER TABLE agent_tokens
  DROP CONSTRAINT agent_tokens_system_id_fkey,
  ADD CONSTRAINT agent_tokens_system_id_fkey
    FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;
