ALTER TABLE controlplane_projects
    ADD COLUMN list_ordinal BIGINT GENERATED ALWAYS AS IDENTITY;
ALTER TABLE controlplane_projects
    ADD CONSTRAINT controlplane_projects_tenant_list_ordinal_key
    UNIQUE (tenant_id, list_ordinal);

ALTER TABLE controlplane_environments
    ADD COLUMN list_ordinal BIGINT GENERATED ALWAYS AS IDENTITY;
ALTER TABLE controlplane_environments
    ADD CONSTRAINT controlplane_environments_tenant_list_ordinal_key
    UNIQUE (tenant_id, list_ordinal);

CREATE INDEX controlplane_projects_tenant_page
    ON controlplane_projects (tenant_id, list_ordinal);
CREATE INDEX controlplane_environments_tenant_page
    ON controlplane_environments (tenant_id, list_ordinal);
