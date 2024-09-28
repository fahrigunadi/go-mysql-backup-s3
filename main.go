package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
)

const backupHistoryLogFile = "backup_history.log"

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	s3AccessKeyID := os.Getenv("S3_ACCESS_KEY")
	s3SecretAccessKey := os.Getenv("S3_SECRET_KEY")
	s3Region := os.Getenv("S3_REGION")
	s3BucketName := os.Getenv("S3_BUCKET_NAME")
	s3Endpoint := os.Getenv("S3_ENDPOINT")

	backupFolderPrefix := os.Getenv("BACKUP_FOLDER_PREFIX")
	backupFilePrefix := os.Getenv("BACKUP_FILE_PREFIX")

	backupFolder := fmt.Sprintf("%sbackup", backupFolderPrefix)
	err = createBackupFolder(backupFolder)
	if err != nil {
		log.Fatalf("Error creating backup folder: %v", err)
	}

	backupFileName := filepath.Join(backupFolder, fmt.Sprintf("%s%s.sql", backupFilePrefix, time.Now().Format("20060102_150405")))

	// Backup all MySQL databases using mysqldump
	err = backupMySQL(dbUser, dbPassword, dbHost, dbPort, backupFileName)
	if err != nil {
		log.Fatalf("Error backing up MySQL: %v", err)
	}

	previousBackupFile := readPreviousBackupLog()
	err = uploadToS3(s3AccessKeyID, s3SecretAccessKey, s3Region, s3BucketName, s3Endpoint, backupFileName)
	if err != nil {
		log.Printf("Error uploading to S3: %v", err)
		appendToBackupLog(fmt.Sprintf("Failed to upload backup: %s, error: %v", backupFileName, err))
	} else {
		appendToBackupLog(fmt.Sprintf("Successfully uploaded backup: %s", backupFileName))
	}

	// Delete the previous backup file in S3, if it exists
	if previousBackupFile != "" {
		err = deleteFromS3(s3AccessKeyID, s3SecretAccessKey, s3Region, s3BucketName, s3Endpoint, previousBackupFile)
		if err != nil {
			log.Printf("Error deleting previous backup file from S3: %v", err)
			appendToBackupLog(fmt.Sprintf("Failed to delete previous backup: %s, error: %v", previousBackupFile, err))
		} else {
			appendToBackupLog(fmt.Sprintf("Successfully deleted previous backup: %s", previousBackupFile))
		}
	}

	// Save the name of the current backup file to the log for future deletion
	saveCurrentBackupLog(filepath.Base(backupFileName))

	// Delete the local backup file after upload
	err = deleteLocalBackup(backupFileName)
	if err != nil {
		log.Fatalf("Error deleting local backup file: %v", err)
	}

	log.Println("Backup and upload completed successfully, and the local backup file was deleted.")
}

// Function to create the backup folder if it doesn't exist
func createBackupFolder(folderName string) error {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		err := os.Mkdir(folderName, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create folder: %w", err)
		}
		log.Printf("Backup folder created: %s\n", folderName)
	}
	return nil
}

// Function to backup MySQL databases using mysqldump
func backupMySQL(user, password, host, port, backupFileName string) error {
	cmd := exec.Command("mysqldump", "--all-databases", "-u"+user, "-p"+password, "-h"+host, "-P"+port, "-r", backupFileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Function to upload the backup file to S3
func uploadToS3(accessKey, secretKey, region, bucketName, endpoint, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Configure AWS S3 session
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
	}

	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	s3Client := s3.New(sess)

	// Read the backup file into a buffer
	buffer := make([]byte, fileStat.Size())
	_, err = file.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Upload the backup file to S3
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filepath.Base(fileName)),
		Body:   bytes.NewReader(buffer),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Function to delete the local backup file after upload
func deleteLocalBackup(fileName string) error {
	err := os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	log.Printf("Successfully deleted local backup file: %s\n", fileName)
	return nil
}

// Function to delete the previous backup file from S3
func deleteFromS3(accessKey, secretKey, region, bucketName, endpoint, fileName string) error {
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
	}

	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	s3Client := s3.New(sess)

	// Delete the file from S3
	_, err = s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete S3 file: %w", err)
	}

	log.Printf("Deleted previous backup from S3: %s\n", fileName)
	return nil
}

// Function to save the current backup file name to a log file
func saveCurrentBackupLog(fileName string) {
	err := os.WriteFile("previous_backup.log", []byte(fileName), 0644)
	if err != nil {
		log.Fatalf("Failed to write current backup to log: %v", err)
	}
}

// Function to read the previous backup file name from the log file
func readPreviousBackupLog() string {
	data, err := os.ReadFile("previous_backup.log")
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		log.Fatalf("Failed to read previous backup log: %v", err)
	}
	return string(data)
}

// Function to append the status of upload and deletion to a history log
func appendToBackupLog(message string) {
	f, err := os.OpenFile(backupHistoryLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open backup history log: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s at %s\n", message, time.Now().Format(time.RFC3339)))
	if err != nil {
		log.Fatalf("Failed to write to backup history log: %v", err)
	}
}
