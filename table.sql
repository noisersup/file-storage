CREATE TABLE "file_tree" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "name" STRING(255),
  "hash" STRING(32),
  "parent_id" UUID,
  CONSTRAINT "primary" PRIMARY KEY (id ASC)
);

CREATE UNIQUE INDEX fileDupliaction ON file_tree (encrypted_name, parent_id);

CREATE TABLE "file_tree_config" (
    "root" UUID NOT NULL
);

CREATE TABLE "users" (
	"username" STRING(16) NOT NULL,
	"password" STRING(60),
  	CONSTRAINT "primary" PRIMARY KEY (username)
);
