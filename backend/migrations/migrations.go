package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed postgres/*.sql
var files embed.FS

type Migration struct {
	Version string
	SQL     string
}

func PostgreSQL() ([]Migration, error) {
	entries, err := fs.ReadDir(files, "postgres")
	if err != nil {
		return nil, fmt.Errorf("read PostgreSQL migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := files.ReadFile("postgres/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		migrations = append(migrations, Migration{Version: entry.Name(), SQL: string(content)})
	}
	return migrations, nil
}
