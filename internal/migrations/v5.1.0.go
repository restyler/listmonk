package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_1_0 performs the DB migrations for dynamic segments and SQL snippets.
func V5_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Create SQL snippets table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sql_snippets (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			query_sql TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_by INTEGER NULL REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create dynamic segments table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS dynamic_segments (
			id SERIAL PRIMARY KEY,
			uuid uuid NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			list_id INTEGER NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
			snippet_id INTEGER REFERENCES sql_snippets(id) ON DELETE SET NULL,
			is_active BOOLEAN DEFAULT true,
			last_run_at TIMESTAMP WITH TIME ZONE,
			last_run_stats JSONB,
			created_by INTEGER NULL REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(name, list_id)
		);
	`); err != nil {
		return err
	}

	// Create dynamic segment runs table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS dynamic_segment_runs (
			id SERIAL PRIMARY KEY,
			segment_id INTEGER NOT NULL REFERENCES dynamic_segments(id) ON DELETE CASCADE,
			added_count INTEGER DEFAULT 0,
			removed_count INTEGER DEFAULT 0,
			total_matched INTEGER DEFAULT 0,
			execution_time_ms INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'success',
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for better performance
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_dynamic_segments_list_id ON dynamic_segments(list_id);
		CREATE INDEX IF NOT EXISTS idx_dynamic_segments_snippet_id ON dynamic_segments(snippet_id);
		CREATE INDEX IF NOT EXISTS idx_dynamic_segments_is_active ON dynamic_segments(is_active);
		CREATE INDEX IF NOT EXISTS idx_dynamic_segment_runs_segment_id ON dynamic_segment_runs(segment_id);
		CREATE INDEX IF NOT EXISTS idx_dynamic_segment_runs_created_at ON dynamic_segment_runs(created_at);
	`); err != nil {
		return err
	}

	// Insert some default SQL snippets
	if _, err := db.Exec(`
		INSERT INTO sql_snippets (name, description, query_sql) VALUES
			('Active Subscribers', 'Get all confirmed subscribers', 'status = ''confirmed'''),
			('Recent Signups', 'Subscribers who joined in the last 30 days', 'status = ''confirmed'' AND created_at >= NOW() - INTERVAL ''30 days'''),
			('Inactive Subscribers', 'Subscribers who haven''t been sent emails recently', 'status = ''confirmed'' AND updated_at <= NOW() - INTERVAL ''90 days'''),
			('High Value Subscribers', 'Subscribers with age greater than 39', '(subscribers.attribs->>''age'')::INT > 39'),
			('Premium Tier', 'Subscribers in premium tier', 'subscribers.attribs->>''tier'' = ''premium'''),
			('Active in Last Week', 'Subscribers active in the last 7 days', 'status = ''confirmed'' AND updated_at >= NOW() - INTERVAL ''7 days''')
		ON CONFLICT (name) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
