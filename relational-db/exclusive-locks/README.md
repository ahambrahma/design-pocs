# Exclusive Locks — Web Check-in Seat Booking POC

Problem statement
-----------------
Design a web check-in seat-selection system for an airline where passengers select seats for an already-confirmed reservation (PNR). The system must prevent two passengers from booking the same seat. If a passenger attempts to book a seat already assigned to someone else, show an error and request re-selection.

This POC explores multiple approaches to coordinate concurrent seat bookings against a relational database and shows trade-offs in correctness and performance.

Approaches tried
-----------------
1. No-update (select then update)
   - Flow: SELECT a seat WHERE user_id IS NULL LIMIT 1; then UPDATE that row to set user_id.
   - Problem: Without row-level locking the SELECT can return the same seat to multiple concurrent workers, causing UPDATE conflicts or overwrites.
   - Behavior: Under high concurrency, multiple goroutines may claim the same seat unless additional locks or transactional semantics are used.

2. With UPDATE / SELECT ... FOR UPDATE
   - Flow: Start a transaction, SELECT ... FOR UPDATE to lock a candidate row, then UPDATE and COMMIT.
   - Benefit: Row-level exclusive lock prevents other transactions from selecting the same row concurrently (MySQL InnoDB / Postgres).
   - Caveats: Must run inside transactions; syntax differs by DB. Ensure indexes/where clause pick stable rows to lock.

3. Optimistic locking (version column)
   - Flow: Table has a `version` column. Read the row and its version, attempt an UPDATE that sets the new user_id and increments version WHERE id = ? AND version = ?. If RowsAffected == 0, a concurrent update happened — retry or fail.
   - Benefit: No long locks; scales well for low-contention scenarios.
   - Caveats: Under heavy contention many retries may be needed; must add a `version` column and atomic UPDATE checks.

4. Update-skip / conditional update
   - Flow: Try UPDATE seats SET user_id = ? WHERE id = ? AND user_id IS NULL; check RowsAffected to decide success or failure.
   - Benefit: Single-statement claim; helps avoid some races if you already know the id.
   - Caveats: Still requires a safe way to pick an id to attempt (selecting a candidate may race).

Implementation notes
--------------------
- The POCs use a simple connection pool (db.CommonPool) that provides *sql.DB wrappers to goroutines.
- The code assumes MySQL/InnoDB by default (github.com/go-sql-driver/mysql). SELECT ... FOR UPDATE works in InnoDB when used inside transactions.
- Optimistic locking implementations require a `version` column in the seat table (e.g. seats_v2 with `version`).

Database setup
--------------
1. Create and seed the DB:
   - If using the provided SQL file:
     sqlite3 airline_booking.db < /Users/shubhamsharma/airline_booking.sql
     or for MySQL:
     mysql -u root -p airline_booking < /Users/shubhamsharma/airline_booking.sql
2. Ensure the tables include any extra columns needed by approaches (e.g. `version` for optimistic locking — the POC resets `seats_v2` in db.Init).

Running POCs
------------
- Ensure module path and imports align with your `go.mod`. Example:
  module github.com/shubhamsharma/projects/go/design-pocs/relational-db/exclusive-locks
- From the repo root:
  go run ./main.go
- main.go selects which approach to run; uncomment the approach you want to exercise.

Troubleshooting & tips
----------------------
- Package import paths in Go are module-based. Use a go.mod with a module path that matches imports, or add `replace` directives to point imports to local folders.
- Use SELECT ... FOR UPDATE inside a transaction for exclusive row locks (supported in InnoDB / Postgres). Make sure autocommit is off (use tx.Begin()).
- Use RowsAffected checks on UPDATE statements to detect lost races in optimistic or conditional-update strategies.
- For simulations, run many concurrent goroutines to exercise race conditions and confirm correctness.

Summary
-------
This project demonstrates different DB-level concurrency control techniques for booking seats in a concurrent environment. Each approach has trade-offs; choose pessimistic locking (FOR UPDATE) for strong correctness guarantees at the cost of locking, or optimistic locking for better scalability when conflicts are rare.