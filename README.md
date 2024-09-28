# MySQL Automatic Backup to S3-Compatible Storage

This project is a Golang-based solution that automatically backs up all MySQL databases and uploads the backup to an S3-compatible storage service. It logs the success or failure of both the upload and the deletion of previous backups. A local log file keeps track of all upload and deletion events for easy monitoring and debugging.

## Features

- **Automatic MySQL Database Backup**: Backups all databases in MySQL using `mysqldump`.
- **S3-Compatible Upload**: Uploads the backup file to an S3 bucket (AWS S3, DigitalOcean Spaces, Wasabi, etc.).
- **Automatic Deletion of Previous Backups**: Deletes the previous backup from the S3 bucket after a new upload.
- **Detailed Logging**: Logs the success or failure of the backup upload and previous backup deletion to a local log file (`backup_history.log`).

## Prerequisites

Before running this project, ensure the following tools are installed:

- **Golang** (version 1.22.2+)
- **MySQL** and **mysqldump**
- An **S3-compatible** storage service (such as AWS S3, DigitalOcean Spaces, Wasabi)

## Installation

1. **Clone the repository**:

    ```bash
    git clone https://github.com/fahrigunadi/go-mysql-backup-s3.git
    cd go-mysql-backup-s3
    ```

2. **Install Go dependencies**:

    Run the following command to install all necessary Go modules:

    ```bash
    go mod tidy
    ```

3. **Copy `.env.example` file**:

    A `.env.example` file is provided in the project. Copy it to `.env` and adjust the configuration to match your setup.

    ```bash
    cp .env.example .env
    ```

    Edit the `.env` file with your MySQL and S3 configuration:

    ```env
    # MySQL Configuration
    DB_HOST=localhost
    DB_PORT=3306
    DB_USER=
    DB_PASSWORD=
    
    # Backup Prefix Configuration
    BACKUP_FOLDER_PREFIX=sql_
    BACKUP_FILE_PREFIX=sql_backups_
    
    # S3 Configuration
    S3_ACCESS_KEY=your_s3_access_key
    S3_SECRET_KEY=your_s3_secret_key
    S3_REGION=your_s3_region
    S3_BUCKET_NAME=your_s3_bucket_name
    S3_ENDPOINT=https://s3.amazonaws.com
    ```

    Replace the values with your actual MySQL credentials and S3 credentials.

## Usage

### Run the Backup

After setting up the `.env` file, run the backup script using the Go command:

```bash
go run main.go
```

This command will:

1. **Backup Process**:
   - The script will create a backup of all your MySQL databases using `mysqldump` and save the resulting `.sql` file in a folder specified by the `BACKUP_FOLDER_PREFIX` in your `.env` file.
   - The backup file will be named based on the `BACKUP_FILE_PREFIX` and the current timestamp, ensuring each backup has a unique filename (e.g., `db_backup_20240928_140501.sql`).

2. **Upload Process**:
   - After creating the backup, the script will automatically upload the `.sql` backup file to the S3 bucket specified in the `.env` file.
   - If the upload is successful, this event is logged in the `backup_history.log` file.

3. **Deletion of Previous Backups**:
   - After uploading the new backup, the script will check if a previous backup exists in the S3 bucket (based on the previous backup log) and delete it.
   - This ensures that only the most recent backup is kept in the S3 bucket, saving storage space and avoiding unnecessary accumulation of backups.
   - The success or failure of the deletion will also be logged in the `backup_history.log` file.

4. **Logging**:
   - The script will log both successful and failed events, including upload attempts and deletion attempts, in the `backup_history.log` file. Each entry will include a timestamp for easy tracking.

### Example of `backup_history.log`
```log
Successfully uploaded backup: db_backup_20240928_140501.sql at 2024-09-28T14:05:01Z
Successfully deleted previous backup: db_backup_20240927_135301.sql at 2024-09-28T14:05:30Z
Failed to upload backup: db_backup_20240928_145001.sql, error: connection timeout at 2024-09-28T14:50:10Z
```

### Automate the Backup

You can schedule this script to run automatically at regular intervals using cron jobs (on Linux) or Task Scheduler (on Windows).

#### Automating with Cron (Linux/Mac)

To run the backup script daily at 2 AM using cron:

1. Open your crontab:

    ```sh
    crontab -e
    ```

2. Add the following line to schedule the script to run daily at 2 AM:

    ```
    0 2 * * * /usr/local/go/bin/go run /path/to/your/project/main.go
    ```

   Make sure to replace `/path/to/your/project` with the actual path to your Go project directory.

#### Automating with Task Scheduler (Windows)

For Windows users, you can automate the backup process using Task Scheduler:

1. Open Task Scheduler and create a new task.
2. In the "Actions" tab, set the action to "Start a program."
3. Set the program/script to `go` and in the "Add arguments" field, enter:

    ```
    run main.go
    ```

4. Set the schedule for when you want the backup to run (e.g., daily at 2 AM).

## Logging Details

The logging system records both success and failure events for the following:

1. **Backup Uploads**:
   - Logs whether the upload was successful or failed, along with error messages for failures.
   
2. **Backup Deletions**:
   - Logs whether the deletion of the previous backup from S3 was successful or failed, with error messages if any.

All logs are stored in `backup_history.log` with timestamps to track when each event occurred.

