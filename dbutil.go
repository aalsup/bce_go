package main

import (
	"database/sql"
)

const DbSchemaVersion = 1

const CreateCompletionCommandSql = ` 
	CREATE TABLE IF NOT EXISTS command (
      uuid TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      parent_cmd TEXT,
      FOREIGN KEY(parent_cmd) REFERENCES command(uuid) ON DELETE CASCADE
    );
	CREATE UNIQUE INDEX command_name_idx
 		ON command (name);
	CREATE INDEX command_parent_idx
		ON command (parent_cmd); `

const CreateCompletionCommandAliasSql = `
	CREATE TABLE IF NOT EXISTS command_alias (
    	uuid TEXT PRIMARY KEY,
        cmd_uuid TEXT NOT NULL,
        name TEXT NOT NULL,
        FOREIGN KEY(cmd_uuid) REFERENCES command(uuid) ON DELETE CASCADE
    );
	CREATE INDEX command_alias_name_idx
		ON command_alias (name);
    CREATE INDEX command_alias_cmd_uuid_idx
        ON command_alias (cmd_uuid);
    CREATE UNIQUE INDEX command_alias_cmd_name_idx
        ON command_alias (cmd_uuid, name);
`
const CreateCompletionCommandArgSql = `
	CREATE TABLE IF NOT EXISTS command_arg (
		uuid TEXT PRIMARY KEY,
        cmd_uuid TEXT NOT NULL,
        arg_type TEXT NOT NULL
        	CHECK (arg_type IN ('NONE', 'OPTION', 'FILE', 'TEXT')),
        description TEXT NOT NULL, 
        long_name TEXT, 
        short_name TEXT, 
        FOREIGN KEY(cmd_uuid) REFERENCES command(uuid) ON DELETE CASCADE, 
        CHECK ( (long_name IS NOT NULL) OR (short_name IS NOT NULL) ) 
	); 
	CREATE INDEX command_arg_cmd_uuid_idx 
        ON command_arg (cmd_uuid); 
	CREATE UNIQUE INDEX command_arg_longname_idx 
        ON command_arg (cmd_uuid, long_name); 
`

const CreateCompletionCommandOptSql = `
	CREATE TABLE IF NOT EXISTS command_opt (
        uuid TEXT PRIMARY KEY, 
        cmd_arg_uuid TEXT NOT NULL, 
        name TEXT NOT NULL, 
		FOREIGN KEY(cmd_arg_uuid) REFERENCES command_arg(uuid) ON DELETE CASCADE 
	);
	CREATE INDEX command_opt_cmd_arg_idx 
        ON command_opt (cmd_arg_uuid); 
	CREATE UNIQUE INDEX command_opt_arg_name_idx 
        ON command_opt (cmd_arg_uuid, name); 
`

func DbOpen(filename string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	_, err = conn.Exec("PRAGMA foreign_keys = 1;")
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func DbClose(conn *sql.DB) {
	_ = conn.Close()
}

func DbGetSchemaVersion(conn *sql.DB) (int, error) {
	var version int

	row, err := conn.Query("PRAGMA user_version;")
	if err != nil {
		return 0, err
	}
	if row.Next() {
		row.Scan(&version)
	}
	row.Close()
	return version, nil
}

func DbCreateSchema(conn *sql.DB) error {
	_, err := conn.Exec(CreateCompletionCommandSql)
	if err != nil {
		return err
	}

	_, err = conn.Exec(CreateCompletionCommandAliasSql)
	if err != nil {
		return err
	}

	_, err = conn.Exec(CreateCompletionCommandArgSql)
	if err != nil {
		return err
	}

	_, err = conn.Exec(CreateCompletionCommandOptSql)
	if err != nil {
		return err
	}

	_, err = conn.Exec("PRAGMA user_version = 1;")
	return err
}
