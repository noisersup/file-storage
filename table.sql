CREATE TABLE "file_tree" (
  "id" UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  "encrypted_name" STRING(255),
  "hash" STRING(64),
  "duplicate" INT,
  "parent_id" UUID,
  "is_directory" BOOL,
  CONSTRAINT "primary" PRIMARY KEY (id ASC)
);

CREATE UNIQUE INDEX fileDupliaction ON file_tree (encrypted_name, parent_id);

CREATE TABLE "file_tree_config" (
    "root" UUID NOT NULL
);

CREATE TABLE "users" (
  	"id" UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
	"username" STRING(16) NOT NULL UNIQUE,
	"password" STRING(60) NOT NULL,
	"key"	   STRING(44) NOT NULL UNIQUE,
  	CONSTRAINT "primary" PRIMARY KEY (username)
);
