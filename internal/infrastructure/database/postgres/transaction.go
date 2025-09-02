package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Transaction struct {
	tx     pgx.Tx
	closed bool
}

type TransactionManager struct {
	client *Client
}

type TxFunc func(tx *Transaction) error

func NewTransactionManager(client *Client) *TransactionManager {
	return &TransactionManager{
		client: client,
	}
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, fn TxFunc) error {
	return tm.WithTransactionIsolation(ctx, pgx.TxIsoLevel(""), fn)
}

func (tm *TransactionManager) WithTransactionIsolation(ctx context.Context, isoLevel pgx.TxIsoLevel, fn TxFunc) error {
	if tm.client.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	conn, err := tm.client.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for transaction: %w", err)
	}
	defer conn.Release()

	// Options de transaction avec niveau d'isolation
	txOptions := pgx.TxOptions{}
	if isoLevel != "" {
		txOptions.IsoLevel = isoLevel
	}

	pgxTx, err := conn.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	tx := &Transaction{
		tx:     pgxTx,
		closed: false,
	}

	// Rollback automatique en cas d'erreur avec defer
	defer func() {
		if !tx.closed {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// Log de l'erreur de rollback mais ne pas masquer l'erreur originale
				fmt.Printf("Warning: failed to rollback transaction: %v\n", rollbackErr)
			}
		}
	}()

	// Exécuter la fonction dans la transaction
	if err := fn(tx); err != nil {
		return err
	}

	// Commit si tout s'est bien passé
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (t *Transaction) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if t.closed {
		return nil, fmt.Errorf("transaction is closed")
	}
	return t.tx.Query(ctx, sql, args...)
}

func (t *Transaction) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	// QueryRow ne peut pas retourner d'erreur directement, 
	// mais l'erreur sera disponible lors du Scan
	if t.closed {
		// Retourner un row qui génèrera une erreur lors du scan
		return &closedTxRow{err: fmt.Errorf("transaction is closed")}
	}
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *Transaction) Exec(ctx context.Context, sql string, args ...interface{}) error {
	if t.closed {
		return fmt.Errorf("transaction is closed")
	}
	_, err := t.tx.Exec(ctx, sql, args...)
	return err
}

func (t *Transaction) Commit(ctx context.Context) error {
	if t.closed {
		return fmt.Errorf("transaction is already closed")
	}
	
	err := t.tx.Commit(ctx)
	t.closed = true
	return err
}

func (t *Transaction) Rollback(ctx context.Context) error {
	if t.closed {
		return nil // Déjà fermée, pas d'erreur
	}
	
	err := t.tx.Rollback(ctx)
	t.closed = true
	return err
}

func (t *Transaction) IsClosed() bool {
	return t.closed
}

// Helper pour les savepoints (transactions imbriquées)
func (t *Transaction) CreateSavepoint(ctx context.Context, name string) error {
	if t.closed {
		return fmt.Errorf("transaction is closed")
	}
	
	sql := fmt.Sprintf("SAVEPOINT %s", pgx.Identifier{name}.Sanitize())
	_, err := t.tx.Exec(ctx, sql)
	return err
}

func (t *Transaction) RollbackToSavepoint(ctx context.Context, name string) error {
	if t.closed {
		return fmt.Errorf("transaction is closed")
	}
	
	sql := fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", pgx.Identifier{name}.Sanitize())
	_, err := t.tx.Exec(ctx, sql)
	return err
}

func (t *Transaction) ReleaseSavepoint(ctx context.Context, name string) error {
	if t.closed {
		return fmt.Errorf("transaction is closed")
	}
	
	sql := fmt.Sprintf("RELEASE SAVEPOINT %s", pgx.Identifier{name}.Sanitize())
	_, err := t.tx.Exec(ctx, sql)
	return err
}

// Type pour gérer le cas où QueryRow est appelé sur une transaction fermée
type closedTxRow struct {
	err error
}

func (r *closedTxRow) Scan(dest ...interface{}) error {
	return r.err
}

// Wrapper pour opérations transactionnelles courantes

func (tm *TransactionManager) InsertWithReturn(ctx context.Context, table string, columns []string, values []interface{}, returnColumns []string) (pgx.Row, error) {
	var row pgx.Row
	
	err := tm.WithTransaction(ctx, func(tx *Transaction) error {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		
		sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
			table,
			joinIdentifiers(columns),
			joinStrings(placeholders, ","),
			joinIdentifiers(returnColumns),
		)
		
		row = tx.QueryRow(ctx, sql, values...)
		return nil
	})
	
	return row, err
}

func (tm *TransactionManager) UpdateWithCondition(ctx context.Context, table string, setColumns []string, setValues []interface{}, whereCondition string, whereArgs []interface{}) error {
	return tm.WithTransaction(ctx, func(tx *Transaction) error {
		setPairs := make([]string, len(setColumns))
		for i, col := range setColumns {
			setPairs[i] = fmt.Sprintf("%s = $%d", col, i+1)
		}
		
		sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
			table,
			joinStrings(setPairs, ","),
			whereCondition,
		)
		
		allArgs := append(setValues, whereArgs...)
		return tx.Exec(ctx, sql, allArgs...)
	})
}

// Helpers utilitaires
func joinIdentifiers(identifiers []string) string {
	if len(identifiers) == 0 {
		return ""
	}
	
	result := identifiers[0]
	for i := 1; i < len(identifiers); i++ {
		result += ", " + identifiers[i]
	}
	return result
}

func joinStrings(strings []string, separator string) string {
	if len(strings) == 0 {
		return ""
	}
	
	result := strings[0]
	for i := 1; i < len(strings); i++ {
		result += separator + strings[i]
	}
	return result
}