CREATE DATABASE `test`;

USE test;

CREATE TABLE `user` (
    `uid` int NOT NULL AUTO_INCREMENT,
    `username` varchar(64) DEFAULT NULL,
    `departname` varchar(64) DEFAULT NULL,
    `created` date DEFAULT NULL,
    `status` int NOT NULL,
    PRIMARY KEY (`uid`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4;