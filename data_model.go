package main

import "database/sql"

const sqlReadCommand = `
	SELECT c.Uuid, c.Name, c.parent_cmd
	FROM command c
	JOIN command_alias a ON a.cmd_uuid = c.Uuid
	WHERE c.Name = ?1 OR a.Name = ?2
`

const sqlReadCommandAliases = `
	SELECT a.Uuid, a.cmd_uuid, a.Name 
	FROM command_alias a
	WHERE a.cmd_uuid = ?1
`

const sqlReadSubCommands = `
	SELECT c.Uuid, c.Name, c.parent_cmd
	FROM command c
	WHERE c.parent_cmd = ?1
	ORDER BY c.Name
`

const sqlReadCommandArgs = `
	SELECT ca.Uuid, ca.cmd_uuid, ca.arg_type, ca.Description, ca.long_name, ca.short_name
	FROM command_arg ca
	JOIN command c ON c.Uuid = ca.cmd_uuid
	WHERE c.Uuid = ?1
	ORDER BY ca.long_name, ca.short_name
`

const sqlReadCommandOpts = `
	SELECT co.Uuid, co.cmd_arg_uuid, co.Name
	FROM command_opt co
	JOIN command_arg ca ON ca.Uuid = co.cmd_arg_uuid
	WHERE ca.Uuid = ?1
	ORDER BY co.Name
`

const sqlWriteCommand = `
	INSERT INTO command
		(uuid, name, parent_cmd)
	VALUES 
		(?1, ?2, ?3)
`

const sqlWriteCommandAlias = `
    INSERT INTO command_alias
    	(uuid, cmd_uuid, name)
    VALUES
		(?1, ?2, ?3)
`

const sqlWriteCommandArg = `
    INSERT INTO command_arg
        (uuid, cmd_uuid, arg_type, description, long_name, short_name)
    VALUES
		(?1, ?2, ?3, ?4, ?5, ?6)
`

const sqlWriteCommandOpt = `
	INSERT INTO command_opt
		(uuid, cmd_arg_uuid, name)
	VALUES
		(?1, ?2, ?3)
`

const sqlDeleteCommand = `
	DELETE FROM command
	WHERE name = ?1
	AND parent_cmd IS NULL
`

type BceCommand struct {
	Uuid               string            `json:"uuid"`
	Name               string            `json:"name"`
	ParentCmdUuid      *string           `json:"-"`
	Aliases            []BceCommandAlias `json:"aliases"`
	SubCommands        []BceCommand      `json:"sub_commands"`
	Args               []BceCommandArg   `json:"args"`
	IsPresentOnCmdLine bool              `json:"-"`
}

type BceCommandAlias struct {
	Uuid    string `json:"uuid"`
	CmdUuid string `json:"-"`
	Name    string `json:"name"`
}

type BceCommandArg struct {
	Uuid               string          `json:"uuid"`
	CmdUuid            string          `json:"-"`
	ArgType            string          `json:"arg_type"`
	Description        string          `json:"description"`
	LongName           string          `json:"long_name"`
	ShortName          string          `json:"short_name"`
	IsPresentOnCmdLine bool            `json:"-"`
	Opts               []BceCommandOpt `json:"opts"`
}

type BceCommandOpt struct {
	Uuid    string `json:"uuid"`
	ArgUuid string `json:"-"`
	Name    string `json:"name"`
}

func DbQueryCommand(conn *sql.DB, cmdName string) (*BceCommand, error) {
	var cmd BceCommand

	stmt, err := conn.Prepare(sqlReadCommand)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cmdName, cmdName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&cmd.Uuid, &cmd.Name, &cmd.ParentCmdUuid)
		if err != nil {
			return nil, err
		}
	}

	cmd.Aliases, err = DbQueryCommandAliases(conn, cmd.Uuid)
	if err != nil {
		return nil, err
	}
	cmd.SubCommands, err = DbQuerySubCommands(conn, cmd.Uuid)
	if err != nil {
		return nil, err
	}
	cmd.Args, err = DbQueryCommandArgs(conn, cmd.Uuid)
	if err != nil {
		return nil, err
	}

	return &cmd, nil
}

func DbQueryCommandAliases(conn *sql.DB, parentCmdUuid string) ([]BceCommandAlias, error) {
	var aliases []BceCommandAlias

	stmt, err := conn.Prepare(sqlReadCommandAliases)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(parentCmdUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var alias BceCommandAlias
		err = rows.Scan(&alias.Uuid, &alias.CmdUuid, &alias.Name)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}

	return aliases, nil
}

func DbQuerySubCommands(conn *sql.DB, parentCmdUuid string) ([]BceCommand, error) {
	var subCmds []BceCommand

	stmt, err := conn.Prepare(sqlReadSubCommands)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(parentCmdUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var subCmd BceCommand
		err = rows.Scan(&subCmd.Uuid, &subCmd.Name, &subCmd.ParentCmdUuid)
		if err != nil {
			return nil, err
		}

		// populate child Aliases
		subCmd.Aliases, err = DbQueryCommandAliases(conn, subCmd.Uuid)
		if err != nil {
			return nil, err
		}

		// populate child Args
		subCmd.Args, err = DbQueryCommandArgs(conn, subCmd.Uuid)
		if err != nil {
			return nil, err
		}

		// populate child sub-cmds
		subCmd.SubCommands, err = DbQuerySubCommands(conn, subCmd.Uuid)
		if err != nil {
			return nil, err
		}

		subCmds = append(subCmds, subCmd)
	}

	return subCmds, nil
}

func DbQueryCommandArgs(conn *sql.DB, cmdUuid string) ([]BceCommandArg, error) {
	var args []BceCommandArg

	stmt, err := conn.Prepare(sqlReadCommandArgs)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cmdUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var arg BceCommandArg
		// ca.Uuid, ca.cmd_uuid, ca.arg_type, ca.Description, ca.long_name, ca.short_name
		err := rows.Scan(&arg.Uuid, &arg.CmdUuid, &arg.ArgType, &arg.Description, &arg.LongName, &arg.ShortName)
		if err != nil {
			return nil, err
		}
		arg.Opts, err = DbQueryCommandOpts(conn, arg.Uuid)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return args, nil
}

func DbQueryCommandOpts(conn *sql.DB, argUuid string) ([]BceCommandOpt, error) {
	var opts []BceCommandOpt

	stmt, err := conn.Prepare(sqlReadCommandOpts)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(argUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var opt BceCommandOpt
		err := rows.Scan(&opt.Uuid, &opt.ArgUuid, &opt.Name)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}

	return opts, nil
}

func DbInsertCommand(conn *sql.DB, cmd BceCommand) error {
	// insert the command
	stmt, err := conn.Prepare(sqlWriteCommand)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(cmd.Uuid, cmd.Name, cmd.ParentCmdUuid)
	}
	if err != nil {
		return err
	}

	// insert the aliases
	for _, alias := range cmd.Aliases {
		err = DbInsertCommandAlias(conn, alias)
		if err != nil {
			return err
		}
	}

	// insert the args
	for _, arg := range cmd.Args {
		err = DbInsertCommandArg(conn, arg)
		if err != nil {
			return err
		}
	}

	return err
}

func DbInsertCommandAlias(conn *sql.DB, alias BceCommandAlias) error {
	// insert the alias
	stmt, err := conn.Prepare(sqlWriteCommandAlias)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(alias.Uuid, alias.CmdUuid, alias.Name)
	}
	return err
}

func DbInsertCommandArg(conn *sql.DB, arg BceCommandArg) error {
	// insert the arg
	stmt, err := conn.Prepare(sqlWriteCommandArg)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(arg.Uuid, arg.CmdUuid, arg.ArgType, arg.Description, arg.LongName, arg.ShortName)
	}
	if err != nil {
		return err
	}

	// insert the opts
	for _, opt := range arg.Opts {
		err = DbInsertCommandOpt(conn, opt)
		if err != nil {
			return err
		}
	}
	return err
}

func DbInsertCommandOpt(conn *sql.DB, opt BceCommandOpt) error {
	// insert the opt
	stmt, err := conn.Prepare(sqlWriteCommandOpt)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(opt.Uuid, opt.ArgUuid, opt.Name)
	}
	return err
}

func DbDeleteCommand(conn *sql.DB, commandName string) error {
	// delete the command (cascade to children)
	stmt, err := conn.Prepare(sqlDeleteCommand)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(commandName)
	}
	return err
}
