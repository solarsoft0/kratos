CREATE TABLE `sms_codes` (
`id` char(36) NOT NULL,
PRIMARY KEY(`id`),
`phone` VARCHAR (255) NOT NULL,
`code` VARCHAR (255) NOT NULL,
`flow_id` char(36) NOT NULL,
`expires_at` DATETIME NOT NULL,
`created_at` DATETIME NOT NULL,
`updated_at` DATETIME NOT NULL
) ENGINE=InnoDB;