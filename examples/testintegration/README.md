# PostgreSQL Integration and Database Monitor Automation

This directory contains a PostgreSQL integration and a database monitoring automation for SecAuto.

## Files

1. **postgresql_integration.json** - Integration definition file
2. **postgresql_integration.py** - Integration implementation
3. **database_monitor.py** - Automation script that uses the integration
4. **test_config.json** - Example configuration for testing

## Upload and Testing Instructions

### Step 1: Upload the PostgreSQL Integration

Upload the integration using the SecAuto API:

```bash
# Using curl to upload the integration
curl -X POST http://localhost:9090/integrations/upload \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: multipart/form-data" \
  -F "definition=@postgresql_integration.json" \
  -F "script=@postgresql_integration.py"

# Or using the CLI
./secauto integration upload --name postgresql \
  --definition postgresql_integration.json \
  --script postgresql_integration.py
```

The system will:
- Save the integration definition
- Install dependencies (psycopg2-binary)
- Build the integration environment

### Step 2: Configure PostgreSQL Connection

Each client has their own PostgreSQL configuration. Configure the PostgreSQL integration for a specific client:

```bash
# Replace with your actual client ID, API key, and database credentials
curl -X 'POST' \
  'http://localhost:9090/clients/client_e46133b074343c59/integrations/postgresql/config' \
  -H 'accept: */*' \
  -H 'X-API-Key: your-api-key-here' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "postgresql",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "your_database",
      "username": "your_username",
      "password": "your_password",
      "ssl_mode": "prefer"
    }
  }'
```

**Important Notes:**
- Replace `your-api-key-here` with your actual SecAuto API key
- Replace `your_database`, `your_username`, `your_password` with your PostgreSQL credentials
- The server is running on port `9090` (not 8080)
- The integration name is specified in the URL path (`/postgresql/config`) AND in the request body
- This endpoint accepts both POST and PUT methods for create/update operations

### Step 3: Test the Integration

**First, list the client's configured integrations:**

```bash
# List all integrations for a client
curl -X 'GET' \
  'http://localhost:9090/clients/client_e46133b074343c59/integrations' \
  -H 'accept: */*' \
  -H 'X-API-Key: your-api-key-here'
```

**Test the PostgreSQL connection:**

```bash
# Test connection - replace with your API key and client ID
curl -X 'POST' \
  'http://localhost:9090/clients/client_e46133b074343c59/integrations/postgresql/execute' \
  -H 'accept: */*' \
  -H 'X-API-Key: your-api-key-here' \
  -H 'Content-Type: application/json' \
  -d '{"function": "test_connection"}'
```

**List tables for a specific client:**

```bash
curl -X 'POST' \
  'http://localhost:9090/clients/client_e46133b074343c59/integrations/postgresql/execute' \
  -H 'accept: */*' \
  -H 'X-API-Key: secauto-api-key-2024-07-14' \
  -H 'Content-Type: application/json' \
  -d '{"function": "list_tables", "params": {"schema": "public"}}'
```

**List items from a table with client context:**

```bash


```

### Step 4: Upload the Database Monitor Automation

Upload the automation script:

```bash
curl -X POST http://localhost:9090/automations\
  -H "X-API-Key: secauto-api-key-2024-07-14'" \
  -H "Content-Type: multipart/form-data" \
  -F "script=@database_monitor.py"

# Or using CLI
./secauto automation upload --name database_monitor \
  --script database_monitor.py
```

### Step 5: Run the Automation

Execute the automation with client context:

```bash
# Run automation for a specific client
curl -X 'POST' \
  'http://localhost:9090/clients/client_e46133b074343c59/automations/database_monitor/execute' \
  -H 'accept: */*' \
  -H 'X-API-Key: your-api-key-here' \
  -H 'Content-Type: application/json' \
  -d '{
    "config": {
      "integration": "postgresql",
      "monitoring_rules": [
        {
          "name": "Check events table",
          "type": "row_count",
          "table": "events",
          "threshold": 1000,
          "comparison": "greater_than",
          "severity": "medium"
        },
        {
          "name": "Check recent changes",
          "type": "table_changes",
          "table": "logs",
          "time_column": "created_at",
          "minutes": 60,
          "threshold": 100,
          "severity": "high"
        }
      ]
    }
  }'
```

The automation will automatically use the client's PostgreSQL configuration.

### Step 6: Schedule the Automation (Optional)

Create a schedule to run the monitor every hour for a specific client:

```bash
curl -X 'POST' \
  'http://localhost:9090/clients/client_e46133b074343c59/schedules' \
  -H 'accept: */*' \
  -H 'X-API-Key: your-api-key-here' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Hourly Database Monitor",
    "automation": "database_monitor",
    "schedule": "0 * * * *",
    "enabled": true,
    "config": {
      "integration": "postgresql",
      "monitoring_rules": [
        {
          "name": "Check events table",
          "type": "row_count",
          "table": "events",
          "threshold": 1000,
          "comparison": "greater_than",
          "severity": "medium"
        }
      ]
    }
  }'
```

## Testing Without SecAuto Server

You can test the integration locally with client context:

```bash
# Test the integration directly with client ID
echo '{
  "client_id": "test_client_123",
  "function": "test_connection",
  "config": {
    "host": "localhost",
    "database": "testdb",
    "username": "postgres",
    "password": "password"
  }
}' | python3 postgresql_integration.py

# Test the automation with client ID
python3 database_monitor.py '{
  "client_id": "test_client_123",
  "config": {
    "integration": "postgresql"
  }
}'
```

## Integration Functions

The PostgreSQL integration provides these functions:

1. **test_connection()** - Test database connectivity
2. **list_tables(schema)** - List all tables in a schema
3. **list_items(table, limit, offset, order_by, filters)** - Query table data
4. **query(sql, params)** - Execute SELECT queries
5. **get_table_info(table)** - Get table schema information

## Automation Monitoring Rules

The database monitor supports these rule types:

1. **row_count** - Check if table row count exceeds threshold
2. **table_changes** - Monitor recent changes in a table
3. **query_result** - Execute custom query and check results
4. **table_list** - Verify expected tables exist

## Example PostgreSQL Setup

If you need a test PostgreSQL database:

```bash
# Using Docker
docker run --name test-postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=testdb \
  -p 5432:5432 \
  -d postgres:14

# Create test tables
docker exec -it test-postgres psql -U postgres testdb

CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100),
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE logs (
    id SERIAL PRIMARY KEY,
    message TEXT,
    level VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert test data
INSERT INTO events (name) VALUES ('test_event_1'), ('test_event_2');
INSERT INTO users (username, email) VALUES ('user1', 'user1@test.com');
INSERT INTO logs (message, level) VALUES ('System started', 'INFO');
```

## Troubleshooting

1. **psycopg2 installation fails**: Try using `psycopg2-binary` instead
2. **Connection refused**: Check PostgreSQL is running and accessible
3. **Authentication failed**: Verify credentials and pg_hba.conf settings
4. **SSL errors**: Adjust ssl_mode parameter (disable, allow, prefer, require)

## Security Notes

- The integration only allows SELECT queries in the query() function
- Credentials are encrypted when stored in SecAuto
- Use parameterized queries to prevent SQL injection
- Table names are validated before use
- Connection pooling is implemented for efficiency