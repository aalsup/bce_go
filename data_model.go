package main

import "database/sql"

const sqlReadCommand = `
	SELECT c.uuid, c.name, c.parent_cmd
	FROM command c
	JOIN command_alias a ON a.cmd_uuid = c.uuid
	WHERE c.name = ?1 OR a.name = ?2
`

const sqlReadCommandAliases = `
	SELECT a.uuid, a.cmd_uuid, a.name 
	FROM command_alias a
	WHERE a.cmd_uuid = ?1
`

const sqlReadSubCommands = `
	SELECT c.uuid, c.name, c.parent_cmd
	FROM command c
	WHERE c.parent_cmd = ?1
	ORDER BY c.name
`

const sqlReadCommandArgs = `
	SELECT ca.uuid, ca.cmd_uuid, ca.arg_type, ca.description, ca.long_name, ca.short_name
	FROM command_arg ca
	JOIN command c ON c.uuid = ca.cmd_uuid
	WHERE c.uuid = ?1
	ORDER BY ca.long_name, ca.short_name
`

const sqlReadCommandOpts = `
	SELECT co.uuid, co.cmd_arg_uuid, co.name
	FROM command_opt co
	JOIN command_arg ca ON ca.uuid = co.cmd_arg_uuid
	WHERE ca.uuid = ?1
	ORDER BY co.name
`

const sqlReadRootCommandNames = `
	SELECT c.name
	FROM command c
	WHERE c.parent_cmd IS NULL
	ORDER BY c.name
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

func DBQueryCommand(conn *sql.DB, cmdName string) (*BceCommand, error) {
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

	cmd.Aliases, err = cmd.QueryAliases(conn)
	if err != nil {
		return nil, err
	}
	cmd.SubCommands, err = cmd.QuerySubCommands(conn)
	if err != nil {
		return nil, err
	}
	cmd.Args, err = cmd.QueryArgs(conn)
	if err != nil {
		return nil, err
	}

	return &cmd, nil
}

func (cmd BceCommand) QueryAliases(conn *sql.DB) ([]BceCommandAlias, error) {
	var aliases []BceCommandAlias

	stmt, err := conn.Prepare(sqlReadCommandAliases)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cmd.Uuid)
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

func (cmd BceCommand) QuerySubCommands(conn *sql.DB) ([]BceCommand, error) {
	var subCmds []BceCommand

	stmt, err := conn.Prepare(sqlReadSubCommands)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cmd.Uuid)
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
		subCmd.Aliases, err = subCmd.QueryAliases(conn)
		if err != nil {
			return nil, err
		}

		// populate child Args
		subCmd.Args, err = subCmd.QueryArgs(conn)
		if err != nil {
			return nil, err
		}

		// populate child sub-cmds
		subCmd.SubCommands, err = subCmd.QuerySubCommands(conn)
		if err != nil {
			return nil, err
		}

		subCmds = append(subCmds, subCmd)
	}

	return subCmds, nil
}

func (cmd BceCommand) QueryArgs(conn *sql.DB) ([]BceCommandArg, error) {
	var args []BceCommandArg

	stmt, err := conn.Prepare(sqlReadCommandArgs)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cmd.Uuid)
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
		arg.Opts, err = arg.QueryOpts(conn)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return args, nil
}

func (arg BceCommandArg) QueryOpts(conn *sql.DB) ([]BceCommandOpt, error) {
	var opts []BceCommandOpt

	stmt, err := conn.Prepare(sqlReadCommandOpts)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(arg.Uuid)
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

func DBQueryRootCommandNames(conn *sql.DB) ([]string, error) {
	var cmdNames []string

	stmt, err := conn.Prepare(sqlReadRootCommandNames)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cmdName string
		err = rows.Scan(&cmdName)
		if err != nil {
			return nil, err
		}
		cmdNames = append(cmdNames, cmdName)
	}

	return cmdNames, nil
}

func (cmd BceCommand) InsertDB(conn *sql.DB) error {
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
		err = alias.InsertDB(conn)
		if err != nil {
			return err
		}
	}

	// insert the args
	for _, arg := range cmd.Args {
		err = arg.InsertDB(conn)
		if err != nil {
			return err
		}
	}

	return err
}

func (alias BceCommandAlias) InsertDB(conn *sql.DB) error {
	// insert the alias
	stmt, err := conn.Prepare(sqlWriteCommandAlias)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(alias.Uuid, alias.CmdUuid, alias.Name)
	}
	return err
}

func (arg BceCommandArg) InsertDB(conn *sql.DB) error {
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
		err = opt.InsertDB(conn)
		if err != nil {
			return err
		}
	}
	return err
}

func (opt BceCommandOpt) InsertDB(conn *sql.DB) error {
	// insert the opt
	stmt, err := conn.Prepare(sqlWriteCommandOpt)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(opt.Uuid, opt.ArgUuid, opt.Name)
	}
	return err
}

func DBDeleteCommand(conn *sql.DB, commandName string) error {
	// delete the command (cascade to children)
	stmt, err := conn.Prepare(sqlDeleteCommand)
	if err == nil {
		defer stmt.Close()
		_, err = stmt.Exec(commandName)
	}
	return err
}
