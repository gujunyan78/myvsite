-- VSITE 数据库初始化脚本
-- 使用具有 CREATE 权限的 MySQL 账号执行，例如：
--   sudo mysql < scripts/init_db.sql
-- 或： mysql -uroot -p < scripts/init_db.sql

CREATE DATABASE IF NOT EXISTS vsite
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

-- 应用连接账号（开发交付约定的账号密码）
CREATE USER IF NOT EXISTS 'admin'@'localhost' IDENTIFIED BY '!QAZ2wsx';
CREATE USER IF NOT EXISTS 'admin'@'127.0.0.1' IDENTIFIED BY '!QAZ2wsx';
CREATE USER IF NOT EXISTS 'admin'@'%' IDENTIFIED BY '!QAZ2wsx';

GRANT ALL PRIVILEGES ON vsite.* TO 'admin'@'localhost';
GRANT ALL PRIVILEGES ON vsite.* TO 'admin'@'127.0.0.1';
GRANT ALL PRIVILEGES ON vsite.* TO 'admin'@'%';

FLUSH PRIVILEGES;
