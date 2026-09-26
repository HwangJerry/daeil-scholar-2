-- Production schema baseline for tests (schema only, no rows).
-- Source: read-only `mysqldump --no-data --routines --triggers --events` of daeil-prod
-- (MariaDB 10.1.38) on 2026-09-27, when migrations 001-077 were applied.
-- AUTO_INCREMENT values and DEFINER clauses removed; mysql-client DELIMITER lines
-- removed so the Go driver can execute it with multiStatements.
-- Refresh: re-dump the same way and replace this file together with
-- prod_baseline_applied_migrations_20260927.sql.
/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ACCOUNT_DELETION_INTAKE` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `OPERATOR_SEQ` int(11) NOT NULL,
  `EVIDENCE_REFERENCE` varchar(200) NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`REQUEST_ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ACCOUNT_DELETION_REQUEST` (
  `REQUEST_ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) DEFAULT NULL,
  `RECEIPT_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STATUS` varchar(20) NOT NULL DEFAULT 'pending',
  `APPLE_REQUIRED` tinyint(4) NOT NULL DEFAULT '0',
  `KAKAO_REQUIRED` tinyint(4) NOT NULL DEFAULT '0',
  `REQUESTED_AT` datetime NOT NULL,
  `TARGET_AT` datetime NOT NULL,
  `DUE_AT` datetime NOT NULL,
  `COMPLETED_AT` datetime DEFAULT NULL,
  `OPERATOR_SEQ` int(11) DEFAULT NULL,
  `EVIDENCE_REFERENCE` varchar(1000) NOT NULL DEFAULT '',
  `RETAINED_RECORDS` varchar(1000) NOT NULL DEFAULT '',
  `RETENTION_UNTIL` date DEFAULT NULL,
  PRIMARY KEY (`REQUEST_ID`),
  UNIQUE KEY `UK_ADR_RECEIPT` (`RECEIPT_HASH`),
  UNIQUE KEY `UK_ADR_USER` (`USR_SEQ`),
  KEY `IDX_ADR_QUEUE` (`STATUS`,`DUE_AT`),
  KEY `IDX_ADR_COMPLETE` (`COMPLETED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ACCOUNT_ERASURE` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `MODE` varchar(10) NOT NULL DEFAULT 'automatic',
  `STAGE` varchar(30) NOT NULL DEFAULT 'queued',
  `LAST_CODE` varchar(80) NOT NULL DEFAULT '',
  `NEXT_ATTEMPT_AT` datetime NOT NULL,
  `EXTERNAL_EVIDENCE` varchar(200) NOT NULL DEFAULT '',
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`REQUEST_ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ADMIN_ROLE` (
  `USR_SEQ` int(11) NOT NULL,
  `ADMIN_ROLE` enum('root','operator') NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  `CREATED_BY` int(11) DEFAULT NULL,
  `UPDATED_BY` int(11) DEFAULT NULL,
  PRIMARY KEY (`USR_SEQ`),
  KEY `IDX_AAR_ROLE` (`ADMIN_ROLE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_APPLE_CODE_REPLAY` (
  `CODE_HASH` char(64) NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`CODE_HASH`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_APPLE_NONCE_CHALLENGE` (
  `CHALLENGE_ID` varchar(64) NOT NULL,
  `NONCE_HASH` char(64) NOT NULL,
  `EXPIRES_AT` datetime NOT NULL,
  `CONSUMED_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`CHALLENGE_ID`),
  KEY `IDX_ANC_EXPIRES` (`EXPIRES_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_COMMENT_REPORT` (
  `REPORT_ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `POST_SEQ` bigint(20) NOT NULL,
  `COMMENT_SEQ` bigint(20) NOT NULL,
  `REPORTER_SEQ` int(11) NOT NULL,
  `REPORTED_SEQ` int(11) NOT NULL,
  `REASON` varchar(32) NOT NULL,
  `DETAILS` text NOT NULL,
  `CONTENT_SNAPSHOT` text NOT NULL,
  `STATUS` varchar(16) NOT NULL DEFAULT 'open',
  `MODERATOR_SEQ` int(11) DEFAULT NULL,
  `MODERATOR_NOTE` text,
  `CREATED_AT` datetime NOT NULL,
  `RESOLVED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`REPORT_ID`),
  UNIQUE KEY `uq_comment_report` (`REPORTER_SEQ`,`COMMENT_SEQ`),
  KEY `idx_report_queue` (`STATUS`,`REPORT_ID`),
  KEY `idx_comment_reports` (`COMMENT_SEQ`),
  KEY `idx_comment_retention` (`RESOLVED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_DONATION_ARCHIVE_ACCESS` (
  `ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `ARCHIVE_ID` bigint(20) NOT NULL,
  `OPERATOR_SEQ` int(11) NOT NULL,
  `PURPOSE` varchar(30) NOT NULL,
  `ACCESSED_AT` datetime NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `IDX_ARCHIVE_ACCESS` (`ARCHIVE_ID`,`ACCESSED_AT`),
  KEY `IDX_OPERATOR_ACCESS` (`OPERATOR_SEQ`,`ACCESSED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_DONATION_LEGAL_ARCHIVE` (
  `ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `BASIS` varchar(30) NOT NULL,
  `RETAIN_UNTIL` date NOT NULL,
  `CIPHERTEXT` mediumblob NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `IDX_LEGAL_EXPIRY` (`RETAIN_UNTIL`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_DONATION_RETENTION` (
  `O_SEQ` int(11) NOT NULL,
  `BASIS` varchar(30) NOT NULL,
  `BASIS_DATE` date DEFAULT NULL,
  `RETAIN_UNTIL` date DEFAULT NULL,
  `EVIDENCE_REFERENCE` varchar(200) NOT NULL,
  `REVIEWED_AT` datetime NOT NULL,
  `SOURCE_FINGERPRINT` char(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`O_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASED_DONATION_TOTAL` (
  `ID` tinyint(4) NOT NULL,
  `TOTAL_AMOUNT` bigint(20) NOT NULL DEFAULT '0',
  `DONOR_COUNT` bigint(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_CANCELLATION` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `ORIGINAL_STATUS` char(3) NOT NULL,
  `CANCEL_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
  `RESTORE_BLOCKED` tinyint(4) NOT NULL DEFAULT '0',
  `STARTED_AT` datetime DEFAULT NULL,
  `CANCELLED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`REQUEST_ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_CONTEXT` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `CIPHERTEXT` mediumblob NOT NULL,
  `EXPIRES_AT` datetime NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`REQUEST_ID`),
  KEY `idx_erasure_context_expiry` (`EXPIRES_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_FILE` (
  `ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `REQUEST_ID` bigint(20) NOT NULL,
  `URL_PATH` text NOT NULL,
  `URL_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `UK_ERASURE_FILE` (`REQUEST_ID`,`URL_HASH`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_RECEIPT_WORK` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `STATUS` varchar(20) NOT NULL DEFAULT 'unreviewed',
  `ORIGINAL_STORAGE` varchar(30) NOT NULL DEFAULT '',
  `EVIDENCE_REFERENCE` varchar(200) NOT NULL DEFAULT '',
  `OPERATOR_SEQ` int(11) DEFAULT NULL,
  `UPDATED_AT` datetime NOT NULL,
  `COMPLETED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`REQUEST_ID`),
  CONSTRAINT `FK_RECEIPT_WORK_REQUEST` FOREIGN KEY (`REQUEST_ID`) REFERENCES `ALUMNI_ACCOUNT_DELETION_REQUEST` (`REQUEST_ID`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_RESTORE_GUARD` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `COMPLETED_AT` datetime NOT NULL,
  `EXPIRES_AT` datetime NOT NULL,
  PRIMARY KEY (`REQUEST_ID`),
  KEY `IDX_RESTORE_GUARD_EXPIRY` (`EXPIRES_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_SCHEDULE` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `SCHEDULED_AT` datetime NOT NULL,
  `EXPEDITED_AT` datetime DEFAULT NULL,
  `EXPEDITED_BY` int(11) DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`REQUEST_ID`),
  KEY `IDX_ERASURE_SCHEDULE` (`SCHEDULED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_ERASURE_TARGET` (
  `REQUEST_ID` bigint(20) NOT NULL,
  `TARGET` varchar(30) NOT NULL,
  `STATUS` varchar(20) NOT NULL DEFAULT 'pending',
  `EVIDENCE_REFERENCE` varchar(200) NOT NULL DEFAULT '',
  `LAST_CODE` varchar(80) NOT NULL DEFAULT '',
  `ATTEMPTS` int(11) NOT NULL DEFAULT '0',
  `LAST_ATTEMPT_AT` datetime DEFAULT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`REQUEST_ID`,`TARGET`),
  CONSTRAINT `FK_ERASURE_TARGET_REQUEST` FOREIGN KEY (`REQUEST_ID`) REFERENCES `ALUMNI_ACCOUNT_DELETION_REQUEST` (`REQUEST_ID`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_JOB_CATEGORY` (
  `AJC_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `AJC_NAME` varchar(50) NOT NULL,
  `AJC_COLOR` varchar(7) DEFAULT '#4F46E5',
  `AJC_INDX` int(11) DEFAULT '0',
  `OPEN_YN` enum('Y','N') DEFAULT 'Y',
  `REG_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`AJC_SEQ`),
  UNIQUE KEY `UK_NAME` (`AJC_NAME`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_MEMBER_BLOCK` (
  `AMB_SEQ` bigint(20) NOT NULL AUTO_INCREMENT,
  `BLOCKER_USR_SEQ` int(11) NOT NULL,
  `BLOCKED_USR_SEQ` int(11) NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`AMB_SEQ`),
  UNIQUE KEY `UK_AMB_DIRECTION` (`BLOCKER_USR_SEQ`,`BLOCKED_USR_SEQ`),
  KEY `IDX_AMB_BLOCKED` (`BLOCKED_USR_SEQ`,`BLOCKER_USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_MESSAGE` (
  `AM_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `AM_SENDER_SEQ` int(11) NOT NULL,
  `AM_RECVR_SEQ` int(11) NOT NULL,
  `AM_CONTENT` text NOT NULL,
  `AM_READ_YN` enum('Y','N') DEFAULT 'N',
  `AM_DEL_SENDER` enum('Y','N') DEFAULT 'N',
  `AM_DEL_RECVR` enum('Y','N') DEFAULT 'N',
  `REG_DATE` datetime DEFAULT NULL,
  `READ_DATE` datetime DEFAULT NULL,
  `AM_CLIENT_MESSAGE_ID` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `AM_VISIBLE_RECVR` enum('Y','N') NOT NULL DEFAULT 'Y',
  `AM_SUPPRESSION_REASON` varchar(30) DEFAULT NULL,
  `PURGE_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`AM_SEQ`),
  UNIQUE KEY `UK_AM_SENDER_CLIENT` (`AM_SENDER_SEQ`,`AM_CLIENT_MESSAGE_ID`),
  KEY `IDX_SENDER` (`AM_SENDER_SEQ`,`REG_DATE`),
  KEY `IDX_RECVR` (`AM_RECVR_SEQ`,`AM_READ_YN`,`REG_DATE`),
  KEY `IDX_AM_RECVR_VISIBLE` (`AM_RECVR_SEQ`,`AM_VISIBLE_RECVR`,`REG_DATE`),
  KEY `IDX_AM_PURGE` (`PURGE_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_MESSAGE_REPORT` (
  `REPORT_ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `MESSAGE_ID` bigint(20) NOT NULL,
  `REPORTER_SEQ` int(11) NOT NULL,
  `REPORTED_SEQ` int(11) NOT NULL,
  `REASON` varchar(32) NOT NULL,
  `DETAILS` text NOT NULL,
  `CONTENT_SNAPSHOT` text NOT NULL,
  `STATUS` varchar(16) NOT NULL DEFAULT 'open',
  `MODERATOR_SEQ` int(11) DEFAULT NULL,
  `MODERATOR_NOTE` text,
  `CREATED_AT` datetime NOT NULL,
  `RESOLVED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`REPORT_ID`),
  UNIQUE KEY `uq_message_report` (`REPORTER_SEQ`,`MESSAGE_ID`),
  KEY `idx_report_queue` (`STATUS`,`CREATED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_MOBILE_APP_EVENT` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `PLATFORM` varchar(16) NOT NULL,
  `EVENT_TYPE` varchar(50) NOT NULL,
  `USER_ID` int(11) DEFAULT NULL,
  `APP_VERSION` varchar(50) NOT NULL,
  `OS_VERSION` varchar(50) NOT NULL,
  `DEVICE_MODEL` varchar(100) DEFAULT NULL,
  `OCCURRED_AT` datetime NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `IDX_AME_PLATFORM` (`PLATFORM`),
  KEY `IDX_AME_EVENT_TYPE` (`EVENT_TYPE`),
  KEY `IDX_AME_OCCURRED_AT` (`OCCURRED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_MOBILE_DEVICE_TOKEN` (
  `MDT_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) NOT NULL,
  `PLATFORM` varchar(16) NOT NULL,
  `DEVICE_TOKEN` varchar(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `APNS_ENVIRONMENT` enum('sandbox','production') DEFAULT NULL,
  `BUNDLE_ID` varchar(255) DEFAULT NULL,
  `LOCALE` varchar(16) DEFAULT NULL,
  `STATUS` enum('ACTIVE','INACTIVE','STALE','UNVERIFIED','REVOKED') DEFAULT 'ACTIVE',
  `INVALID_COUNT` int(11) NOT NULL DEFAULT '0',
  `LAST_SEEN_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime DEFAULT NULL,
  `UPDATED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`MDT_SEQ`),
  UNIQUE KEY `UK_MOBILE_DEVICE_TOKEN` (`DEVICE_TOKEN`),
  KEY `IDX_USR` (`USR_SEQ`,`STATUS`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_MOBILE_REFRESH_TOKEN` (
  `MRT_JTI` varchar(64) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `MRT_SID` varchar(64) NOT NULL,
  `EXPIRES_AT` datetime NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  `CONSUMED_AT` datetime DEFAULT NULL,
  `REVOKED_AT` datetime DEFAULT NULL,
  `ROTATED_TO_JTI` varchar(64) DEFAULT NULL,
  `MRT_REVOKED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`MRT_JTI`),
  KEY `IDX_USR` (`USR_SEQ`),
  KEY `IDX_EXPIRES` (`EXPIRES_AT`),
  KEY `IDX_REVOKED` (`MRT_REVOKED_AT`),
  KEY `IDX_MRT_SID` (`MRT_SID`),
  KEY `IDX_MRT_REVOKED_AT` (`REVOKED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_NOTIFICATION` (
  `AN_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) NOT NULL,
  `AN_TYPE` varchar(30) NOT NULL,
  `AN_TITLE` varchar(200) NOT NULL,
  `AN_BODY` varchar(500) DEFAULT NULL,
  `AN_REF_SEQ` int(11) DEFAULT NULL,
  `AN_READ_YN` enum('Y','N') DEFAULT 'N',
  `REG_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`AN_SEQ`),
  KEY `IDX_USR_READ` (`USR_SEQ`,`AN_READ_YN`,`REG_DATE`),
  KEY `IDX_USR_DATE` (`USR_SEQ`,`REG_DATE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_PASSWORD_RESET` (
  `APR_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) NOT NULL,
  `APR_TOKEN` varchar(64) NOT NULL,
  `APR_USED_YN` enum('Y','N') DEFAULT 'N',
  `EXPIRES_AT` datetime NOT NULL,
  `REG_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`APR_SEQ`),
  UNIQUE KEY `IDX_TOKEN` (`APR_TOKEN`),
  KEY `IDX_USR` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_PHONE_VERIFICATION` (
  `APV_SEQ` bigint(20) NOT NULL AUTO_INCREMENT,
  `APV_ID` char(32) NOT NULL,
  `PHONE` varchar(20) NOT NULL,
  `CODE_HASH` char(64) NOT NULL,
  `GRANT_TOKEN_HASH` char(64) DEFAULT NULL,
  `ATTEMPTS` int(11) NOT NULL DEFAULT '0',
  `VERIFIED_YN` char(1) NOT NULL DEFAULT 'N',
  `CONSUMED_YN` char(1) NOT NULL DEFAULT 'N',
  `EXPIRES_AT` datetime NOT NULL,
  `GRANT_EXPIRES_AT` datetime DEFAULT NULL,
  `REG_DATE` datetime NOT NULL,
  PRIMARY KEY (`APV_SEQ`),
  UNIQUE KEY `uq_phone_verification_id` (`APV_ID`),
  UNIQUE KEY `uq_phone_verification_grant` (`GRANT_TOKEN_HASH`),
  KEY `idx_phone_verification_throttle` (`PHONE`,`REG_DATE`),
  KEY `idx_phone_verification_retention` (`REG_DATE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_PROFILE_FILE_HISTORY` (
  `USR_SEQ` int(11) NOT NULL,
  `URL_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `URL_PATH` text NOT NULL,
  `RECORDED_AT` datetime NOT NULL,
  PRIMARY KEY (`USR_SEQ`,`URL_HASH`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_PUSH_OUTBOX` (
  `PO_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `EVENT_TYPE` varchar(64) NOT NULL,
  `EVENT_ID` varchar(191) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `MDT_SEQ` int(11) NOT NULL,
  `DEVICE_TOKEN` varchar(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `APNS_ENVIRONMENT` enum('sandbox','production') NOT NULL DEFAULT 'production',
  `BUNDLE_ID` varchar(255) DEFAULT NULL,
  `TITLE` varchar(255) NOT NULL,
  `BODY` varchar(1000) NOT NULL,
  `PAYLOAD_JSON` longtext NOT NULL,
  `STATUS` enum('PENDING','PROCESSING','SENT','FAILED','DEAD') NOT NULL DEFAULT 'PENDING',
  `ATTEMPT_COUNT` int(11) NOT NULL DEFAULT '0',
  `NEXT_ATTEMPT_AT` datetime NOT NULL,
  `LAST_ERROR_CODE` varchar(128) DEFAULT NULL,
  `LAST_ERROR_MESSAGE` varchar(1000) DEFAULT NULL,
  `CLAIM_TOKEN` varchar(64) DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `UPDATED_AT` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `SENT_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`PO_SEQ`),
  UNIQUE KEY `UK_PUSH_OUTBOX_EVENT_TOKEN` (`EVENT_ID`,`MDT_SEQ`),
  KEY `IDX_PUSH_OUTBOX_DUE` (`STATUS`,`NEXT_ATTEMPT_AT`),
  KEY `IDX_PUSH_OUTBOX_CLAIM` (`CLAIM_TOKEN`),
  KEY `IDX_PUSH_OUTBOX_CREATED` (`CREATED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_PUSH_PREFERENCE` (
  `USR_SEQ` int(11) NOT NULL,
  `NOTICE_ENABLED` enum('Y','N') NOT NULL DEFAULT 'Y',
  `MESSAGE_ENABLED` enum('Y','N') NOT NULL DEFAULT 'Y',
  `MESSAGE_PREVIEW_ENABLED` enum('Y','N') NOT NULL DEFAULT 'Y',
  `CREATED_AT` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `UPDATED_AT` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`USR_SEQ`),
  KEY `IDX_PUSH_PREFERENCE_NOTICE` (`NOTICE_ENABLED`),
  KEY `IDX_PUSH_PREFERENCE_MESSAGE` (`MESSAGE_ENABLED`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_SOCIAL_CREDENTIAL` (
  `USR_SEQ` int(11) NOT NULL,
  `PROVIDER` varchar(10) NOT NULL,
  `ENCRYPTED_CREDENTIAL` text NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`USR_SEQ`,`PROVIDER`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_SOCIAL_LINK_CONTINUATION` (
  `SLC_TOKEN_HASH` char(64) NOT NULL,
  `SLC_PROVIDER` varchar(20) NOT NULL,
  `SLC_SUBJECT` varchar(191) NOT NULL,
  `SLC_EMAIL` varchar(255) DEFAULT NULL,
  `SLC_STATUS` enum('READY','CONSUMED') NOT NULL DEFAULT 'READY',
  `SLC_EXPIRES_AT` datetime NOT NULL,
  `SLC_CONSUMED_AT` datetime DEFAULT NULL,
  `SLC_CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`SLC_TOKEN_HASH`),
  KEY `IDX_SLC_STATUS_EXPIRY` (`SLC_STATUS`,`SLC_EXPIRES_AT`),
  KEY `IDX_SLC_PROVIDER_SUBJECT` (`SLC_PROVIDER`,`SLC_SUBJECT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_SOCIAL_LINK_REAUTH_GUARD` (
  `SLR_PROVIDER` varchar(20) NOT NULL,
  `SLR_SUBJECT` varchar(191) NOT NULL,
  `SLR_FAILED_ATTEMPTS` int(11) NOT NULL DEFAULT '0',
  `SLR_LOCKED_AT` datetime DEFAULT NULL,
  `SLR_EXPIRES_AT` datetime NOT NULL,
  `SLR_UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`SLR_PROVIDER`,`SLR_SUBJECT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_SOCIAL_REVOCATION_OUTBOX` (
  `OUTBOX_ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) NOT NULL,
  `PROVIDER` varchar(10) NOT NULL,
  `ACTION` varchar(30) NOT NULL,
  `STATUS` varchar(20) NOT NULL,
  `CLAIM_TOKEN` varchar(64) DEFAULT NULL,
  `ATTEMPT_COUNT` int(11) NOT NULL DEFAULT '0',
  `NEXT_ATTEMPT_AT` datetime NOT NULL,
  `LAST_ERROR` varchar(500) DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  `OPEN_ACTION` varchar(30) AS (
            CASE WHEN STATUS IN ('PENDING','PROCESSING') THEN ACTION ELSE NULL END) PERSISTENT,
  PRIMARY KEY (`OUTBOX_ID`),
  UNIQUE KEY `UQ_SRO_OPEN_ACTION` (`USR_SEQ`,`PROVIDER`,`OPEN_ACTION`),
  KEY `IDX_SRO_DUE` (`STATUS`,`NEXT_ATTEMPT_AT`),
  KEY `IDX_SRO_CLAIM` (`CLAIM_TOKEN`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_UPLOAD_OWNER` (
  `F_SEQ` int(11) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `URL_PATH` text NOT NULL,
  PRIMARY KEY (`F_SEQ`),
  KEY `IDX_UPLOAD_OWNER` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_USER_TAG` (
  `AUT_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) NOT NULL,
  `AUT_TAG` varchar(30) NOT NULL,
  `AUT_INDX` int(11) DEFAULT '0',
  `REG_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`AUT_SEQ`),
  UNIQUE KEY `UK_USR_TAG` (`USR_SEQ`,`AUT_TAG`),
  KEY `IDX_USR_SEQ` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ALUMNI_VERIFICATION` (
  `USR_SEQ` int(11) NOT NULL,
  `STATUS` enum('unsubmitted','pending','rejected','approved','reapproval_pending') NOT NULL,
  `GRADUATION_YEAR` smallint(5) unsigned DEFAULT NULL,
  `COHORT` varchar(20) DEFAULT NULL,
  `DEPARTMENT` varchar(100) DEFAULT NULL,
  `REJECTION_REASON` varchar(500) DEFAULT NULL,
  `SUBMITTED_AT` datetime DEFAULT NULL,
  `REVIEWED_AT` datetime DEFAULT NULL,
  `REVIEWED_BY` int(11) DEFAULT NULL,
  `APPROVED_GRADUATION_YEAR` smallint(5) unsigned DEFAULT NULL,
  `APPROVED_COHORT` varchar(20) DEFAULT NULL,
  `APPROVED_DEPARTMENT` varchar(100) DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`USR_SEQ`),
  KEY `IDX_AV_STATUS_UPDATED` (`STATUS`,`UPDATED_AT`),
  KEY `IDX_AV_REVIEWER` (`REVIEWED_BY`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_ACCOUNT_STATE` (
  `ACCOUNT_ID` int(11) NOT NULL,
  `STATUS` enum('ACTIVE','SUSPENDED','WITHDRAWN') NOT NULL,
  `SUSPENDED_AT` datetime DEFAULT NULL,
  `WITHDRAWN_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`ACCOUNT_ID`),
  KEY `IDX_AUTH_ACCOUNT_STATE_STATUS` (`STATUS`,`UPDATED_AT`),
  CONSTRAINT `FK_AUTH_ACCOUNT_STATE_ACCOUNT` FOREIGN KEY (`ACCOUNT_ID`) REFERENCES `WEO_MEMBER` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_CONSENT` (
  `CONSENT_ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `ACCOUNT_ID` int(11) NOT NULL,
  `CONSENT_TYPE` enum('TERMS','PRIVACY','MARKETING') NOT NULL,
  `CONSENT_VERSION` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `IS_REQUIRED` tinyint(1) NOT NULL,
  `IS_ACCEPTED` tinyint(1) NOT NULL,
  `ACCEPTED_AT` datetime DEFAULT NULL,
  `WITHDRAWN_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`CONSENT_ID`),
  UNIQUE KEY `UQ_AUTH_CONSENT_ACCOUNT_TYPE_VERSION` (`ACCOUNT_ID`,`CONSENT_TYPE`,`CONSENT_VERSION`),
  KEY `IDX_AUTH_CONSENT_ACCOUNT_CREATED` (`ACCOUNT_ID`,`CREATED_AT`),
  CONSTRAINT `FK_AUTH_CONSENT_ACCOUNT` FOREIGN KEY (`ACCOUNT_ID`) REFERENCES `WEO_MEMBER` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_EMAIL_VERIFICATION` (
  `TOKEN_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `NORMALIZED_EMAIL` varchar(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STATUS` enum('READY','CONSUMED','CANCELLED','EXPIRED') NOT NULL,
  `ATTEMPT_COUNT` int(10) unsigned NOT NULL DEFAULT '0',
  `EXPIRES_AT` datetime NOT NULL,
  `CONSUMED_AT` datetime DEFAULT NULL,
  `CANCELLED_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`TOKEN_HASH`),
  KEY `IDX_AUTH_EMAIL_VERIFICATION_EMAIL_STATUS` (`NORMALIZED_EMAIL`,`STATUS`,`EXPIRES_AT`),
  KEY `IDX_AUTH_EMAIL_VERIFICATION_STATUS_EXPIRY` (`STATUS`,`EXPIRES_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_IDENTITY` (
  `IDENTITY_ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `ACCOUNT_ID` int(11) NOT NULL,
  `PROVIDER` enum('EMAIL','KAKAO','APPLE','LOCAL_USERNAME') NOT NULL,
  `SUBJECT_KEY` varchar(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `NORMALIZED_EMAIL` varchar(255) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `STATUS` enum('ACTIVE','DISABLED','REVOKED') NOT NULL,
  `METADATA_TEXT` text,
  `VERIFIED_AT` datetime DEFAULT NULL,
  `LAST_AUTHENTICATED_AT` datetime DEFAULT NULL,
  `REVOKED_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`IDENTITY_ID`),
  UNIQUE KEY `UQ_AUTH_IDENTITY_PROVIDER_SUBJECT` (`PROVIDER`,`SUBJECT_KEY`),
  UNIQUE KEY `UQ_AUTH_IDENTITY_ID_ACCOUNT` (`IDENTITY_ID`,`ACCOUNT_ID`),
  UNIQUE KEY `UQ_AUTH_IDENTITY_ID_PROVIDER` (`IDENTITY_ID`,`PROVIDER`),
  UNIQUE KEY `UQ_AUTH_IDENTITY_ID_ACCOUNT_PROVIDER` (`IDENTITY_ID`,`ACCOUNT_ID`,`PROVIDER`),
  UNIQUE KEY `UQ_AUTH_IDENTITY_NORMALIZED_EMAIL` (`NORMALIZED_EMAIL`),
  KEY `IDX_AUTH_IDENTITY_ACCOUNT_STATUS` (`ACCOUNT_ID`,`STATUS`),
  CONSTRAINT `FK_AUTH_IDENTITY_ACCOUNT` FOREIGN KEY (`ACCOUNT_ID`) REFERENCES `WEO_MEMBER` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_IDENTITY_MIGRATION_JOURNAL` (
  `RUN_ID` char(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STEP_KEY` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STATUS` enum('STARTED','APPLIED','FAILED') NOT NULL,
  `FAILURE_CODE` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `STARTED_AT` datetime NOT NULL,
  `APPLIED_AT` datetime DEFAULT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`RUN_ID`,`STEP_KEY`),
  KEY `IDX_AUTH_IDENTITY_MIGRATION_JOURNAL_STATUS` (`STATUS`,`UPDATED_AT`),
  CONSTRAINT `FK_AUTH_IDENTITY_MIGRATION_JOURNAL_RUN` FOREIGN KEY (`RUN_ID`) REFERENCES `AUTH_IDENTITY_MIGRATION_RUN` (`RUN_ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_IDENTITY_MIGRATION_RUN` (
  `RUN_ID` char(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STATUS` enum('PREFLIGHT_PASSED','APPLYING','APPLIED','FAILED') NOT NULL,
  `SOURCE_FINGERPRINT` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `CONFLICT_COUNT` int(10) unsigned NOT NULL,
  `FAILURE_CODE` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `STARTED_AT` datetime NOT NULL,
  `COMPLETED_AT` datetime DEFAULT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`RUN_ID`),
  KEY `IDX_AUTH_IDENTITY_MIGRATION_RUN_STATUS` (`STATUS`,`UPDATED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_PASSWORD_CREDENTIAL` (
  `IDENTITY_ID` bigint(20) unsigned NOT NULL,
  `PROVIDER` enum('EMAIL','KAKAO','APPLE','LOCAL_USERNAME') NOT NULL,
  `ALGORITHM` varchar(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `PARAMETERS_TEXT` varchar(255) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `PASSWORD_HASH` varchar(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STATUS` enum('ACTIVE','DISABLED') NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`IDENTITY_ID`),
  KEY `FK_AUTH_PASSWORD_CREDENTIAL_IDENTITY` (`IDENTITY_ID`,`PROVIDER`),
  CONSTRAINT `FK_AUTH_PASSWORD_CREDENTIAL_IDENTITY` FOREIGN KEY (`IDENTITY_ID`, `PROVIDER`) REFERENCES `AUTH_IDENTITY` (`IDENTITY_ID`, `PROVIDER`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER TRG_AUTH_PASSWORD_CREDENTIAL_PROVIDER_INSERT
BEFORE INSERT ON AUTH_PASSWORD_CREDENTIAL
FOR EACH ROW
BEGIN
    IF NEW.PROVIDER NOT IN ('EMAIL','LOCAL_USERNAME') THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'password credential requires EMAIL or LOCAL_USERNAME';
    END IF;
END */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER TRG_AUTH_PASSWORD_CREDENTIAL_PROVIDER_UPDATE
BEFORE UPDATE ON AUTH_PASSWORD_CREDENTIAL
FOR EACH ROW
BEGIN
    IF NEW.PROVIDER NOT IN ('EMAIL','LOCAL_USERNAME') THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'password credential requires EMAIL or LOCAL_USERNAME';
    END IF;
END */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_PHONE_CLAIM` (
  `CANONICAL_PHONE` varchar(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `ACCOUNT_ID` int(11) NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`CANONICAL_PHONE`),
  UNIQUE KEY `UQ_AUTH_PHONE_CLAIM_ACCOUNT` (`ACCOUNT_ID`),
  CONSTRAINT `FK_AUTH_PHONE_CLAIM_ACCOUNT` FOREIGN KEY (`ACCOUNT_ID`) REFERENCES `WEO_MEMBER` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_PROVIDER_CREDENTIAL` (
  `CREDENTIAL_ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `IDENTITY_ID` bigint(20) unsigned DEFAULT NULL,
  `CONTINUATION_TOKEN_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `PROVIDER` enum('EMAIL','KAKAO','APPLE','LOCAL_USERNAME') NOT NULL,
  `KEY_ID` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `NONCE_BYTES` varbinary(32) NOT NULL,
  `ALGORITHM` varchar(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `CIPHERTEXT` blob NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  `REVOKED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`CREDENTIAL_ID`),
  UNIQUE KEY `UQ_AUTH_PROVIDER_CREDENTIAL_IDENTITY` (`IDENTITY_ID`),
  UNIQUE KEY `UQ_AUTH_PROVIDER_CREDENTIAL_CONTINUATION` (`CONTINUATION_TOKEN_HASH`),
  UNIQUE KEY `UQ_AUTH_PROVIDER_CREDENTIAL_ID_OWNER` (`CREDENTIAL_ID`,`IDENTITY_ID`,`PROVIDER`),
  KEY `FK_AUTH_PROVIDER_CREDENTIAL_IDENTITY` (`IDENTITY_ID`,`PROVIDER`),
  KEY `FK_AUTH_PROVIDER_CREDENTIAL_CONTINUATION` (`CONTINUATION_TOKEN_HASH`,`PROVIDER`),
  CONSTRAINT `FK_AUTH_PROVIDER_CREDENTIAL_CONTINUATION` FOREIGN KEY (`CONTINUATION_TOKEN_HASH`, `PROVIDER`) REFERENCES `AUTH_SIGNUP_CONTINUATION` (`TOKEN_HASH`, `PROVIDER`),
  CONSTRAINT `FK_AUTH_PROVIDER_CREDENTIAL_IDENTITY` FOREIGN KEY (`IDENTITY_ID`, `PROVIDER`) REFERENCES `AUTH_IDENTITY` (`IDENTITY_ID`, `PROVIDER`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER TRG_AUTH_PROVIDER_CREDENTIAL_OWNER_INSERT
BEFORE INSERT ON AUTH_PROVIDER_CREDENTIAL
FOR EACH ROW
BEGIN
    IF NEW.PROVIDER NOT IN ('KAKAO','APPLE') THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider credential requires KAKAO or APPLE';
    END IF;
    IF (NEW.IDENTITY_ID IS NULL AND NEW.CONTINUATION_TOKEN_HASH IS NULL)
       OR (NEW.IDENTITY_ID IS NOT NULL AND NEW.CONTINUATION_TOKEN_HASH IS NOT NULL) THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider credential requires exactly one owner';
    END IF;
    IF OCTET_LENGTH(NEW.NONCE_BYTES) <> 12 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider credential nonce must be 12 bytes';
    END IF;
END */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER TRG_AUTH_PROVIDER_CREDENTIAL_OWNER_UPDATE
BEFORE UPDATE ON AUTH_PROVIDER_CREDENTIAL
FOR EACH ROW
BEGIN
    IF NEW.PROVIDER NOT IN ('KAKAO','APPLE') THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider credential requires KAKAO or APPLE';
    END IF;
    IF (NEW.IDENTITY_ID IS NULL AND NEW.CONTINUATION_TOKEN_HASH IS NULL)
       OR (NEW.IDENTITY_ID IS NOT NULL AND NEW.CONTINUATION_TOKEN_HASH IS NOT NULL) THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider credential requires exactly one owner';
    END IF;
    IF OCTET_LENGTH(NEW.NONCE_BYTES) <> 12 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider credential nonce must be 12 bytes';
    END IF;
END */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_PROVIDER_REVOKE_OUTBOX` (
  `OUTBOX_ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `IDEMPOTENCY_KEY` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `ACCOUNT_ID` int(11) NOT NULL,
  `IDENTITY_ID` bigint(20) unsigned NOT NULL,
  `CREDENTIAL_ID` bigint(20) unsigned DEFAULT NULL,
  `PROVIDER` enum('EMAIL','KAKAO','APPLE','LOCAL_USERNAME') NOT NULL,
  `STATUS` enum('PENDING','PROCESSING','DELIVERED','FAILED') NOT NULL,
  `ATTEMPT_COUNT` int(10) unsigned NOT NULL,
  `NEXT_ATTEMPT_AT` datetime NOT NULL,
  `LOCKED_AT` datetime DEFAULT NULL,
  `DELIVERED_AT` datetime DEFAULT NULL,
  `LAST_ERROR_CODE` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`OUTBOX_ID`),
  UNIQUE KEY `UQ_AUTH_PROVIDER_REVOKE_IDEMPOTENCY` (`IDEMPOTENCY_KEY`),
  KEY `IDX_AUTH_PROVIDER_REVOKE_DELIVERY` (`STATUS`,`NEXT_ATTEMPT_AT`,`OUTBOX_ID`),
  KEY `FK_AUTH_PROVIDER_REVOKE_IDENTITY_ACCOUNT` (`IDENTITY_ID`,`ACCOUNT_ID`,`PROVIDER`),
  KEY `FK_AUTH_PROVIDER_REVOKE_CREDENTIAL` (`CREDENTIAL_ID`,`IDENTITY_ID`,`PROVIDER`),
  CONSTRAINT `FK_AUTH_PROVIDER_REVOKE_CREDENTIAL` FOREIGN KEY (`CREDENTIAL_ID`, `IDENTITY_ID`, `PROVIDER`) REFERENCES `AUTH_PROVIDER_CREDENTIAL` (`CREDENTIAL_ID`, `IDENTITY_ID`, `PROVIDER`),
  CONSTRAINT `FK_AUTH_PROVIDER_REVOKE_IDENTITY_ACCOUNT` FOREIGN KEY (`IDENTITY_ID`, `ACCOUNT_ID`, `PROVIDER`) REFERENCES `AUTH_IDENTITY` (`IDENTITY_ID`, `ACCOUNT_ID`, `PROVIDER`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER TRG_AUTH_PROVIDER_REVOKE_PROVIDER_INSERT
BEFORE INSERT ON AUTH_PROVIDER_REVOKE_OUTBOX
FOR EACH ROW
BEGIN
    IF NEW.PROVIDER NOT IN ('KAKAO','APPLE') THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider revoke requires KAKAO or APPLE';
    END IF;
END */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER TRG_AUTH_PROVIDER_REVOKE_PROVIDER_UPDATE
BEFORE UPDATE ON AUTH_PROVIDER_REVOKE_OUTBOX
FOR EACH ROW
BEGIN
    IF NEW.PROVIDER NOT IN ('KAKAO','APPLE') THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'provider revoke requires KAKAO or APPLE';
    END IF;
END */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_SESSION_FAMILY` (
  `FAMILY_ID` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `ACCOUNT_ID` int(11) NOT NULL,
  `IDENTITY_ID` bigint(20) unsigned NOT NULL,
  `AUTH_METHOD` enum('EMAIL','KAKAO','APPLE','LOCAL_USERNAME') NOT NULL,
  `GENERATION` bigint(20) unsigned NOT NULL,
  `STATUS` enum('ACTIVE','REVOKED','EXPIRED') NOT NULL,
  `EXPIRES_AT` datetime NOT NULL,
  `LAST_ROTATED_AT` datetime DEFAULT NULL,
  `REVOKED_AT` datetime DEFAULT NULL,
  `REVOKE_REASON_CODE` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`FAMILY_ID`),
  KEY `IDX_AUTH_SESSION_FAMILY_ACCOUNT_STATUS` (`ACCOUNT_ID`,`STATUS`),
  KEY `IDX_AUTH_SESSION_FAMILY_IDENTITY_STATUS` (`IDENTITY_ID`,`STATUS`),
  KEY `FK_AUTH_SESSION_FAMILY_IDENTITY_ACCOUNT` (`IDENTITY_ID`,`ACCOUNT_ID`,`AUTH_METHOD`),
  CONSTRAINT `FK_AUTH_SESSION_FAMILY_ACCOUNT` FOREIGN KEY (`ACCOUNT_ID`) REFERENCES `WEO_MEMBER` (`USR_SEQ`),
  CONSTRAINT `FK_AUTH_SESSION_FAMILY_IDENTITY_ACCOUNT` FOREIGN KEY (`IDENTITY_ID`, `ACCOUNT_ID`, `AUTH_METHOD`) REFERENCES `AUTH_IDENTITY` (`IDENTITY_ID`, `ACCOUNT_ID`, `PROVIDER`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `AUTH_SIGNUP_CONTINUATION` (
  `TOKEN_HASH` char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `PROVIDER` enum('EMAIL','KAKAO','APPLE') NOT NULL,
  `SUBJECT_KEY` varchar(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `STATUS` enum('READY','CONSUMED','CANCELLED','EXPIRED') NOT NULL,
  `PREFILL_TEXT` text,
  `EXPIRES_AT` datetime NOT NULL,
  `CONSUMED_AT` datetime DEFAULT NULL,
  `CANCELLED_AT` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`TOKEN_HASH`),
  UNIQUE KEY `UQ_AUTH_SIGNUP_CONTINUATION_TOKEN_PROVIDER` (`TOKEN_HASH`,`PROVIDER`),
  KEY `IDX_AUTH_SIGNUP_CONTINUATION_SUBJECT_STATUS` (`PROVIDER`,`SUBJECT_KEY`,`STATUS`,`EXPIRES_AT`),
  KEY `IDX_AUTH_SIGNUP_CONTINUATION_STATUS_EXPIRY` (`STATUS`,`EXPIRES_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `DONATION_CONFIG` (
  `DC_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `DC_GOAL` bigint(20) NOT NULL DEFAULT '0',
  `DC_MANUAL_ADJ` bigint(20) NOT NULL DEFAULT '0',
  `DC_NOTE` varchar(200) DEFAULT NULL,
  `IS_ACTIVE` enum('Y','N') DEFAULT 'Y',
  `REG_DATE` datetime DEFAULT NULL,
  `REG_OPER` int(11) DEFAULT NULL,
  `DC_OVERWRITE` enum('Y','N') NOT NULL DEFAULT 'N',
  `DC_MANUAL_DONOR_CNT` int(11) NOT NULL DEFAULT '0',
  `DC_TIER_SPROUT_MIN` bigint(20) NOT NULL DEFAULT '1',
  `DC_TIER_SAPLING_MIN` bigint(20) NOT NULL DEFAULT '10000',
  `DC_TIER_TREE_MIN` bigint(20) NOT NULL DEFAULT '50000',
  `DC_TIER_BLOOMING_MIN` bigint(20) NOT NULL DEFAULT '100000',
  `DC_TIER_FRUITING_MIN` bigint(20) NOT NULL DEFAULT '300000',
  `DC_BALANCE_AMOUNT` bigint(20) DEFAULT NULL,
  `DC_BALANCE_AS_OF` date DEFAULT NULL,
  PRIMARY KEY (`DC_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `DONATION_SNAPSHOT` (
  `DS_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `DS_DATE` date NOT NULL,
  `DS_TOTAL` bigint(20) NOT NULL DEFAULT '0',
  `DS_MANUAL_ADJ` bigint(20) NOT NULL DEFAULT '0',
  `DS_DONOR_CNT` int(11) NOT NULL DEFAULT '0',
  `DS_GOAL` bigint(20) NOT NULL DEFAULT '0',
  `REG_DATE` datetime DEFAULT NULL,
  `DS_OVERWRITE` enum('Y','N') NOT NULL DEFAULT 'N',
  PRIMARY KEY (`DS_SEQ`),
  UNIQUE KEY `UK_DATE` (`DS_DATE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `FUNDAMENTAL_MEMBER` (
  `FM_SEQ` int(11) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `USR_SEQ` int(11) DEFAULT NULL COMMENT 'USR_SEQ',
  `FM_NAME` varchar(20) NOT NULL COMMENT '이름',
  `FM_FN` int(2) NOT NULL COMMENT '졸업기수',
  `FM_DEPT` varchar(30) DEFAULT NULL COMMENT '학과',
  `FM_PHONE` varchar(20) DEFAULT NULL COMMENT '휴대폰',
  `FM_SMS` enum('Y','N') DEFAULT 'Y' COMMENT '공개여부',
  `FM_EMAIL` varchar(100) DEFAULT NULL COMMENT '메일',
  `FM_SPAM` enum('Y','N') DEFAULT 'Y' COMMENT '공개여부',
  `FM_COMPANY` varchar(100) DEFAULT NULL COMMENT '회사명',
  `FM_POSITION` varchar(50) DEFAULT NULL COMMENT '직급',
  `FM_BIZ_TEL` varchar(20) DEFAULT NULL COMMENT '회사전화번호',
  `FM_SEX` enum('M','F','N') DEFAULT 'N' COMMENT '성별',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최조등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근수정일자',
  `FM_STATUS` enum('Y','N') DEFAULT 'Y' COMMENT '상태',
  PRIMARY KEY (`FM_SEQ`),
  KEY `INDX_NAME` (`FM_NAME`),
  KEY `INDX_FN` (`FM_FN`),
  KEY `INDX_DEPT` (`FM_DEPT`),
  KEY `INDX_COMPANY` (`FM_COMPANY`),
  KEY `INDX_POSITION` (`FM_POSITION`),
  KEY `IDX_FM_FN` (`FM_FN`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='졸업기수정보';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `FUNDAMENTAL_NUMBERS` (
  `FN` tinyint(2) unsigned NOT NULL,
  `FN_GYEAR` smallint(4) unsigned NOT NULL,
  `FN_DEPT` varchar(2000) NOT NULL COMMENT '학과',
  `REG_DATE` datetime DEFAULT NULL,
  `REG_OPER` int(11) DEFAULT NULL,
  `REG_IPADDR` varchar(20) DEFAULT NULL,
  `EDT_DATE` datetime DEFAULT NULL,
  `EDT_OPER` int(11) DEFAULT NULL,
  `EDT_IPADDR` varchar(20) DEFAULT NULL,
  PRIMARY KEY (`FN`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='기수 및 학과';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `HISTORY_ENTRY` (
  `HE_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `HE_EVENT_DATE` date NOT NULL,
  `HE_TEXT` varchar(500) NOT NULL,
  `HE_SORT_ORDER` smallint(6) NOT NULL DEFAULT '0',
  `REG_DATE` datetime DEFAULT CURRENT_TIMESTAMP,
  `MOD_DATE` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`HE_SEQ`),
  KEY `IDX_DATE` (`HE_EVENT_DATE`,`HE_SORT_ORDER`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `MAIN_AD` (
  `MA_SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `MA_TYPE` char(20) NOT NULL DEFAULT 'MAIN' COMMENT '구분',
  `AD_TIER` enum('PREMIUM','GOLD','NORMAL') DEFAULT 'NORMAL' COMMENT '광고 등급',
  `AD_TITLE_LABEL` varchar(50) DEFAULT '추천 동문 소식' COMMENT '광고 카드 타이틀 라벨',
  `MA_NAME` varchar(50) NOT NULL COMMENT '제품군 SEQ',
  `MA_URL` varchar(100) DEFAULT NULL,
  `MA_IMG` varchar(500) DEFAULT NULL COMMENT '광고 배너 이미지 URL',
  `MA_TARGET` enum('Y','N') DEFAULT 'N' COMMENT '새창여부',
  `OPEN_YN` enum('Y','N') DEFAULT 'Y' COMMENT '상태(Y:활성, N:비활성)',
  `INDX` int(5) DEFAULT '99999' COMMENT '정렬순위',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최초 등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '최초 등록IP',
  `EDT_IPADDR` varchar(15) DEFAULT NULL COMMENT '최근 수정IP',
  `REG_OPER_ID` varchar(20) NOT NULL COMMENT '최초 등록 OPER_ID',
  `EDT_OPER_ID` varchar(20) DEFAULT NULL COMMENT '최근 수정 OPER_ID',
  `VIEW_AREA` varchar(500) NOT NULL DEFAULT '^www^' COMMENT '노출 제휴사',
  `AD_START_DATE` datetime DEFAULT NULL COMMENT 'Publication start (UTC)',
  `AD_END_DATE` datetime DEFAULT NULL COMMENT 'Publication end (UTC)',
  PRIMARY KEY (`MA_SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='AD 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `MAIN_BANNER_AD` (
  `BN_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `BN_NAME` varchar(200) NOT NULL,
  `BN_URL` varchar(500) NOT NULL,
  `OPEN_YN` char(1) NOT NULL DEFAULT 'N',
  `INDX` int(11) NOT NULL DEFAULT '0',
  `BN_START_DATE` datetime DEFAULT NULL,
  `BN_END_DATE` datetime DEFAULT NULL,
  `CREATED_AT` datetime NOT NULL,
  `UPDATED_AT` datetime NOT NULL,
  PRIMARY KEY (`BN_SEQ`),
  KEY `IDX_MBA_OPEN_YN` (`OPEN_YN`),
  KEY `IDX_MBA_OPEN_PERIOD` (`OPEN_YN`,`BN_START_DATE`,`BN_END_DATE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `MAIN_BANNER_AD_IMAGE` (
  `BNI_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `BN_SEQ` int(11) NOT NULL,
  `IMAGE_URL` varchar(500) NOT NULL,
  `SORT_ORDER` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`BNI_SEQ`),
  KEY `IDX_MBAI_BN_SEQ` (`BN_SEQ`),
  CONSTRAINT `FK_MBAI_BN_SEQ` FOREIGN KEY (`BN_SEQ`) REFERENCES `MAIN_BANNER_AD` (`BN_SEQ`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `SUBSCRIPTION` (
  `SUB_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(11) NOT NULL,
  `AMOUNT` int(11) NOT NULL,
  `PAY_TYPE` varchar(10) NOT NULL DEFAULT 'CARD',
  `BILLING_KEY` varchar(64) DEFAULT NULL,
  `CARD_NO` varchar(32) DEFAULT NULL,
  `STATUS` varchar(20) NOT NULL DEFAULT 'active',
  `START_DATE` datetime NOT NULL,
  `NEXT_BILL` datetime NOT NULL,
  `BILL_DAY` tinyint(4) DEFAULT NULL,
  `END_YYYYMM` char(6) DEFAULT NULL,
  `LAST_BILLED_AT` datetime DEFAULT NULL,
  `FAIL_COUNT` int(11) NOT NULL DEFAULT '0',
  `ORDER_SEQ` int(11) DEFAULT NULL,
  `REG_DATE` datetime NOT NULL,
  `EDT_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`SUB_SEQ`),
  KEY `idx_usr_seq` (`USR_SEQ`),
  KEY `idx_bill_day_status` (`BILL_DAY`,`STATUS`),
  KEY `idx_order_seq` (`ORDER_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `USER_SESSION` (
  `SESSION_ID` varchar(64) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `PROVIDER` enum('KAKAO','DIRECT') DEFAULT 'DIRECT',
  `EXPIRES_AT` datetime NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`SESSION_ID`),
  KEY `IDX_USR` (`USR_SEQ`),
  KEY `IDX_EXPIRES` (`EXPIRES_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_AD_COMMENT` (
  `AC_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `MA_SEQ` int(11) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `NICKNAME` varchar(100) NOT NULL,
  `CONTENTS` text NOT NULL,
  `OPEN_YN` enum('Y','N') NOT NULL DEFAULT 'Y',
  `REG_DATE` datetime NOT NULL,
  PRIMARY KEY (`AC_SEQ`),
  KEY `IDX_MA_SEQ` (`MA_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_AD_LIKE` (
  `AL_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `MA_SEQ` int(11) NOT NULL,
  `USR_SEQ` int(11) NOT NULL,
  `OPEN_YN` enum('Y','N') NOT NULL DEFAULT 'Y',
  `REG_DATE` datetime NOT NULL,
  PRIMARY KEY (`AL_SEQ`),
  KEY `IDX_MA_USR` (`MA_SEQ`,`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_AD_LOG` (
  `AL_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `MA_SEQ` int(11) NOT NULL,
  `USR_SEQ` int(11) DEFAULT NULL,
  `AL_TYPE` enum('VIEW','CLICK') NOT NULL,
  `AL_DATE` datetime NOT NULL,
  `AL_IPADDR` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`AL_SEQ`),
  KEY `IDX_MA_SEQ` (`MA_SEQ`),
  KEY `IDX_DATE` (`AL_DATE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_APP_PUSH` (
  `SEQ` int(11) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `APS_SEQ` int(10) DEFAULT NULL COMMENT '기준 KEY',
  `SCA_SEQ` int(10) NOT NULL DEFAULT '0' COMMENT 'Target USR_SEQ or NOTICE(0)',
  `USR_SEQ` int(10) NOT NULL COMMENT 'USR SEQ',
  `APS_TITLE` varchar(200) DEFAULT NULL COMMENT '푸쉬제목',
  `APS_MSG` text COMMENT '푸쉬내용',
  `JOIN_LINK` varchar(30) DEFAULT NULL,
  `JOIN_SEQ` int(11) DEFAULT NULL,
  `OPEN_YN` enum('Y','N','S') DEFAULT 'S' COMMENT 'S:발송,Y:읽음,N:삭제',
  `REGDATE` datetime DEFAULT NULL,
  PRIMARY KEY (`SEQ`),
  KEY `USR_SEQ` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='APP 푸쉬내용';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_APP_PUSHSAND` (
  `APS_SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'APS_SEQ',
  `SCA_SEQ` int(10) NOT NULL DEFAULT '0' COMMENT '발송하는 USR_SEQ or NOTICE(0)',
  `APS_TYPE` varchar(10) DEFAULT NULL,
  `APS_TITLE` varchar(200) DEFAULT NULL COMMENT '푸쉬제목',
  `APS_MSG` text COMMENT '푸쉬내용',
  `APS_CNT` int(6) DEFAULT NULL COMMENT '발송대상수',
  `APS_SAND` int(6) DEFAULT NULL COMMENT '실제발송수',
  `APS_SAND_TYPE` enum('N','R') DEFAULT 'N' COMMENT 'N:실시간, R:예약',
  `OPEN_YN` enum('Y','N') DEFAULT 'Y',
  `APS_DATE` datetime DEFAULT NULL,
  `REGDATE` datetime DEFAULT NULL,
  PRIMARY KEY (`APS_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='APP 푸쉬관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_BANNER_AD_LOG` (
  `BNL_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `BN_SEQ` int(11) NOT NULL,
  `LOG_TYPE` enum('VIEW','CLICK') NOT NULL,
  `CREATED_AT` datetime NOT NULL,
  PRIMARY KEY (`BNL_SEQ`),
  KEY `IDX_WBAL_BN_SEQ` (`BN_SEQ`),
  CONSTRAINT `FK_WBAL_BN_SEQ` FOREIGN KEY (`BN_SEQ`) REFERENCES `MAIN_BANNER_AD` (`BN_SEQ`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_BOARDBBS` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT '고유넘버',
  `GATE` varchar(20) NOT NULL COMMENT 'GATEGORY',
  `JOIN_SEQ` int(10) DEFAULT NULL COMMENT 'JOIN SEQ',
  `P_ID` int(10) DEFAULT '0' COMMENT 'HIDDEN SEQ',
  `B_NO` int(8) NOT NULL DEFAULT '0' COMMENT 'GATE내 넘버',
  `R_NO` int(2) NOT NULL DEFAULT '0' COMMENT '답글횟수',
  `USR_SEQ` int(10) DEFAULT NULL COMMENT 'USR_SEQ',
  `REG_ID` varchar(12) DEFAULT NULL COMMENT '아이디',
  `REG_NAME` varchar(50) NOT NULL COMMENT '성명',
  `REG_EMAIL` varchar(50) DEFAULT NULL COMMENT '이메일',
  `REG_TEL` varchar(20) DEFAULT NULL,
  `OPEN_TYPE` enum('Y','N') DEFAULT 'Y' COMMENT '공개여부',
  `REG_PWD` varchar(50) DEFAULT NULL,
  `REPLY_MAIL` enum('Y','N') DEFAULT 'N' COMMENT '답변메일받기',
  `SUBJECT` varchar(200) NOT NULL COMMENT '제목',
  `CONTENTS` text COMMENT '내용',
  `CONTENTS_MD` mediumtext COMMENT '원본 Markdown 텍스트 (편집용)',
  `CONTENT_FORMAT` enum('LEGACY','MARKDOWN') DEFAULT 'LEGACY' COMMENT '콘텐츠 포맷',
  `CONTENTS_TYPE` enum('E','H','T') DEFAULT 'T' COMMENT '내용형태(E:EDITOR, H:HTML, T:TEXT)',
  `HIT` int(10) DEFAULT '0' COMMENT '조회수',
  `LIKE_CNT` int(10) DEFAULT '0' COMMENT 'LIKE',
  `FILES` varchar(50) DEFAULT NULL COMMENT '첨부파일',
  `THUMBNAIL_URL` varchar(500) DEFAULT NULL COMMENT '대표 이미지 URL',
  `SUMMARY` varchar(200) DEFAULT NULL COMMENT '본문 요약',
  `IS_PINNED` enum('Y','N') DEFAULT 'N' COMMENT '상단 고정 여부',
  `RE_ID` varchar(12) DEFAULT NULL COMMENT '답변등록자ID',
  `RE_NAME` varchar(20) DEFAULT NULL COMMENT '답변등록자명',
  `RE_CONTENTS` text COMMENT '답변',
  `RE_CONTENTS_TYPE` enum('E','H','T') DEFAULT 'T' COMMENT '내용형태(E:EDITOR, H:HTML, T:TEXT)',
  `RE_FILES` varchar(50) DEFAULT NULL COMMENT '첨부파일',
  `RE_DATE` datetime DEFAULT NULL COMMENT '답변등록일자',
  `SDATE` date DEFAULT NULL COMMENT '이벤트 시작일자',
  `EDATE` date DEFAULT NULL COMMENT '이벤트 종료일자',
  `OPEN_YN` enum('Y','N','R','E') NOT NULL DEFAULT 'Y' COMMENT '상태(Y:활성, N:비활성)',
  `STEP` enum('A','U') NOT NULL DEFAULT 'U' COMMENT '단계(A:관리자, N:사용자)',
  `REG_DATE` datetime NOT NULL COMMENT '최초작성일자',
  `EDT_DATE` datetime NOT NULL COMMENT '최근수정일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '수정 및 작성 IP',
  PRIMARY KEY (`SEQ`),
  KEY `IDX_BBS_FEED` (`GATE`,`OPEN_YN`,`SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='통합게시글 TABLE';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_BOARDCOMAND` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT '고유넘버',
  `BC_TYPE` char(1) DEFAULT 'B' COMMENT 'B:게시판',
  `JOIN_SEQ` int(10) NOT NULL COMMENT '게시글 SEQ',
  `REPLY_KEY` varchar(255) NOT NULL DEFAULT '100' COMMENT '댓글 KEY',
  `USR_SEQ` int(10) NOT NULL COMMENT 'USR_SEQ',
  `NICKNAME` varchar(20) NOT NULL COMMENT '작성필명',
  `CONTENTS` text COMMENT '아이디',
  `TYPE_AU` enum('A','U') NOT NULL DEFAULT 'U' COMMENT 'A:공지, U:일반',
  `OPEN_YN` enum('Y','N','B') NOT NULL DEFAULT 'Y' COMMENT '상태(Y:활성, N:비활성, B:신고)',
  `REG_DATE` datetime NOT NULL COMMENT '최초작성일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '수정 및 작성 IP',
  PRIMARY KEY (`SEQ`),
  KEY `IDX_BCOM_JOIN` (`JOIN_SEQ`,`BC_TYPE`,`OPEN_YN`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='게시판 댓글';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_BOARDLIKE` (
  `SEQ` int(11) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `USR_SEQ` int(11) NOT NULL COMMENT 'USR SEQ',
  `BBS_SEQ` int(11) NOT NULL COMMENT 'BBS SEQ',
  `OPEN_YN` enum('Y','N') DEFAULT 'Y' COMMENT '등록여부',
  `REG_DATE` datetime DEFAULT NULL COMMENT '등록일자',
  `DEL_DATE` datetime DEFAULT NULL COMMENT '해제일자',
  PRIMARY KEY (`SEQ`),
  KEY `IDX_BBS_LIKE` (`BBS_SEQ`,`OPEN_YN`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='좋아요';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_BOOKMARK` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `KO_SEQ` int(10) NOT NULL COMMENT 'OPER SEQ',
  `JOBS_SEQ` int(10) NOT NULL COMMENT 'JOBCODE SEQ',
  `REG_DATE` datetime DEFAULT NULL COMMENT '등록일자',
  PRIMARY KEY (`SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='퀵메뉴 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_CODE` (
  `SEQ` int(5) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `CTYPE` char(3) NOT NULL COMMENT '구분',
  `CNAME` varchar(100) DEFAULT NULL COMMENT '코드명',
  `INDX` int(5) DEFAULT '99',
  `OPEN_YN` enum('Y','N') DEFAULT 'N',
  `REG_DATE` datetime DEFAULT NULL,
  `EDT_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='코드 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_CONTENTS` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT '고유넘버',
  `CTYPE` varchar(3) DEFAULT NULL,
  `CNAME` varchar(255) NOT NULL COMMENT '명칭',
  `CONTENTS` longtext COMMENT '내용',
  `CONTENTS_CN` text COMMENT '내용_CN',
  `OPEN_YN` enum('Y','N') NOT NULL DEFAULT 'Y' COMMENT '상태(Y:활성, N:비활성)',
  `REG_DATE` datetime NOT NULL COMMENT '최초작성일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '수정 및 작성 IP',
  PRIMARY KEY (`SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='컨텐츠 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_FILES` (
  `F_SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'F_SEQ',
  `F_GATE` varchar(30) NOT NULL DEFAULT 'PL' COMMENT '구분(P:상품)',
  `F_JOIN_SEQ` int(10) DEFAULT NULL COMMENT 'JOIN SEQ',
  `TYPE_NAME` varchar(255) DEFAULT NULL COMMENT 'FILE VIEW NAME',
  `FILE_NAME` varchar(255) DEFAULT NULL COMMENT 'FILE NAME',
  `FILE_SIZE` varchar(10) DEFAULT NULL COMMENT 'FILE SIZE',
  `FILE_WIDTH` varchar(10) DEFAULT NULL COMMENT '넓이(이미지)',
  `FILE_HEIGHT` varchar(10) DEFAULT NULL COMMENT '높이(이미지)',
  `FILE_PATH` varchar(255) DEFAULT NULL COMMENT '파일경로',
  `FILE_ORG_NAME` varchar(100) DEFAULT NULL COMMENT 'ORG_NAME',
  `OPEN_YN` enum('Y','N') NOT NULL DEFAULT 'Y' COMMENT '상태(Y:활성, N:비활성)',
  `REG_DATE` datetime DEFAULT NULL COMMENT '파일 등록 시각',
  `DOWN_CNT` int(5) DEFAULT '0' COMMENT '다운로드 횟수',
  `INDX` int(4) DEFAULT '9999' COMMENT '노출 순위',
  PRIMARY KEY (`F_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='컨텐츠 파일 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_JOBCODE` (
  `JOBS_SEQ` int(5) NOT NULL AUTO_INCREMENT COMMENT 'JOBS_SEQ',
  `JOBS_NAME` varchar(100) NOT NULL COMMENT '시스템명',
  `JOBS_LINK` varchar(100) DEFAULT NULL COMMENT '링크경로',
  `JOBS_CONTENTS` varchar(200) DEFAULT NULL COMMENT '설명',
  `ADMNAME` varchar(30) DEFAULT NULL COMMENT '담당자',
  `JOBS_HIGH_SEQ` int(5) DEFAULT NULL COMMENT '상위참조 JOBS_SEQ',
  `INDX` int(5) DEFAULT '99',
  `OPEN_YN` enum('Y','N') DEFAULT 'N',
  `REG_DATE` datetime DEFAULT NULL,
  `EDT_DATE` datetime DEFAULT NULL,
  `REG_IPADDR` varchar(15) DEFAULT NULL,
  `EDT_IPADDR` varchar(15) DEFAULT NULL,
  `WP_CHECK` enum('Y','N') DEFAULT 'N' COMMENT '제휴사 메뉴여부',
  PRIMARY KEY (`JOBS_SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='시스템 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_JOBCODE_GRANT` (
  `KO_SEQ` int(5) NOT NULL COMMENT 'KO_SEQ',
  `JOBS_SEQ` int(5) NOT NULL COMMENT 'JOBS_SEQ',
  `LST_JOBS` enum('Y','N') DEFAULT 'N' COMMENT '리스트권한',
  `RED_JOBS` enum('Y','N') DEFAULT 'N' COMMENT '상세보기권한',
  `CRT_JOBS` enum('Y','N') DEFAULT 'N' COMMENT '생성권한',
  `DEL_JOBS` enum('Y','N') DEFAULT 'N' COMMENT '삭제권한',
  `UPD_JOBS` enum('Y','N') DEFAULT 'N' COMMENT '수정권한',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최초 등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '최초 등록 IP',
  `EDT_IPADDR` varchar(15) DEFAULT NULL COMMENT '최근 수정 IP',
  `REG_OPER_ID` varchar(20) NOT NULL COMMENT '최초 등록 OPER_ID',
  `EDT_OPER_ID` varchar(20) DEFAULT NULL COMMENT '최근 수정 OPER_ID',
  PRIMARY KEY (`KO_SEQ`,`JOBS_SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='권한부여';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_MEMBER` (
  `USR_SEQ` int(5) NOT NULL AUTO_INCREMENT COMMENT 'USR_SEQ',
  `USR_ID` varchar(20) NOT NULL COMMENT '사용자 ID',
  `USR_NAME` varchar(100) NOT NULL COMMENT '사용자 명',
  `USR_NICK` varchar(20) DEFAULT NULL COMMENT '닉네임',
  `USR_PWD` varchar(50) NOT NULL COMMENT '비밀번호',
  `USR_PWD_CNT` int(1) DEFAULT '0' COMMENT '비밀번호 오류횟수',
  `USR_STATUS` char(3) DEFAULT 'BBB' COMMENT '상태',
  `USR_DEPT` varchar(30) DEFAULT NULL COMMENT '학과',
  `USR_FN` int(2) DEFAULT NULL COMMENT '기수',
  `USR_PUSHID` varchar(100) DEFAULT NULL COMMENT '푸쉬ID(채팅앱)',
  `USR_THUMNAIL` varchar(300) DEFAULT NULL COMMENT '썸네일',
  `USR_GATE_IN` char(1) DEFAULT NULL COMMENT '경로(N:네이버, K:카카오, F:페이스북, G:구글, D:직접)',
  `USR_SOCIAL_ID` varchar(30) DEFAULT NULL COMMENT '소셜 아이디',
  `USR_YMD` varchar(15) DEFAULT NULL COMMENT '생년월일',
  `USR_PHONE` varchar(20) DEFAULT NULL COMMENT '휴대폰번호',
  `USR_SMS` enum('Y','N') DEFAULT 'Y' COMMENT '문자메일링',
  `USR_EMAIL` varchar(200) DEFAULT NULL COMMENT '메일주소',
  `USR_MAILING` enum('Y','N') DEFAULT 'Y' COMMENT '메일링',
  `AGREE_PERIOD` varchar(2) DEFAULT '99' COMMENT '개인정보 보관 기간',
  `USR_COMPANY` varchar(100) DEFAULT NULL COMMENT '직장명',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최초 등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '최초 등록IP',
  `EDT_IPADDR` varchar(15) DEFAULT NULL COMMENT '최근 수정IP',
  `LAST_LOG_DATE` datetime DEFAULT NULL COMMENT '최근 방문일자',
  `TOTAL_LOG_CNT` int(10) DEFAULT '0' COMMENT '누적 방문수(로그인수)',
  `R_MAKE` varchar(20) DEFAULT 'www' COMMENT '제휴사',
  `USR_PHOTO` varchar(500) DEFAULT NULL,
  `USR_BIZ_NAME` varchar(100) DEFAULT NULL,
  `USR_BIZ_DESC` varchar(200) DEFAULT NULL,
  `USR_BIZ_ADDR` varchar(200) DEFAULT NULL,
  `USR_POSITION` varchar(100) DEFAULT NULL,
  `USR_JOB_CAT` int(11) DEFAULT NULL,
  `USR_PHONE_PUBLIC` enum('Y','N') NOT NULL DEFAULT 'Y',
  `USR_EMAIL_PUBLIC` enum('Y','N') NOT NULL DEFAULT 'Y',
  `USR_BIZ_CARD` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`USR_SEQ`),
  KEY `IDX_USR_STATUS` (`USR_STATUS`),
  KEY `IDX_USR_REG_DATE` (`REG_DATE`),
  KEY `IDX_WEO_MEMBER_PHONE` (`USR_PHONE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='회원정보';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8 */ ;
/*!50003 SET character_set_results = utf8 */ ;
/*!50003 SET collation_connection  = utf8_general_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
/*!50003 CREATE*/ /*!50017 */ /*!50003 TRIGGER trg_erasure_cancel_restore_guard AFTER UPDATE ON WEO_MEMBER FOR EACH ROW
 UPDATE ALUMNI_ERASURE_CANCELLATION c JOIN ALUMNI_ACCOUNT_DELETION_REQUEST d ON d.REQUEST_ID=c.REQUEST_ID
 SET c.RESTORE_BLOCKED=1 WHERE d.USR_SEQ=NEW.USR_SEQ AND d.STATUS='pending' AND OLD.USR_STATUS='AAA' */;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_MEMBER_LOG` (
  `L_SEQ` int(10) NOT NULL AUTO_INCREMENT,
  `USR_SEQ` int(5) NOT NULL COMMENT '사용자 SEQ',
  `LOG_DATE` datetime NOT NULL COMMENT '로그인 시간',
  `REG_DATE` datetime NOT NULL COMMENT '최초 등록일자',
  `REG_GEOCODE` varchar(50) NOT NULL COMMENT '등록 좌표',
  `REG_IPADDR` varchar(15) NOT NULL COMMENT '최초 등록IP',
  `REG_AGENT` varchar(255) DEFAULT NULL,
  `SESSIONID` varchar(40) DEFAULT NULL COMMENT 'SESSION ID',
  PRIMARY KEY (`L_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='사용자 로그정보';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_MEMBER_OUT` (
  `SEQ` int(5) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `USR_SEQ` int(10) NOT NULL COMMENT 'USR_SEQ',
  `REASON` text COMMENT '탈퇴사유',
  `REG_DATE` datetime DEFAULT NULL,
  PRIMARY KEY (`SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='탈퇴 회원 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_MEMBER_PUSH` (
  `PUSH_ID` varchar(300) NOT NULL COMMENT 'PUSH ID',
  `USR_SEQ` int(10) NOT NULL COMMENT 'NSC_MEMBER USR_SEQ',
  `LOG_AGENT` varchar(200) DEFAULT NULL COMMENT 'HTTP_USER_AGENT',
  `REG_DATE` datetime DEFAULT NULL COMMENT '등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '접근일자',
  PRIMARY KEY (`PUSH_ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='push 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_MEMBER_SOCIAL` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `NMS_GATE` char(2) NOT NULL COMMENT 'KT:카카오톡,NV:네이버,FB:페이스북,GG:구글',
  `USR_SEQ` int(10) NOT NULL COMMENT 'NSC_MEMBER USR_SEQ',
  `NMS_ID` varchar(50) NOT NULL COMMENT 'SOCIAL ID',
  `NMS_EMAIL` varchar(255) DEFAULT NULL,
  `NMS_TOKEN` varchar(100) DEFAULT NULL COMMENT 'SOCIAL TOKEN',
  `NMS_THUMNAIL` varchar(100) DEFAULT NULL COMMENT 'THUMNAIL ADDRESS',
  `NMS_STATUS` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `NMS_EMAIL_ENABLED` enum('Y','N') NOT NULL DEFAULT 'Y',
  `REG_DATE` datetime DEFAULT NULL COMMENT '연동일자',
  `OUT_DATE` datetime DEFAULT NULL COMMENT '해제일자',
  PRIMARY KEY (`SEQ`),
  UNIQUE KEY `UK_USR_PROVIDER` (`USR_SEQ`,`NMS_GATE`),
  UNIQUE KEY `UK_PROVIDER_SUBJECT` (`NMS_GATE`,`NMS_ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='SOCIAL CONNECT';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_OPERATOR` (
  `KO_SEQ` int(5) NOT NULL AUTO_INCREMENT COMMENT 'KO_SEQ',
  `OPER_ID` varchar(20) NOT NULL COMMENT '운영자ID',
  `OPER_NAME` varchar(20) NOT NULL COMMENT '운영자명',
  `OPER_PWD` varchar(50) NOT NULL COMMENT '비밀번호',
  `OPERSTATUS_CD` enum('Y','N','I','D') DEFAULT 'I' COMMENT '상태',
  `EMP_PHONE` varchar(20) DEFAULT NULL,
  `EMP_TEL` varchar(20) DEFAULT NULL,
  `EMP_EMAIL` varchar(100) DEFAULT NULL,
  `LOG_TIME` int(2) DEFAULT '99' COMMENT '로그지연시간(99:지속, 분)',
  `MULTI_LOG` enum('Y','N') DEFAULT 'Y' COMMENT '다중형태(Y:다중, N:단일)',
  `REG_DATE` datetime NOT NULL COMMENT '최초 등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `REG_IPADDR` varchar(15) NOT NULL COMMENT '최초 등록IP',
  `EDT_IPADDR` varchar(15) DEFAULT NULL COMMENT '최근 수정IP',
  `REG_OPER_ID` varchar(20) NOT NULL COMMENT '최초 등록 OPER_ID',
  `EDT_OPER_ID` varchar(20) DEFAULT NULL COMMENT '최근 수정 OPER_ID',
  `LAST_LOG_DATE` datetime DEFAULT NULL COMMENT '최근 방문일자',
  `TOTAL_LOG_CNT` int(10) DEFAULT '0' COMMENT '누적 방문수(로그인수)',
  `WP_SEQ` int(10) DEFAULT '0' COMMENT '제휴사 SEQ',
  PRIMARY KEY (`KO_SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='관리자(운영자)회원정보';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_OPERATOR_LOG` (
  `KO_SEQ` int(5) NOT NULL COMMENT '운영자 SEQ',
  `LOG_DATE` datetime NOT NULL COMMENT '로그인 시간',
  `REG_DATE` datetime NOT NULL COMMENT '최초 등록일자',
  `REG_IPADDR` varchar(15) NOT NULL COMMENT '최초 등록IP',
  `REG_AGENT` varchar(255) DEFAULT NULL,
  `SESSIONID` varchar(40) DEFAULT NULL COMMENT 'SESSION ID'
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='운영자 로그정보';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_OPERATOR_PAGE_LOG` (
  `KO_SEQ` int(5) NOT NULL COMMENT '운영자 SEQ',
  `JOBS_SEQ` int(5) NOT NULL COMMENT '로그인 시간',
  `REG_DATE` datetime NOT NULL COMMENT '최초 등록일자',
  `REG_IPADDR` varchar(15) NOT NULL COMMENT '최초 등록IP'
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='운영자 페이지로그정보';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_ORDER` (
  `O_SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'O_SEQ',
  `O_GATE` enum('S','P','F') NOT NULL DEFAULT 'S' COMMENT 'P:정기, S:비정기, F:동행',
  `USR_SEQ` int(10) NOT NULL COMMENT '사용자 SEQ',
  `O_PAY_TYPE` char(4) DEFAULT 'FREE' COMMENT 'CARD,BANK,FREE',
  `O_BANK_ACCOUNT` varchar(50) DEFAULT NULL COMMENT '무통장입금계좌',
  `O_DEPOSIT_NAME` varchar(20) DEFAULT NULL COMMENT '무통장입금자명',
  `O_PAYMENT` enum('Y','N') DEFAULT 'N' COMMENT 'Y:결제, N:미결제',
  `O_REGDATE` datetime DEFAULT NULL COMMENT '신청일자',
  `O_PAYDATE` datetime DEFAULT NULL COMMENT '결제일자',
  `O_CANCEL_DATE` datetime DEFAULT NULL COMMENT '취소일자',
  `O_STATUS` char(1) DEFAULT 'I' COMMENT 'Y:이용중, N:삭제, C:취소, M:이동, R:환불, I:이용전, E:종료',
  `O_PRICE` int(10) DEFAULT '0' COMMENT '정상금액',
  `O_PAY` int(10) DEFAULT '0' COMMENT '결제금액',
  `O_MEMO` text COMMENT '메모',
  `O_TYPE` enum('A','B','C') NOT NULL DEFAULT 'C' COMMENT 'A:후원금, B:해피나눔, C:기타',
  `P_SEQ` int(11) DEFAULT NULL COMMENT '동행 seq',
  `O_RECEIPT` enum('Y','N') NOT NULL DEFAULT 'N' COMMENT '영수증(Y:발급, N:미발급)',
  `O_PG_SEQ` int(11) DEFAULT NULL COMMENT 'PG SEQ',
  `REG_OPER` int(10) DEFAULT NULL COMMENT '최초 등록 SEQ',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최초 등록일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '최초 등록IP',
  `EDT_OPER` int(10) DEFAULT NULL COMMENT '최근 수정 운영자 SEQ',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `EDT_IPADDR` varchar(15) DEFAULT NULL COMMENT '최근 수정IP',
  `O_SOURCE` varchar(30) NOT NULL DEFAULT 'other',
  `O_TRANSACTION_NO` varchar(191) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `O_COMPOSITE_KEY` char(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `O_DONATION_DATE` date DEFAULT NULL,
  `O_GROSS_AMOUNT` bigint(20) unsigned DEFAULT NULL,
  `O_REFUNDED_AMOUNT` bigint(20) unsigned NOT NULL DEFAULT '0',
  `O_NET_RECEIVED_AMOUNT` bigint(20) unsigned DEFAULT NULL,
  `O_LIFECYCLE_STATUS` varchar(30) NOT NULL DEFAULT 'pending',
  `O_PAYMENT_METHOD` varchar(30) DEFAULT NULL,
  `O_ACCOUNT_USR_SEQ` int(11) DEFAULT NULL,
  `O_DONOR_NAME` varchar(100) DEFAULT NULL,
  `O_DONOR_PHONE` varchar(32) DEFAULT NULL,
  `O_DONOR_COHORT` varchar(20) DEFAULT NULL,
  `O_DONOR_DEPARTMENT` varchar(100) DEFAULT NULL,
  `O_LEGAL_RETENTION_UNTIL` datetime DEFAULT NULL,
  `O_ACCOUNT_UNLINKED_AT` datetime DEFAULT NULL,
  PRIMARY KEY (`O_SEQ`),
  UNIQUE KEY `UK_WO_SOURCE_TRANSACTION` (`O_SOURCE`,`O_TRANSACTION_NO`),
  UNIQUE KEY `UK_WO_COMPOSITE_KEY` (`O_COMPOSITE_KEY`),
  KEY `USR_SEQ` (`USR_SEQ`),
  KEY `IDX_WO_LIFECYCLE_DATE` (`O_LIFECYCLE_STATUS`,`O_DONATION_DATE`,`O_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='주문통합관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_ORDER_PROFILE` (
  `OP_SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'OP_SEQ',
  `USR_SEQ` int(10) NOT NULL COMMENT '사용자 SEQ',
  `P_LIMIT` varchar(6) DEFAULT '999999' COMMENT '만료년월(YYYYMM)',
  `P_SEQ` int(10) DEFAULT NULL COMMENT '동행 SEQ',
  `OP_AMOUNT` int(11) DEFAULT '0' COMMENT '금액',
  `OP_DAY` char(2) DEFAULT NULL COMMENT '일자',
  `OP_CARDNO` varchar(30) DEFAULT NULL COMMENT '카드키',
  `OP_STATUS` enum('Y','N') DEFAULT 'Y' COMMENT 'Y:이용중, N:취소',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최초 등록일자',
  `REG_IPADDR` varchar(15) DEFAULT NULL COMMENT '최초 등록IP',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `EDT_IPADDR` varchar(15) DEFAULT NULL COMMENT '최근 수정IP',
  PRIMARY KEY (`OP_SEQ`),
  KEY `USR_SEQ` (`USR_SEQ`),
  KEY `OP_DAY` (`OP_DAY`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='정기결제관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_PG_DATA` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `CNO` varchar(30) DEFAULT NULL COMMENT '거래번호',
  `RES_CD` varchar(10) NOT NULL COMMENT '결과 코드',
  `RES_MSG` varchar(100) NOT NULL COMMENT '결과 메시지',
  `AMOUNT` int(10) DEFAULT NULL COMMENT '결제금액',
  `NUM_CARD` varchar(16) DEFAULT NULL COMMENT '카드번호',
  `NM_BANK` varchar(30) DEFAULT NULL COMMENT '은행명',
  `NUM_BANK` varchar(30) DEFAULT NULL COMMENT '계좌번호',
  `TRAN_DATE` varchar(14) DEFAULT NULL COMMENT '승인일시',
  `AUTH_NO` varchar(10) DEFAULT NULL COMMENT '승인번호',
  `PAY_TYPE` varchar(3) DEFAULT NULL COMMENT '결제타입',
  `O_SEQ` int(10) NOT NULL COMMENT '주문 SEQ',
  PRIMARY KEY (`SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='AllAt PG LOG';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_POPUP` (
  `SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `PU_NAME` varchar(50) NOT NULL COMMENT '명칭',
  `PU_URL` varchar(100) DEFAULT NULL,
  `PU_TARGET` enum('Y','N') DEFAULT 'N' COMMENT '새창여부',
  `PU_SIZE` varchar(100) DEFAULT NULL,
  `PU_GEO` varchar(100) DEFAULT NULL,
  `OPEN_YN` enum('Y','N') DEFAULT 'Y' COMMENT '상태(Y:활성, N:비활성)',
  `INDX` int(5) DEFAULT '99999' COMMENT '정렬순위',
  `REG_DATE` datetime DEFAULT NULL COMMENT '최초 등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '최근 수정일자',
  `VIEW_AREA` varchar(500) DEFAULT '^www^',
  PRIMARY KEY (`SEQ`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='팝업 관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_PUSH_NOTI` (
  `USR_SEQ` int(10) NOT NULL COMMENT 'NSC_MEMBER USR_SEQ',
  `B_CHK` enum('Y','N') DEFAULT 'Y' COMMENT '장학(Y:수신, N:미수신)',
  `E_CHK` enum('Y','N') DEFAULT 'Y' COMMENT '행사(Y:수신, N:미수신)',
  `S_CHK` enum('Y','N') DEFAULT 'Y' COMMENT '동문(Y:수신, N:미수신)',
  `N_CHK` enum('Y','N') DEFAULT 'Y' COMMENT '공지(Y:수신, N:미수신)',
  `R_CHK` enum('Y','N') DEFAULT 'Y' COMMENT '답글(Y:수신, N:미수신)',
  `L_CHK` enum('Y','N') DEFAULT 'Y' COMMENT '좋아요(Y:수신, N:미수신)',
  `REG_DATE` datetime DEFAULT NULL COMMENT '등록일자',
  `EDT_DATE` datetime DEFAULT NULL COMMENT '수정일자',
  PRIMARY KEY (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='push 설정관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_SMS` (
  `SEQ` int(11) NOT NULL AUTO_INCREMENT COMMENT 'SEQ',
  `SCA_SEQ` int(10) NOT NULL COMMENT 'Anal SEQ',
  `USR_SEQ` int(10) NOT NULL COMMENT 'USR SEQ',
  `SMS_TYPE` varchar(10) DEFAULT NULL COMMENT '구분(SYS_CONFIG참조)',
  `SMS_MSG` varchar(2000) DEFAULT NULL COMMENT '문자내용',
  `SMS_PHONE` varchar(20) DEFAULT NULL COMMENT '수신연락처',
  `REGDATE` datetime DEFAULT NULL,
  `SCS_SEQ` int(10) DEFAULT NULL COMMENT '기준 KEY',
  `SMS_RESULT_KEY` int(11) DEFAULT NULL COMMENT 'SMS HISTORY KEY',
  PRIMARY KEY (`SEQ`),
  KEY `USR_SEQ` (`USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='수신문자';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_SMSSAND` (
  `SCS_SEQ` int(10) NOT NULL AUTO_INCREMENT COMMENT 'SCS_SEQ',
  `SCA_SEQ` varchar(10) NOT NULL COMMENT 'Anal SEQ or ALL',
  `SCS_JOIN_VALUE` varchar(10) DEFAULT NULL COMMENT '연결 SEQ or ALL',
  `SCS_TYPE` varchar(10) DEFAULT NULL COMMENT '구분(SYS_CONFIG참조)',
  `SCS_MSG` varchar(2000) DEFAULT NULL COMMENT '문자내용',
  `SCS_CNT` int(6) DEFAULT NULL COMMENT '발송대상수',
  `SCS_SAND` int(6) DEFAULT NULL COMMENT '실제발송수',
  `SCS_SAND_TYPE` enum('N','R') DEFAULT 'N' COMMENT 'N:실시간, R:예약',
  `SCS_DATE` datetime DEFAULT NULL,
  `REGDATE` datetime DEFAULT NULL,
  `CALLBACK` varchar(15) DEFAULT '16449273' COMMENT '발신번호',
  `SCS_MAKE` varchar(20) DEFAULT 'www' COMMENT '제휴사',
  `SCS_ACODE` varchar(20) DEFAULT NULL COMMENT '(구)애널코드',
  `SCS_PTYPE` varchar(20) DEFAULT NULL COMMENT '(구)문자구분',
  `SCS_IDX` int(10) DEFAULT NULL COMMENT '(구)문자 KEY',
  PRIMARY KEY (`SCS_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='발송문자관리';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_VISIT_DAILY` (
  `VD_DATE` date NOT NULL,
  `VD_VISITOR_ID` char(36) NOT NULL,
  `VD_USR_SEQ` int(11) NOT NULL DEFAULT '0',
  `VD_FIRST_TS` datetime NOT NULL,
  `VD_LAST_TS` datetime NOT NULL,
  `VD_HITS` int(10) unsigned NOT NULL DEFAULT '1',
  `VD_UA_HASH` char(16) DEFAULT NULL,
  `VD_IP_HASH` char(16) DEFAULT NULL,
  PRIMARY KEY (`VD_DATE`,`VD_VISITOR_ID`),
  KEY `IX_VD_DATE_USR` (`VD_DATE`,`VD_USR_SEQ`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_VISIT_LOG` (
  `VL_KEY` varchar(50) NOT NULL COMMENT '접속코드',
  `VL_IP` varchar(15) NOT NULL COMMENT '접속IP',
  `VL_DATE` date NOT NULL COMMENT '접속일자',
  `VL_TIME` time NOT NULL COMMENT '접속시간',
  `VL_PAGE` varchar(200) NOT NULL COMMENT '접속페이지',
  `VL_PARAM` varchar(200) NOT NULL COMMENT '접속파라미터',
  `VL_REFERER` tinytext COMMENT '접속정보',
  `VL_AGENT` varchar(200) NOT NULL COMMENT 'AGENT 정보',
  `VL_TARGET` varchar(3) NOT NULL COMMENT '접속대상',
  `VL_MAKE` varchar(20) NOT NULL COMMENT '제휴사',
  PRIMARY KEY (`VL_KEY`),
  KEY `VL_DATE` (`VL_DATE`),
  KEY `VL_IP` (`VL_IP`),
  KEY `VL_TARGET` (`VL_TARGET`),
  KEY `VL_MAKE` (`VL_MAKE`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='접속통계(공통)';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_VISIT_LOGIN` (
  `VJ_KEY` varchar(50) NOT NULL COMMENT '접속코드',
  `VJ_DATE` date NOT NULL COMMENT '접속일자',
  `VJ_TIME` time NOT NULL COMMENT '접속시간',
  `VJ_SEQ` int(10) NOT NULL COMMENT '사용자 SEQ',
  `VJ_TARGET` varchar(3) NOT NULL COMMENT '접속대상',
  `VJ_MAKE` varchar(20) NOT NULL COMMENT '제휴사',
  KEY `VJ_KEY` (`VJ_KEY`),
  KEY `VJ_DATE` (`VJ_DATE`),
  KEY `VJ_SEQ` (`VJ_SEQ`),
  KEY `VJ_TARGET` (`VJ_TARGET`),
  KEY `VJ_MAKE` (`VJ_MAKE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='접속통계(로그인)';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_VISIT_PAGE` (
  `VP_KEY` varchar(50) NOT NULL COMMENT '접속코드',
  `VP_DATE` date NOT NULL COMMENT '접속일자',
  `VP_TIME` time NOT NULL COMMENT '접속시간',
  `VP_PAGE` varchar(200) NOT NULL COMMENT '접속페이지',
  `VP_PARAM` varchar(200) NOT NULL COMMENT '접속파라미터',
  `VP_TARGET` varchar(3) NOT NULL COMMENT '접속대상',
  `VP_MAKE` varchar(20) NOT NULL COMMENT '제휴사',
  KEY `VP_KEY` (`VP_KEY`),
  KEY `VP_DATE` (`VP_DATE`),
  KEY `VP_PAGE` (`VP_PAGE`),
  KEY `VP_TARGET` (`VP_TARGET`),
  KEY `VP_MAKE` (`VP_MAKE`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='접속통계(상세)';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_VISIT_SUMMARY` (
  `VS_DATE` date NOT NULL,
  `VS_DAU_TOTAL` int(10) unsigned NOT NULL DEFAULT '0',
  `VS_DAU_MEMBER` int(10) unsigned NOT NULL DEFAULT '0',
  `VS_DAU_ANON` int(10) unsigned NOT NULL DEFAULT '0',
  `VS_MAU_TOTAL` int(10) unsigned NOT NULL DEFAULT '0',
  `VS_PAGEVIEWS` int(10) unsigned NOT NULL DEFAULT '0',
  `REG_DATE` datetime NOT NULL,
  PRIMARY KEY (`VS_DATE`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `WEO_VISIT_TOT` (
  `VT_DATE` date NOT NULL COMMENT '접속일자',
  `VT_UV_CNT` int(10) NOT NULL DEFAULT '0',
  `VT_PV_CNT` int(10) NOT NULL DEFAULT '0',
  `VT_TARGET` varchar(3) NOT NULL COMMENT '접속대상',
  `VT_MAKE` varchar(20) NOT NULL COMMENT '제휴사',
  PRIMARY KEY (`VT_DATE`,`VT_TARGET`,`VT_MAKE`),
  KEY `VT_DATE` (`VT_DATE`),
  KEY `VT_TARGET` (`VT_TARGET`),
  KEY `VT_MAKE` (`VT_MAKE`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT='접속통계(요약)';
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `_migration_history` (
  `filename` varchar(255) CHARACTER SET ascii NOT NULL,
  `sha256` char(64) CHARACTER SET ascii NOT NULL,
  `applied_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`filename`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `_migration_journal` (
  `filename` varchar(255) CHARACTER SET ascii NOT NULL,
  `sha256` char(64) CHARACTER SET ascii NOT NULL,
  `state` varchar(16) CHARACTER SET ascii NOT NULL,
  `started_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`filename`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `_migration_runner_lock` (
  `lock_name` varchar(32) CHARACTER SET ascii NOT NULL,
  `acquired_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`lock_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_client_builds` (
  `PLATFORM` varchar(16) NOT NULL,
  `BUILD` bigint(20) NOT NULL,
  `VERSION_NAME` varchar(64) NOT NULL DEFAULT '',
  `FIRST_SEEN_AT` datetime NOT NULL,
  `LAST_SEEN_AT` datetime NOT NULL,
  PRIMARY KEY (`PLATFORM`,`BUILD`),
  KEY `IDX_ACB_PLATFORM_LAST_SEEN` (`PLATFORM`,`LAST_SEEN_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_settings` (
  `AS_KEY` varchar(100) NOT NULL,
  `AS_VALUE` text NOT NULL,
  `AS_DESCRIPTION` varchar(500) NOT NULL DEFAULT '',
  `AS_PUBLIC` char(1) NOT NULL DEFAULT 'N',
  `UPDATED_AT` datetime NOT NULL,
  `UPDATED_BY` int(11) DEFAULT NULL,
  PRIMARY KEY (`AS_KEY`),
  KEY `IDX_APP_SETTINGS_PUBLIC` (`AS_PUBLIC`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_update_policy_history` (
  `AUPH_SEQ` int(11) NOT NULL AUTO_INCREMENT,
  `PLATFORM` varchar(16) NOT NULL,
  `BEFORE_JSON` text NOT NULL,
  `AFTER_JSON` text NOT NULL,
  `CHANGED_BY` int(11) DEFAULT NULL,
  `CHANGED_AT` datetime NOT NULL,
  PRIMARY KEY (`AUPH_SEQ`),
  KEY `IDX_AUPH_PLATFORM_CHANGED` (`PLATFORM`,`CHANGED_AT`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `notification_templates` (
  `NT_KEY` varchar(100) NOT NULL,
  `NT_CHANNEL` varchar(10) NOT NULL,
  `NT_TITLE` varchar(200) NOT NULL DEFAULT '',
  `NT_BODY` text NOT NULL,
  `NT_VERSION` int(11) NOT NULL DEFAULT '1',
  `UPDATED_AT` datetime NOT NULL,
  `UPDATED_BY` int(11) DEFAULT NULL,
  PRIMARY KEY (`NT_KEY`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
