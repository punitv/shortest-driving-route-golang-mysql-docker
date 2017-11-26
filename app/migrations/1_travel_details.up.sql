CREATE TABLE `travel_details` (
    `id` INT(10) NOT NULL AUTO_INCREMENT,
    `path` VARCHAR(5000) NULL DEFAULT NULL,
    `token` VARCHAR(64) NULL DEFAULT NULL,
    `status` TINYINT NULL DEFAULT NULL,
    `total_distance` INT NULL DEFAULT NULL,
    `total_time` INT NULL DEFAULT NULL,
    `response_error` VARCHAR(1000) NULL DEFAULT NULL,
    PRIMARY KEY (`id`)
) character set utf8mb4 collate utf8mb4_unicode_ci;