//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package sqlite3

import (
	"context"
	sqlpkg "database/sql"
	sqldriver "database/sql/driver"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"
)

const DriverName = "sqlite3"

const (
	sqliteOK       = 0
	sqliteError    = 1
	sqliteInternal = 2
	sqlitePerm     = 3
	sqliteAbort    = 4
	sqliteBusy     = 5
	sqliteLocked   = 6
	sqliteNoMem    = 7
	sqliteReadOnly = 8
	sqliteInterrupt = 9
	sqliteIOErr    = 10
	sqliteCorrupt  = 11
	sqliteFull     = 13
	sqliteCantOpen = 14
	sqliteMisuse   = 21
	sqliteRange    = 25
	sqliteDone     = 101
	sqliteRow      = 100
)

const (
	sqliteInteger = 1
	sqliteFloat   = 2
	sqliteText    = 3
	sqliteBlob    = 4
	sqliteNull    = 5
)

const (
	sqliteOpenReadOnly  = 0x00000001
	sqliteOpenReadWrite = 0x00000002
	sqliteOpenCreate    = 0x00000004
)

const (
	isolationLevelDefault         sqldriver.IsolationLevel = 0
	isolationLevelReadUncommitted sqldriver.IsolationLevel = 1
	isolationLevelSerializable    sqldriver.IsolationLevel = 6
)

type cSQLite3 struct{}
type cSQLiteStmt struct{}

func allocCString(value string) *byte __asm__("runtime_alloc_cstring")
func freeCString(ptr *byte) __asm__("runtime_free_cstring")
func pointerValue(ptr *byte) uint32 __asm__("runtime_pointer_value")
func copyBytesRaw(ptr uint32, size uint32) []byte __asm__("runtime_copy_bytes")
func cstringToStringRaw(ptr uint32) string __asm__("runtime_cstring_to_gostring")

func sqlite3KosInitialize() int32 __asm__("sqlite3_kos_initialize")
func sqlite3KosOpen(filename *byte, db **cSQLite3, flags int32) int32 __asm__("sqlite3_kos_open")
func sqlite3KosPrepare(db *cSQLite3, sql *byte, stmt **cSQLiteStmt) int32 __asm__("sqlite3_kos_prepare")
func sqlite3KosExec(db *cSQLite3, sql *byte) int32 __asm__("sqlite3_kos_exec")
func sqlite3KosBindText(stmt *cSQLiteStmt, index int32, text *byte, size int32) int32 __asm__("sqlite3_kos_bind_text")
func sqlite3KosBindBlob(stmt *cSQLiteStmt, index int32, data *byte, size int32) int32 __asm__("sqlite3_kos_bind_blob")
func sqlite3CloseV2(db *cSQLite3) int32 __asm__("sqlite3_close_v2")
func sqlite3Errmsg(db *cSQLite3) *byte __asm__("sqlite3_errmsg")
func sqlite3Errstr(code int32) *byte __asm__("sqlite3_errstr")
func sqlite3Changes64(db *cSQLite3) int64 __asm__("sqlite3_changes64")
func sqlite3LastInsertRowid(db *cSQLite3) int64 __asm__("sqlite3_last_insert_rowid")
func sqlite3BindParameterCount(stmt *cSQLiteStmt) int32 __asm__("sqlite3_bind_parameter_count")
func sqlite3BindParameterIndex(stmt *cSQLiteStmt, name *byte) int32 __asm__("sqlite3_bind_parameter_index")
func sqlite3BindNull(stmt *cSQLiteStmt, index int32) int32 __asm__("sqlite3_bind_null")
func sqlite3BindInt64(stmt *cSQLiteStmt, index int32, value int64) int32 __asm__("sqlite3_bind_int64")
func sqlite3BindDouble(stmt *cSQLiteStmt, index int32, value float64) int32 __asm__("sqlite3_bind_double")
func sqlite3Step(stmt *cSQLiteStmt) int32 __asm__("sqlite3_step")
func sqlite3Finalize(stmt *cSQLiteStmt) int32 __asm__("sqlite3_finalize")
func sqlite3Reset(stmt *cSQLiteStmt) int32 __asm__("sqlite3_reset")
func sqlite3ClearBindings(stmt *cSQLiteStmt) int32 __asm__("sqlite3_clear_bindings")
func sqlite3ColumnCount(stmt *cSQLiteStmt) int32 __asm__("sqlite3_column_count")
func sqlite3ColumnName(stmt *cSQLiteStmt, index int32) *byte __asm__("sqlite3_column_name")
func sqlite3ColumnDecltype(stmt *cSQLiteStmt, index int32) *byte __asm__("sqlite3_column_decltype")
func sqlite3ColumnType(stmt *cSQLiteStmt, index int32) int32 __asm__("sqlite3_column_type")
func sqlite3ColumnInt64(stmt *cSQLiteStmt, index int32) int64 __asm__("sqlite3_column_int64")
func sqlite3ColumnDouble(stmt *cSQLiteStmt, index int32) float64 __asm__("sqlite3_column_double")
func sqlite3ColumnBytes(stmt *cSQLiteStmt, index int32) int32 __asm__("sqlite3_column_bytes")
func sqlite3ColumnText(stmt *cSQLiteStmt, index int32) *byte __asm__("sqlite3_column_text")
func sqlite3ColumnBlob(stmt *cSQLiteStmt, index int32) *byte __asm__("sqlite3_column_blob")

var driverInstance sqliteDriver

var (
	int64Type   = reflect.TypeOf(int64(0))
	float64Type = reflect.TypeOf(float64(0))
	boolType    = reflect.TypeOf(false)
	stringType  = reflect.TypeOf("")
	bytesType   = reflect.TypeOf([]byte(nil))
	anyType     = reflect.TypeOf((*interface{})(nil)).Elem()
)

type sqliteDriver struct{}

type connector struct {
	name        string
	cleanupPath string
}

type conn struct {
	db *cSQLite3
}

type stmt struct {
	conn     *conn
	stmt     *cSQLiteStmt
	numInput int
}

type rows struct {
	conn            *conn
	stmt            *cSQLiteStmt
	ctx             context.Context
	finalizeOnClose bool
	closed          bool
	columns         []string
	declTypes       []string
}

type tx struct {
	conn                   *conn
	resetReadUncommitted   bool
	done                   bool
}

type result struct {
	lastInsertID int64
	rowsAffected int64
}

type emptyRows struct{}

func init() {
	sqlpkg.Register(DriverName, &driverInstance)
}

func Open(name string) (*sqlpkg.DB, error) {
	db, err := sqlpkg.Open(DriverName, name)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (d *sqliteDriver) Open(name string) (sqldriver.Conn, error) {
	connector, err := d.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return connector.Connect(context.Background())
}

func (d *sqliteDriver) OpenConnector(name string) (sqldriver.Connector, error) {
	return &connector{name: name}, nil
}

func (c *connector) Connect(ctx context.Context) (sqldriver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if rc := sqlite3KosInitialize(); rc != sqliteOK {
		return nil, sqliteErrorf(rc, nil, "initialize")
	}

	var db *cSQLite3
	cname, err := withCString(c.name)
	if err != nil {
		return nil, err
	}
	defer freeCString(cname)

	rc := sqlite3KosOpen(cname, &db, sqliteOpenReadWrite|sqliteOpenCreate)
	if rc != sqliteOK {
		openErr := sqliteErrorf(rc, db, "open")
		if db != nil {
			_ = sqlite3CloseV2(db)
		}
		return nil, openErr
	}

	return &conn{db: db}, nil
}

func (c *connector) Driver() sqldriver.Driver {
	return &driverInstance
}

func (c *connector) Close() error {
	if c.cleanupPath == "" {
		return nil
	}
	err := os.Remove(c.cleanupPath)
	c.cleanupPath = ""
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *conn) Prepare(query string) (sqldriver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *conn) PrepareContext(ctx context.Context, query string) (sqldriver.Stmt, error) {
	st, err := c.prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("sqlite3 prepare: empty query")
	}
	return &stmt{
		conn:     c,
		stmt:     st,
		numInput: int(sqlite3BindParameterCount(st)),
	}, nil
}

func (c *conn) Close() error {
	if c.db == nil {
		return nil
	}
	rc := sqlite3CloseV2(c.db)
	c.db = nil
	if rc != sqliteOK {
		return sqliteErrorf(rc, nil, "close")
	}
	return nil
}

func (c *conn) Begin() (sqldriver.Tx, error) {
	return c.BeginTx(context.Background(), sqldriver.TxOptions{})
}

func (c *conn) BeginTx(ctx context.Context, opts sqldriver.TxOptions) (sqldriver.Tx, error) {
	if err := c.ensureOpen(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if opts.ReadOnly {
		return nil, fmt.Errorf("sqlite3 begin: read-only transactions are not supported")
	}

	resetReadUncommitted := false
	switch opts.Isolation {
	case isolationLevelDefault, isolationLevelSerializable:
	case isolationLevelReadUncommitted:
		resetReadUncommitted = true
		if err := c.execSQL("PRAGMA read_uncommitted = 1"); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("sqlite3 begin: unsupported isolation level %d", int(opts.Isolation))
	}

	if err := c.execSQL("BEGIN"); err != nil {
		if resetReadUncommitted {
			_ = c.execSQL("PRAGMA read_uncommitted = 0")
		}
		return nil, err
	}

	return &tx{
		conn:                 c,
		resetReadUncommitted: resetReadUncommitted,
	}, nil
}

func (c *conn) Ping(ctx context.Context) error {
	if err := c.ensureOpen(); err != nil {
		return err
	}
	return ctx.Err()
}

func (c *conn) ResetSession(ctx context.Context) error {
	if err := c.ensureOpen(); err != nil {
		return err
	}
	return ctx.Err()
}

func (c *conn) IsValid() bool {
	return c.db != nil
}

func (c *conn) Exec(query string, args []sqldriver.Value) (sqldriver.Result, error) {
	return c.ExecContext(context.Background(), query, valuesToNamed(args))
}

func (c *conn) Query(query string, args []sqldriver.Value) (sqldriver.Rows, error) {
	return c.QueryContext(context.Background(), query, valuesToNamed(args))
}

func (c *conn) ExecContext(ctx context.Context, query string, args []sqldriver.NamedValue) (sqldriver.Result, error) {
	st, err := c.prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return sqldriver.ResultNoRows, nil
	}
	defer sqlite3Finalize(st)
	return c.execPrepared(ctx, st, args)
}

func (c *conn) QueryContext(ctx context.Context, query string, args []sqldriver.NamedValue) (sqldriver.Rows, error) {
	st, err := c.prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return emptyRows{}, nil
	}
	if err := bindNamedValues(c.db, st, args); err != nil {
		_ = sqlite3Finalize(st)
		return nil, err
	}
	return &rows{
		conn:            c,
		stmt:            st,
		ctx:             ctx,
		finalizeOnClose: true,
	}, nil
}

func (c *conn) CheckNamedValue(value *sqldriver.NamedValue) error {
	converted, err := sqldriver.DefaultParameterConverter.ConvertValue(value.Value)
	if err != nil {
		return err
	}
	value.Value = converted
	return nil
}

func (c *conn) prepare(ctx context.Context, query string) (*cSQLiteStmt, error) {
	if err := c.ensureOpen(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	cquery, err := withCString(query)
	if err != nil {
		return nil, err
	}
	defer freeCString(cquery)

	var st *cSQLiteStmt
	rc := sqlite3KosPrepare(c.db, cquery, &st)
	if rc != sqliteOK {
		return nil, sqliteErrorf(rc, c.db, "prepare")
	}
	return st, nil
}

func (c *conn) execPrepared(ctx context.Context, st *cSQLiteStmt, args []sqldriver.NamedValue) (sqldriver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := bindNamedValues(c.db, st, args); err != nil {
		return nil, err
	}
	defer resetStatement(st)

	rc := sqlite3Step(st)
	switch rc {
	case sqliteDone:
		return result{
			lastInsertID: sqlite3LastInsertRowid(c.db),
			rowsAffected: sqlite3Changes64(c.db),
		}, nil
	case sqliteRow:
		return nil, fmt.Errorf("sqlite3 exec: statement returned rows")
	default:
		return nil, sqliteErrorf(rc, c.db, "exec")
	}
}

func (c *conn) execSQL(query string) error {
	if err := c.ensureOpen(); err != nil {
		return err
	}
	cquery, err := withCString(query)
	if err != nil {
		return err
	}
	defer freeCString(cquery)

	rc := sqlite3KosExec(c.db, cquery)
	if rc != sqliteOK {
		return sqliteErrorf(rc, c.db, "exec")
	}
	return nil
}

func (c *conn) ensureOpen() error {
	if c.db == nil {
		return sqldriver.ErrBadConn
	}
	return nil
}

func (s *stmt) Close() error {
	if s.stmt == nil {
		return nil
	}
	rc := sqlite3Finalize(s.stmt)
	s.stmt = nil
	if rc != sqliteOK {
		return sqliteErrorf(rc, nil, "finalize")
	}
	return nil
}

func (s *stmt) NumInput() int {
	return s.numInput
}

func (s *stmt) Exec(args []sqldriver.Value) (sqldriver.Result, error) {
	return s.ExecContext(context.Background(), valuesToNamed(args))
}

func (s *stmt) Query(args []sqldriver.Value) (sqldriver.Rows, error) {
	return s.QueryContext(context.Background(), valuesToNamed(args))
}

func (s *stmt) ExecContext(ctx context.Context, args []sqldriver.NamedValue) (sqldriver.Result, error) {
	if err := s.conn.ensureOpen(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resetStatement(s.stmt)
	return s.conn.execPrepared(ctx, s.stmt, args)
}

func (s *stmt) QueryContext(ctx context.Context, args []sqldriver.NamedValue) (sqldriver.Rows, error) {
	if err := s.conn.ensureOpen(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resetStatement(s.stmt)
	if err := bindNamedValues(s.conn.db, s.stmt, args); err != nil {
		return nil, err
	}
	return &rows{
		conn:            s.conn,
		stmt:            s.stmt,
		ctx:             ctx,
		finalizeOnClose: false,
	}, nil
}

func (s *stmt) CheckNamedValue(value *sqldriver.NamedValue) error {
	return s.conn.CheckNamedValue(value)
}

func (r *rows) Columns() []string {
	if r.columns != nil {
		return append([]string(nil), r.columns...)
	}

	count := int(sqlite3ColumnCount(r.stmt))
	r.columns = make([]string, count)
	for index := 0; index < count; index++ {
		r.columns[index] = cstringToString(sqlite3ColumnName(r.stmt, int32(index)))
	}
	return append([]string(nil), r.columns...)
}

func (r *rows) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true

	var rc int32
	if r.stmt != nil {
		if r.finalizeOnClose {
			rc = sqlite3Finalize(r.stmt)
		} else {
			resetStatement(r.stmt)
			rc = sqliteOK
		}
		r.stmt = nil
	}
	if rc != sqliteOK {
		return sqliteErrorf(rc, nil, "rows close")
	}
	return nil
}

func (r *rows) Next(dest []sqldriver.Value) error {
	if r.closed {
		return io.EOF
	}
	if r.ctx != nil {
		if err := r.ctx.Err(); err != nil {
			return err
		}
	}

	rc := sqlite3Step(r.stmt)
	switch rc {
	case sqliteRow:
		for index := 0; index < len(dest); index++ {
			dest[index] = r.columnValue(index)
		}
		return nil
	case sqliteDone:
		return io.EOF
	default:
		return sqliteErrorf(rc, r.conn.db, "step")
	}
}

func (r *rows) ColumnTypeScanType(index int) reflect.Type {
	typeName := r.ColumnTypeDatabaseTypeName(index)
	switch {
	case strings.Contains(typeName, "BOOL"):
		return boolType
	case strings.Contains(typeName, "INT"):
		return int64Type
	case strings.Contains(typeName, "REAL"), strings.Contains(typeName, "FLOA"), strings.Contains(typeName, "DOUB"), strings.Contains(typeName, "NUM"):
		return float64Type
	case strings.Contains(typeName, "BLOB"):
		return bytesType
	case strings.Contains(typeName, "CHAR"), strings.Contains(typeName, "CLOB"), strings.Contains(typeName, "TEXT"), strings.Contains(typeName, "DATE"), strings.Contains(typeName, "TIME"):
		return stringType
	default:
		return anyType
	}
}

func (r *rows) ColumnTypeDatabaseTypeName(index int) string {
	if r.declTypes == nil {
		count := int(sqlite3ColumnCount(r.stmt))
		r.declTypes = make([]string, count)
		for current := 0; current < count; current++ {
			r.declTypes[current] = normalizeDeclType(cstringToString(sqlite3ColumnDecltype(r.stmt, int32(current))))
		}
	}
	if index < 0 || index >= len(r.declTypes) {
		return ""
	}
	return r.declTypes[index]
}

func (r *rows) columnValue(index int) sqldriver.Value {
	columnType := sqlite3ColumnType(r.stmt, int32(index))
	switch columnType {
	case sqliteNull:
		return nil
	case sqliteInteger:
		return sqlite3ColumnInt64(r.stmt, int32(index))
	case sqliteFloat:
		return sqlite3ColumnDouble(r.stmt, int32(index))
	case sqliteText:
		size := sqlite3ColumnBytes(r.stmt, int32(index))
		if size == 0 {
			return ""
		}
		return string(copyFromPointer(sqlite3ColumnText(r.stmt, int32(index)), size))
	case sqliteBlob:
		size := sqlite3ColumnBytes(r.stmt, int32(index))
		if size == 0 {
			return []byte{}
		}
		return copyFromPointer(sqlite3ColumnBlob(r.stmt, int32(index)), size)
	default:
		size := sqlite3ColumnBytes(r.stmt, int32(index))
		if size == 0 {
			return ""
		}
		return string(copyFromPointer(sqlite3ColumnText(r.stmt, int32(index)), size))
	}
}

func (t *tx) Commit() error {
	if t.done {
		return fmt.Errorf("sqlite3 commit: transaction is already closed")
	}
	t.done = true
	err := t.conn.execSQL("COMMIT")
	if t.resetReadUncommitted {
		_ = t.conn.execSQL("PRAGMA read_uncommitted = 0")
	}
	return err
}

func (t *tx) Rollback() error {
	if t.done {
		return fmt.Errorf("sqlite3 rollback: transaction is already closed")
	}
	t.done = true
	err := t.conn.execSQL("ROLLBACK")
	if t.resetReadUncommitted {
		_ = t.conn.execSQL("PRAGMA read_uncommitted = 0")
	}
	return err
}

func (r result) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

func (emptyRows) Columns() []string {
	return nil
}

func (emptyRows) Close() error {
	return nil
}

func (emptyRows) Next(dest []sqldriver.Value) error {
	return io.EOF
}

func valuesToNamed(values []sqldriver.Value) []sqldriver.NamedValue {
	if len(values) == 0 {
		return nil
	}
	result := make([]sqldriver.NamedValue, len(values))
	for index := 0; index < len(values); index++ {
		result[index] = sqldriver.NamedValue{
			Ordinal: index + 1,
			Value:   values[index],
		}
	}
	return result
}

func bindNamedValues(db *cSQLite3, st *cSQLiteStmt, values []sqldriver.NamedValue) error {
	for _, value := range values {
		index := int32(value.Ordinal)
		if value.Name != "" {
			namedIndex := bindParameterIndex(st, ":"+value.Name)
			if namedIndex == 0 {
				namedIndex = bindParameterIndex(st, "@"+value.Name)
			}
			if namedIndex == 0 {
				namedIndex = bindParameterIndex(st, "$"+value.Name)
			}
			if namedIndex != 0 {
				index = namedIndex
			}
		}
		if index <= 0 {
			return fmt.Errorf("sqlite3 bind: invalid parameter index")
		}

		var rc int32
		switch current := value.Value.(type) {
		case nil:
			rc = sqlite3BindNull(st, index)
		case int64:
			rc = sqlite3BindInt64(st, index, current)
		case float64:
			rc = sqlite3BindDouble(st, index, current)
		case bool:
			if current {
				rc = sqlite3BindInt64(st, index, 1)
			} else {
				rc = sqlite3BindInt64(st, index, 0)
			}
		case string:
			text, err := withCString(current)
			if err != nil {
				return err
			}
			rc = sqlite3KosBindText(st, index, text, int32(len(current)))
			freeCString(text)
		case []byte:
			if len(current) == 0 {
				rc = sqlite3KosBindBlob(st, index, nil, 0)
			} else {
				rc = sqlite3KosBindBlob(st, index, &current[0], int32(len(current)))
			}
		case time.Time:
			formatted := current.Format("2006-01-02 15:04:05.999999999Z07:00")
			text, err := withCString(formatted)
			if err != nil {
				return err
			}
			rc = sqlite3KosBindText(st, index, text, int32(len(formatted)))
			freeCString(text)
		default:
			return fmt.Errorf("sqlite3 bind: unsupported parameter type %T", value.Value)
		}

		if rc != sqliteOK {
			return sqliteErrorf(rc, db, "bind")
		}
	}
	return nil
}

func bindParameterIndex(st *cSQLiteStmt, name string) int32 {
	cname, err := withCString(name)
	if err != nil {
		return 0
	}
	defer freeCString(cname)
	return sqlite3BindParameterIndex(st, cname)
}

func resetStatement(st *cSQLiteStmt) {
	if st == nil {
		return
	}
	_ = sqlite3ClearBindings(st)
	_ = sqlite3Reset(st)
}

func withCString(value string) (*byte, error) {
	ptr := allocCString(value)
	if ptr == nil {
		return nil, fmt.Errorf("sqlite3: out of memory")
	}
	return ptr, nil
}

func cstringToString(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	return cstringToStringRaw(pointerValue(ptr))
}

func copyFromPointer(ptr *byte, size int32) []byte {
	if ptr == nil || size <= 0 {
		return []byte{}
	}
	return copyBytesRaw(pointerValue(ptr), uint32(size))
}

func sqliteErrorf(code int32, db *cSQLite3, op string) error {
	message := ""
	if db != nil {
		message = cstringToString(sqlite3Errmsg(db))
	}
	if message == "" {
		message = cstringToString(sqlite3Errstr(code))
	}
	if message == "" {
		message = "sqlite error"
	}
	return fmt.Errorf("sqlite3 %s: %s (code %d)", op, message, int(code))
}

func normalizeDeclType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if index := strings.IndexByte(value, '('); index >= 0 {
		value = value[:index]
	}
	return strings.ToUpper(strings.TrimSpace(value))
}
