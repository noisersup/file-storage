CREATE TABLE "file_tree" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "encrypted_name" STRING(255),
  "parent_id" UUID,
  CONSTRAINT "primary" PRIMARY KEY (id ASC)
);

CREATE UNIQUE INDEX fileDupliaction ON file_tree (encrypted_name, parent_id);
