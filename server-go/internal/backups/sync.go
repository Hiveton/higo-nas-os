package backups

import (
	"context"

	"higoos/server-go/internal/filesync"
)

// syncResult / verifyResult and the sync engine now live in internal/filesync so
// the backups and sync domains share one implementation. These thin aliases keep
// the rest of the backups package unchanged.
type syncResult = filesync.SyncResult
type verifyResult = filesync.VerifyResult

func syncTree(ctx context.Context, source, target string) (syncResult, error) {
	return filesync.SyncTree(ctx, source, target, nil)
}

func verifyTree(ctx context.Context, source, target string) (verifyResult, error) {
	return filesync.VerifyTree(ctx, source, target)
}
