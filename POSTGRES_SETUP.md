# PostgreSQL Setup Guide for Candlecore

This guide walks you through setting up PostgreSQL for Candlecore's state persistence.

## Prerequisites

✅ PostgreSQL installed locally (you've done this!)

## Step-by-Step Setup

### 1. Start PostgreSQL Service

```powershell
# Check if PostgreSQL is running
Get-Service -Name postgresql*

# If not running, start it
Start-Service postgresql-x64-<version>
```

### 2. Create Database and User

Open PowerShell and connect to PostgreSQL:

```powershell
# Connect as postgres superuser
psql -U postgres

# Or if that doesn't work, try:
& "C:\Program Files\PostgreSQL\<version>\bin\psql.exe" -U postgres
# & "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres
```

In the PostgreSQL prompt, run:

```sql
-- Create a dedicated user for Candlecore
CREATE USER candlecore WITH PASSWORD 'your_secure_password_here';

-- Create the database
CREATE DATABASE candlecore OWNER candlecore;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE candlecore TO candlecore;

-- Exit psql
\q
```

### 3. Run Database Schema

Navigate to your project directory and run the schema:

```powershell
# From candlecore project root
cd c:\Users\CHOCHO\Documents\Workspace\candlecore

# Run the schema file
Get-Content database\schema.sql | psql -U candlecore -d candlecore

# If prompted for password, enter the password you set above
```

**Alternative using psql directly:**

```powershell
psql -U candlecore -d candlecore -f database\schema.sql
```

### 4. Verify Database Setup

Check that tables were created:

```powershell
psql -U candlecore -d candlecore
```

In psql:

```sql
-- List all tables
\dt

-- Should show: accounts, positions, orders, trades

-- Check the initial account
SELECT * FROM accounts;

-- Should show one account with balance 10000.0

-- View the account summary
SELECT * FROM account_summary;

-- Exit
\q
```

### 5. Update Configuration

Edit `config.yaml` to enable database:

```yaml
# Database configuration
database:
  enabled: true # Set to true to use PostgreSQL
  host: localhost
  port: 5432
  user: candlecore
  password: your_secure_password_here
  dbname: candlecore
  sslmode: disable # Use 'require' for production
  account_id: 1 # Which account to use (from accounts table)

# You can still keep file-based backup
state_directory: '.state'
```

**Security Note**: For production, store the password in an environment variable:

```powershell
$env:CANDLECORE_DB_PASSWORD="your_secure_password_here"
```

Then in config.yaml, leave password empty (it will be read from env var).

### 6. Update go.mod and Download Dependencies

```powershell
go mod tidy
```

This will download the PostgreSQL driver (`github.com/lib/pq`).

### 7. Update main.go to Use PostgreSQL Store

Edit `cmd/candlecore/main.go`:

```go
// Import the postgres store
import (
    // ... other imports
    "candlecore/internal/store"
)

func main() {
    // ... existing code ...

    // Initialize state store
    var stateStore engine.StateStore
    var err error

    if cfg.Database.Enabled {
        // Use PostgreSQL store
        log.Info("Using PostgreSQL state store", "host", cfg.Database.Host, "database", cfg.Database.DBName)

        pgStore, err := store.NewPostgresStore(
            cfg.GetDatabaseConnectionString(),
            cfg.Database.AccountID,
        )
        if err != nil {
            log.Error("Failed to initialize PostgreSQL store", "error", err)
            os.Exit(1)
        }
        defer pgStore.Close()

        stateStore = pgStore
    } else {
        // Use file store
        log.Info("Using file-based state store", "directory", cfg.StateDirectory)

        stateStore, err = store.NewFileStore(cfg.StateDirectory)
        if err != nil {
            log.Error("Failed to initialize state store", "error", err)
            os.Exit(1)
        }
    }

    // ... rest of the code remains the same ...
}
```

### 8. Build and Run

```powershell
# Build the application
go build -o candlecore.exe ./cmd/candlecore

# Run it
./candlecore.exe
```

You should see:

```
[timestamp] INFO: Using PostgreSQL state store host=localhost database=candlecore
[timestamp] INFO: Starting Candlecore trading engine
...
```

### 9. Verify Data Persistence

After running, check the database:

```powershell
psql -U candlecore -d candlecore
```

```sql
-- Check account was updated
SELECT * FROM accounts;

-- Check if any positions were created
SELECT * FROM positions;

-- Check trade history
SELECT * FROM trades;

-- View performance summary
SELECT * FROM account_summary;

-- Exit
\q
```

## Database Features

### Tables

- **accounts** - Account balance and equity
- **positions** - Open positions with unrealized P&L
- **orders** - Order history (pending and filled)
- **trades** - Completed trade records

### Views

- **account_summary** - Aggregated account metrics
- **position_summary** - Position details with P&L percentages

### Triggers

- **update_account_equity** - Automatically calculates equity when positions change

## Performance Queries

### Get Trading Performance

```sql
SELECT
    COUNT(*) as total_trades,
    COUNT(*) FILTER (WHERE net_pnl > 0) as winning_trades,
    ROUND(COUNT(*) FILTER (WHERE net_pnl > 0)::DECIMAL / COUNT(*) * 100, 2) as win_rate,
    ROUND(SUM(net_pnl), 2) as total_pnl,
    ROUND(AVG(net_pnl), 2) as avg_pnl,
    ROUND(MAX(net_pnl), 2) as best_trade,
    ROUND(MIN(net_pnl), 2) as worst_trade
FROM trades
WHERE account_id = 1;
```

### Get Trade History by Symbol

```sql
SELECT
    symbol,
    side,
    entry_price,
    exit_price,
    quantity,
    net_pnl,
    closed_at
FROM trades
WHERE account_id = 1
    AND symbol = 'BTC/USD'
ORDER BY closed_at DESC
LIMIT 10;
```

### Get Daily P&L

```sql
SELECT
    DATE(closed_at) as trade_date,
    COUNT(*) as trades,
    ROUND(SUM(net_pnl), 2) as daily_pnl
FROM trades
WHERE account_id = 1
GROUP BY DATE(closed_at)
ORDER BY trade_date DESC;
```

## Troubleshooting

### Connection Failed

```
Error: failed to ping database: dial tcp [::1]:5432: connectex: No connection could be made
```

**Solution**: Ensure PostgreSQL service is running:

```powershell
Start-Service postgresql-x64-*
```

### Authentication Failed

```
Error: pq: password authentication failed for user "candlecore"
```

**Solution**: Verify password in config.yaml matches what you set in Step 2.

### Database Does Not Exist

```
Error: pq: database "candlecore" does not exist
```

**Solution**: Run Step 2 again to create the database.

### Tables Not Found

```
Error: relation "accounts" does not exist
```

**Solution**: Run the schema file (Step 3).

## Switching Between File and Database Store

You can easily switch between file-based and database storage:

**Use Database:**

```yaml
database:
  enabled: true
```

**Use File Store:**

```yaml
database:
  enabled: false
```

Both will maintain separate state, so you can test both approaches.

## Backup and Restore

### Backup Database

```powershell
pg_dump -U candlecore -d candlecore > backup.sql
```

### Restore Database

```powershell
psql -U candlecore -d candlecore < backup.sql
```

### Export Trade History to CSV

```powershell
psql -U candlecore -d candlecore -c "COPY (SELECT * FROM trades ORDER BY closed_at) TO STDOUT CSV HEADER" > trades_export.csv
```

## Production Recommendations

1. **Use SSL**: Set `sslmode: require` in production
2. **Strong Password**: Use a generated password, not a simple one
3. **Environment Variables**: Store password in env vars, not config file
4. **Connection Pooling**: Already configured (max 25 connections)
5. **Regular Backups**: Automate pg_dump for backups
6. **Monitoring**: Monitor database size and query performance

## Next Steps

Once PostgreSQL is working:

1. ✅ Run backtests with database persistence
2. ✅ Query trade history directly from SQL
3. ✅ Build performance dashboards
4. ✅ Analyze strategy metrics over time
5. ✅ Export data for external analysis

## Advanced: Custom Queries

The PostgreSQL store provides additional methods:

```go
// In your code:
ctx := context.Background()

// Get trade history
trades, err := pgStore.GetTradeHistory(ctx, "BTC/USD", 100)

// Get performance metrics
metrics, err := pgStore.GetPerformanceMetrics(ctx)
fmt.Printf("Win Rate: %.2f%%\n", metrics["win_rate"])
fmt.Printf("Total P&L: %.2f\n", metrics["total_pnl"])
```

---

**You're all set!** PostgreSQL provides robust, queryable state persistence for Candlecore. 🚀
