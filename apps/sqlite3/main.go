package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/readline.v1"
	"kos"
	sqlitepkg "sqlite3"
)

const consoleTitle = "SQLite Console"

type shell struct {
	db   *sql.DB
	name string
	rl   *readline.Instance
}

func main() {
	console, ok := kos.OpenConsole(consoleTitle)
	if !ok {
		kos.DebugString("sqlite3: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / ready")
	}

	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 2 {
		return fmt.Errorf("usage: sqlite3 [database-file]")
	}

	name := ":memory:"
	if len(os.Args) == 2 {
		name = os.Args[1]
	}

	db, err := sqlitepkg.Open(name)
	if err != nil {
		return err
	}
	defer db.Close()

	_, _ = fmt.Printf(
		"SQLite Console (KolibriOS)\n" +
			"Usage: sqlite3 [database-file]\n" +
			"Without arguments an in-memory database is opened.\n" +
			"End SQL statements with ';'. Commands: .help .open .tables .schema .quit\n",
	)

	rl, err := readline.NewEx(&readline.Config{Prompt: "sql> "})
	if err != nil {
		return err
	}
	defer rl.Close()

	sh := &shell{
		db:   db,
		name: name,
		rl:   rl,
	}
	sh.printConnection()
	return sh.repl()
}

func (sh *shell) repl() error {
	var buffer strings.Builder

	for {
		line, err := sh.rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			_, _ = fmt.Printf("\n")
			return nil
		}
		if err != nil {
			return err
		}

		trimmed := strings.TrimSpace(line)
		if buffer.Len() == 0 {
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, ".") {
				if done, err := sh.handleCommand(trimmed); done {
					return err
				} else if err != nil {
					_, _ = fmt.Fprintf(sh.rl.Stdout(), "Error: %v\n", err)
				}
				sh.rl.Refresh()
				continue
			}
		}

		if buffer.Len() > 0 {
			buffer.WriteByte('\n')
		}
		buffer.WriteString(line)

		if !statementComplete(buffer.String()) {
			sh.rl.SetPrompt("...> ")
			continue
		}

		statement := strings.TrimSpace(buffer.String())
		buffer.Reset()
		sh.rl.SetPrompt("sql> ")

		if statement == "" {
			continue
		}
		if err := sh.execute(statement); err != nil {
			_, _ = fmt.Fprintf(sh.rl.Stdout(), "Error: %v\n", err)
		}
		sh.rl.Refresh()
	}
}

func (sh *shell) handleCommand(command string) (done bool, err error) {
	switch {
	case command == ".quit", command == ".exit":
		return true, nil
	case command == ".help":
		sh.printHelp()
		return false, nil
	case command == ".tables":
		return false, sh.printTables()
	case strings.HasPrefix(command, ".schema"):
		return false, sh.printSchema(strings.TrimSpace(command[len(".schema"):]))
	case strings.HasPrefix(command, ".open"):
		name := strings.TrimSpace(command[len(".open"):])
		if name == "" {
			return false, fmt.Errorf("usage: .open <database-file|:memory:>")
		}
		return false, sh.open(name)
	default:
		return false, fmt.Errorf("unknown command %s", command)
	}
}

func (sh *shell) open(name string) error {
	db, err := sqlitepkg.Open(name)
	if err != nil {
		return err
	}
	old := sh.db
	sh.db = db
	sh.name = name
	if old != nil {
		_ = old.Close()
	}
	sh.printConnection()
	return nil
}

func (sh *shell) execute(statement string) error {
	if looksLikeQuery(statement) {
		return sh.executeQuery(statement)
	}
	return sh.executeExec(statement)
}

func (sh *shell) executeExec(statement string) error {
	result, err := sh.db.Exec(statement)
	if err != nil {
		return err
	}

	rowsAffected, rowsErr := result.RowsAffected()
	lastInsertID, lastErr := result.LastInsertId()

	if rowsErr == nil {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "Rows affected: %d\n", rowsAffected)
	} else {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "OK\n")
	}
	if looksLikeInsert(statement) && lastErr == nil {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "Last insert rowid: %d\n", lastInsertID)
	}
	return nil
}

func (sh *shell) executeQuery(statement string) error {
	rows, err := sh.db.Query(statement)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "OK\n")
		return nil
	}

	values := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	widths := make([]int, len(columns))
	table := make([][]string, 0, 8)

	for index := 0; index < len(columns); index++ {
		scanTargets[index] = &values[index]
		widths[index] = len(columns[index])
	}

	for rows.Next() {
		if err := rows.Scan(scanTargets...); err != nil {
			return err
		}
		row := make([]string, len(columns))
		for index := 0; index < len(columns); index++ {
			row[index] = formatValue(values[index])
			if len(row[index]) > widths[index] {
				widths[index] = len(row[index])
			}
			values[index] = nil
		}
		table = append(table, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	printTable(sh.rl.Stdout(), columns, widths, table)
	_, _ = fmt.Fprintf(sh.rl.Stdout(), "%d row(s)\n", len(table))
	return nil
}

func (sh *shell) printTables() error {
	rows, err := sh.db.Query(
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	names := make([]string, 0, 8)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(names) == 0 {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "(no tables)\n")
		return nil
	}
	_, _ = fmt.Fprintf(sh.rl.Stdout(), "%s\n", strings.Join(names, " "))
	return nil
}

func (sh *shell) printSchema(name string) error {
	query := "SELECT sql FROM sqlite_master WHERE sql IS NOT NULL"
	var rows *sql.Rows
	var err error

	if name != "" {
		query += " AND name = ?"
		rows, err = sh.db.Query(query+" ORDER BY type, name", name)
	} else {
		rows, err = sh.db.Query(query + " ORDER BY type, name")
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	printed := false
	for rows.Next() {
		var sqlText string
		if err := rows.Scan(&sqlText); err != nil {
			return err
		}
		printed = true
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "%s;\n", strings.TrimSpace(sqlText))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !printed {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "(no schema)\n")
	}
	return nil
}

func (sh *shell) printConnection() {
	if sh.name == ":memory:" {
		_, _ = fmt.Fprintf(sh.rl.Stdout(), "Connected to temporary database (:memory:)\n")
		return
	}
	_, _ = fmt.Fprintf(sh.rl.Stdout(), "Connected to %s\n", sh.name)
}

func (sh *shell) printHelp() {
	_, _ = fmt.Fprintf(
		sh.rl.Stdout(),
		".help                Show this help\n"+
			".open FILE|:memory:  Open another database\n"+
			".tables              List user tables\n"+
			".schema [TABLE]      Show schema DDL\n"+
			".quit                Exit the console\n",
	)
}

func statementComplete(statement string) bool {
	statement = strings.TrimSpace(statement)
	return statement != "" && statement[len(statement)-1] == ';'
}

func looksLikeQuery(statement string) bool {
	head := firstSQLToken(statement)
	if head == "" {
		return false
	}
	switch head {
	case "select", "pragma", "explain", "with", "values":
		return true
	}
	lower := strings.ToLower(statement)
	return strings.Contains(lower, " returning ")
}

func looksLikeInsert(statement string) bool {
	head := firstSQLToken(statement)
	return head == "insert" || head == "replace"
}

func firstSQLToken(statement string) string {
	index := 0
	for {
		for index < len(statement) && isSQLSpace(statement[index]) {
			index++
		}
		if index+1 < len(statement) && statement[index] == '-' && statement[index+1] == '-' {
			index += 2
			for index < len(statement) && statement[index] != '\n' {
				index++
			}
			continue
		}
		if index+1 < len(statement) && statement[index] == '/' && statement[index+1] == '*' {
			index += 2
			for index+1 < len(statement) && !(statement[index] == '*' && statement[index+1] == '/') {
				index++
			}
			if index+1 < len(statement) {
				index += 2
			}
			continue
		}
		break
	}

	start := index
	for index < len(statement) && !isSQLSpace(statement[index]) && statement[index] != ';' {
		index++
	}
	return strings.ToLower(statement[start:index])
}

func isSQLSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func printTable(out io.Writer, headers []string, widths []int, rows [][]string) {
	for index, header := range headers {
		if index > 0 {
			_, _ = fmt.Fprintf(out, "  ")
		}
		_, _ = fmt.Fprintf(out, "%-*s", widths[index], header)
	}
	_, _ = fmt.Fprintf(out, "\n")

	for index, header := range headers {
		if index > 0 {
			_, _ = fmt.Fprintf(out, "  ")
		}
		_, _ = fmt.Fprintf(out, "%s", strings.Repeat("-", len(header)))
		if widths[index] > len(header) {
			_, _ = fmt.Fprintf(out, "%s", strings.Repeat("-", widths[index]-len(header)))
		}
	}
	_, _ = fmt.Fprintf(out, "\n")

	for _, row := range rows {
		for index, value := range row {
			if index > 0 {
				_, _ = fmt.Fprintf(out, "  ")
			}
			_, _ = fmt.Fprintf(out, "%-*s", widths[index], value)
		}
		_, _ = fmt.Fprintf(out, "\n")
	}
}

func formatValue(value any) string {
	switch current := value.(type) {
	case nil:
		return "NULL"
	case []byte:
		return "x'" + strings.ToUpper(hex.EncodeToString(current)) + "'"
	case string:
		return escapeCell(current)
	case bool:
		if current {
			return "true"
		}
		return "false"
	default:
		return escapeCell(fmt.Sprintf("%v", value))
	}
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return value
}
