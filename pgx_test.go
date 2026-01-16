package pgxadapter

import (
	"context"
	"sync"
	"testing"
)

func TestPgxTx_Commit_Idempotency(t *testing.T) {
	ctx := t.Context()

	idx := 0
	mockTx := &MockTx{
		commit: func(ctx context.Context) error {
			idx++
			return nil
		},
	}

	tx := &pgxTx{
		tx:    mockTx,
		ro:    sync.Once{},
		co:    sync.Once{},
		txErr: nil,
	}

	err1 := tx.Commit(ctx)
	err2 := tx.Commit(ctx)
	err3 := tx.Commit(ctx)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("unexpected error occured")
	}

	if idx != 1 {
		t.Fatalf("unexpected index count %d expected 1", idx)
	}
}
