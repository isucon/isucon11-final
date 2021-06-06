CREATE DATABASE IF NOT EXISTS `isucholar` DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_bin;
CREATE USER IF NOT EXISTS `isucon`@`localhost` IDENTIFIED WITH mysql_native_password BY 'isucon';
GRANT ALL ON `isucholar`.* TO `isucon`@`localhost`;
