#!/bin/bash

# Generate MariaDB audit logs for testing syslog encryption
# This script connects to the MariaDB container and executes various SQL commands
# to trigger audit log events that will be encrypted by the sidecar

set -e

CONTAINER_NAME="mariadb"
DB_USER="root"
DB_PASSWORD="rootpassword"
DB_NAME="testdb"

echo "🔍 Generating MariaDB audit logs..."
echo "📊 This will create various SQL operations to test the syslog encryptor"
echo

# Function to execute SQL commands
execute_sql() {
    local sql="$1"
    local description="$2"
    
    echo "[$description]"
    echo "SQL: $sql"
    
    docker exec -i "$CONTAINER_NAME" mariadb -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -e "$sql" 2>/dev/null || true
    echo "✅ Executed"
    echo
    
    # Small delay to see individual log entries
    sleep 1
}

# Wait for MariaDB to be ready
echo "⏳ Waiting for MariaDB to be ready..."
while ! docker exec "$CONTAINER_NAME" mariadb -u"$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1;" 2>/dev/null; do
    sleep 2
done
echo "✅ MariaDB is ready"
echo

# Create test table and data
execute_sql "CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);" "Creating users table"

execute_sql "CREATE TABLE IF NOT EXISTS audit_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    action VARCHAR(100),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);" "Creating audit_test table"

# Generate various types of audit events
echo "🎯 Generating different types of audit events..."
echo

# INSERT operations
execute_sql "INSERT INTO users (username, email) VALUES ('alice', 'alice@example.com');" "INSERT user alice"
execute_sql "INSERT INTO users (username, email) VALUES ('bob', 'bob@example.com');" "INSERT user bob"
execute_sql "INSERT INTO users (username, email) VALUES ('charlie', 'charlie@example.com');" "INSERT user charlie"

# SELECT operations
execute_sql "SELECT * FROM users;" "SELECT all users"
execute_sql "SELECT username, email FROM users WHERE id = 1;" "SELECT specific user"
execute_sql "SELECT COUNT(*) FROM users;" "COUNT users"

# UPDATE operations
execute_sql "UPDATE users SET email = 'alice.updated@example.com' WHERE username = 'alice';" "UPDATE user email"
execute_sql "UPDATE users SET username = 'bob_updated' WHERE id = 2;" "UPDATE username"

# Complex queries
execute_sql "SELECT u.username, u.email, u.created_at 
FROM users u 
WHERE u.created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR) 
ORDER BY u.created_at DESC;" "Complex SELECT with JOIN-like operation"

# INSERT with subquery
execute_sql "INSERT INTO audit_test (action) 
SELECT CONCAT('User count: ', COUNT(*)) FROM users;" "INSERT with subquery"

# Transaction operations
execute_sql "START TRANSACTION; 
INSERT INTO users (username, email) VALUES ('transaction_user', 'tx@example.com');
INSERT INTO audit_test (action) VALUES ('Transaction test');
COMMIT;" "Transaction operations"

# Administrative operations
execute_sql "SHOW TABLES;" "SHOW TABLES command"
execute_sql "DESCRIBE users;" "DESCRIBE table"
execute_sql "SHOW PROCESSLIST;" "SHOW PROCESSLIST"

# DELETE operations
execute_sql "DELETE FROM users WHERE username = 'transaction_user';" "DELETE user"

# Failed operations (will generate error audit logs)
echo "🚨 Generating error audit logs..."
execute_sql "SELECT * FROM nonexistent_table;" "SELECT from non-existent table (will fail)"
execute_sql "INSERT INTO users (nonexistent_column) VALUES ('test');" "INSERT invalid column (will fail)"

# Some sensitive-looking operations for testing
execute_sql "SELECT username, email FROM users WHERE username LIKE '%admin%';" "Search for admin users"
execute_sql "INSERT INTO audit_test (action) VALUES ('Sensitive operation: password reset');" "Log sensitive operation"
execute_sql "SELECT * FROM users WHERE email LIKE '%@admin.%';" "Search for admin emails"

echo
echo "🎉 Audit log generation complete!"
echo "📝 Check the syslog-encryptor output for encrypted audit logs"
echo "🔐 Each SQL operation should generate encrypted JSON entries"
echo
echo "💡 To see the encrypted output in real-time:"
echo "   docker logs -f syslog-encryptor"