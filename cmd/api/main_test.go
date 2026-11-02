package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBaseInaccessible(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{dir, filepath.Join(dir, "absent", "garage.db")} {
		t.Run(path, func(t *testing.T) {
			err := run(context.Background(), path)
			if err == nil || !strings.Contains(err.Error(), "préparer la base") {
				t.Fatalf("échec de démarrage attendu, obtenu : %v", err)
			}
		})
	}
}
