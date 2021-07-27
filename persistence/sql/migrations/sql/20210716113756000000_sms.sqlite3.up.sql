CREATE TABLE "sms_codes" (
"id" TEXT PRIMARY KEY,
"phone" TEXT NOT NULL,
"code" TEXT NOT NULL,
"flow_id" char(36) NOT NULL,
"expires_at" DATETIME NOT NULL,
"created_at" DATETIME NOT NULL,
"updated_at" DATETIME NOT NULL
);