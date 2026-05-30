ALTER TABLE backup_policies
    ADD COLUMN repository_id UUID REFERENCES repositories(id);
