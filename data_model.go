package main

import "database/sql"

const CmdReadSql = `
	SELECT c.uuid, c.name, c.parent_cmd
	FROM command c
	JOIN command_alias a ON a.cmd_uuid = c.uuid
	WHERE c.name = ?1 OR a.name = ?2
`

const CmdAliasesReadSql = `
	SELECT a.uuid, a.cmd_uuid, a.name 
	FROM command_alias a
	WHERE a.cmd_uuid = ?1
`

const SubCmdReadSql = `
	SELECT c.uuid, c.name, c.parent_cmd
	FROM command c
	WHERE c.parent_cmd = ?1
	ORDER BY c.name
`

const CmdArgReadSql = `
	SELECT ca.uuid, ca.cmd_uuid, ca.arg_type, ca.description, ca.long_name, ca.short_name
	FROM command_arg ca
	JOIN command c ON c.uuid = ca.cmd_uuid
	WHERE c.uuid = ?1
	ORDER BY ca.long_name, ca.short_name
`

const CmdOptReadSql = `
	SELECT co.uuid, co.cmd_arg_uuid, co.name
	FROM command_opt co
	JOIN command_arg ca ON ca.uuid = co.cmd_arg_uuid
	WHERE ca.uuid = ?1
	ORDER BY co.name
`

type BceCommand struct {
	uuid               string
	name               string
	parentCmdUuid      *string
	aliases            []BceCommandAlias
	subCommands        []BceCommand
	args               []BceCommandArg
	isPresentOnCmdLine bool
}

type BceCommandAlias struct {
	uuid    string
	cmdUuid string
	name    string
}

type BceCommandArg struct {
	uuid               string
	cmdUuid            string
	argType            string
	description        string
	longName           string
	shortName          string
	isPresentOnCmdLine bool
	opts               []BceCommandOpt
}

type BceCommandOpt struct {
	uuid    string
	argUuid string
	name    string
}

func DbQueryCommand(conn *sql.DB, cmdName string) (*BceCommand, error) {
	var cmd BceCommand

	stmt, err := conn.Prepare(CmdReadSql)
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
		err = rows.Scan(&cmd.uuid, &cmd.name, &cmd.parentCmdUuid)
		if err != nil {
			return nil, err
		}
	}

	cmd.aliases, err = DbQueryCommandAliases(conn, cmd.uuid)
	if err != nil {
		return nil, err
	}
	cmd.subCommands, err = DbQuerySubCommands(conn, cmd.uuid)
	if err != nil {
		return nil, err
	}
	cmd.args, err = DbQueryCommandArgs(conn, cmd.uuid)
	if err != nil {
		return nil, err
	}

	return &cmd, nil
}

func DbQueryCommandAliases(conn *sql.DB, parentCmdUuid string) ([]BceCommandAlias, error) {
	var aliases []BceCommandAlias

	stmt, err := conn.Prepare(CmdAliasesReadSql)
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
		err = rows.Scan(&alias.uuid, &alias.cmdUuid, &alias.name)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}

	return aliases, nil
}

func DbQuerySubCommands(conn *sql.DB, parentCmdUuid string) ([]BceCommand, error) {
	var subCmds []BceCommand

	stmt, err := conn.Prepare(SubCmdReadSql)
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
		err = rows.Scan(&subCmd.uuid, &subCmd.name, &subCmd.parentCmdUuid)
		if err != nil {
			return nil, err
		}

		// TODO: populate child aliases
		subCmd.aliases, err = DbQueryCommandAliases(conn, subCmd.uuid)
		if err != nil {
			return nil, err
		}

		// populate child args
		subCmd.args, err = DbQueryCommandArgs(conn, subCmd.uuid)
		if err != nil {
			return nil, err
		}

		// populate child sub-cmds
		subCmd.subCommands, err = DbQuerySubCommands(conn, subCmd.uuid)
		if err != nil {
			return nil, err
		}

		subCmds = append(subCmds, subCmd)
	}

	return subCmds, nil
}

func DbQueryCommandArgs(conn *sql.DB, cmdUuid string) ([]BceCommandArg, error) {
	var args []BceCommandArg

	stmt, err := conn.Prepare(CmdArgReadSql)
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
		// ca.uuid, ca.cmd_uuid, ca.arg_type, ca.description, ca.long_name, ca.short_name
		err := rows.Scan(&arg.uuid, &arg.cmdUuid, &arg.argType, &arg.description, &arg.longName, &arg.shortName)
		if err != nil {
			return nil, err
		}
		arg.opts, err = DbQueryCommandOpts(conn, arg.uuid)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return args, nil
}

func DbQueryCommandOpts(conn *sql.DB, argUuid string) ([]BceCommandOpt, error) {
	var opts []BceCommandOpt

	stmt, err := conn.Prepare(CmdOptReadSql)
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
		err := rows.Scan(&opt.uuid, &opt.argUuid, &opt.name)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}

	return opts, nil
}
