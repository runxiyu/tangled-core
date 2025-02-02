package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

func Setup(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		create table if not exists public_keys (
			id integer primary key autoincrement,
			did text not null,
			name text not null,
			key text not null,
			created timestamp default current_timestamp,
			unique(did, name, key)
		);
		create table if not exists users (
			id integer primary key autoincrement,
			did text not null,
			unique(did),
			foreign key (did) references public_keys(did) on delete cascade
		);
		create table if not exists repos (
			id integer primary key autoincrement,
			did text not null,
			name text not null,
			description text not null,
			created timestamp default current_timestamp,
			unique(did, name)
		);
		create table if not exists access_levels (
			id integer primary key autoincrement,
			repo_id integer not null,
			did text not null,
			access text not null check (access in ('OWNER', 'WRITER')),
			created timestamp default current_timestamp,
			unique(repo_id, did),
			foreign key (repo_id) references repos(id) on delete cascade
		);
	`)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}
