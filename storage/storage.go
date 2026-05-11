// storage/storage.go
// Banco SQLite com historico de jobs de impressao.

package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Job struct {
	ID        int64
	Name      string
	PDFPath   string
	Pages     int
	ByteSize  int
	CreatedAt time.Time
}

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco: %w", err)
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL,
			pdf_path   TEXT    NOT NULL,
			pages      INTEGER NOT NULL DEFAULT 1,
			byte_size  INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at DESC);
	`)
	return err
}

func (db *DB) InsertJob(j Job) error {
	_, err := db.conn.Exec(
		`INSERT INTO jobs (name, pdf_path, pages, byte_size, created_at) VALUES (?,?,?,?,?)`,
		j.Name, j.PDFPath, j.Pages, j.ByteSize, j.CreatedAt.UTC(),
	)
	return err
}

func (db *DB) ListJobs(limit int) ([]Job, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, pdf_path, pages, byte_size, created_at FROM jobs ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		var ts string
		if err := rows.Scan(&j.ID, &j.Name, &j.PDFPath, &j.Pages, &j.ByteSize, &ts); err != nil {
			return nil, err
		}
		j.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", ts)
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (db *DB) DeleteJob(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM jobs WHERE id = ?`, id)
	return err
}

func (db *DB) CountJobs() (int, error) {
	var n int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM jobs`).Scan(&n)
	return n, err
}
