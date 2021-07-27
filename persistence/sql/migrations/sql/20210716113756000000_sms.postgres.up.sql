CREATE TABLE "sms_codes" (
"id" UUID NOT NULL,
PRIMARY KEY("id"),
"phone" VARCHAR (255) NOT NULL,
"code" VARCHAR (255) NOT NULL,
"flow_id" UUID NOT NULL,
"expires_at" timestamp NOT NULL,
"created_at" timestamp NOT NULL,
"updated_at" timestamp NOT NULL
);