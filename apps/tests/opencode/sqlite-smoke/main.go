package main

import (
	"database/sql"
	"errors"
	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"kos"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

const diskName = "/hd0/1/opencode-sqlite-smoke.db"
const controlName = "opencode-sqlite-process-test"

func waitState(state *uint32, want uint32) {
	for i := 0; i < 2000 && atomic.LoadUint32(state) != want; i++ {
		kos.Sleep(1)
	}
	check(atomic.LoadUint32(state) == want, "SQLite process handshake")
}

func child() {
	runtime.LockOSThread()
	address, size := kos.OpenNamedMemory(controlName, 0, kos.SharedMemoryOpen|kos.SharedMemoryWrite)
	check(address != 0 && size == 4096, "child control area")
	state := (*uint32)(unsafe.Pointer(address))
	db, err := sql.Open("sqlite3", "file:"+diskName+"?_pragma=busy_timeout(0)")
	must(err)
	db.SetMaxOpenConns(1)
	var count int
	err = db.QueryRow("SELECT count(*) FROM messages").Scan(&count)
	check(errors.Is(err, sqlite3.BUSY), "other process must not share the private WAL index")
	must(db.Close())
	atomic.StoreUint32(state, 1)
	waitState(state, 2)
	db = open(diskName)
	must(db.QueryRow("SELECT count(*) FROM messages").Scan(&count))
	check(count == 2, "child sees committed rows")
	tx, err := db.Begin()
	must(err)
	_, err = tx.Exec("UPDATE messages SET text='abandoned transaction' WHERE id=1")
	must(err)
	atomic.StoreUint32(state, 3)
	kos.DebugString("SQLITE_CHILD_OWNER_EXIT")
	// Deliberately leave the transaction, database and native lock mappings
	// open. Process termination must release the kernel mappings without an
	// application-level Close or Rollback and preserve committed data.
}

func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func open(name string) *sql.DB {
	db, err := sql.Open("sqlite3", name)
	must(err)
	db.SetMaxOpenConns(1)
	must(db.Ping())
	return db
}

func exercise(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE messages(id INTEGER PRIMARY KEY, text TEXT NOT NULL, data BLOB)")
	must(err)
	_, err = db.Exec("INSERT INTO messages(text,data) VALUES(?,?)", "Привет, KolibriOS", []byte{0, 1, 42, 255})
	must(err)
	tx, err := db.Begin()
	must(err)
	_, err = tx.Exec("INSERT INTO messages(text) VALUES(?)", "rolled back")
	must(err)
	must(tx.Rollback())
	verify(db)
}

func verify(db *sql.DB) {
	runtime.GC()
	var count int
	must(db.QueryRow("SELECT count(*) FROM messages").Scan(&count))
	check(count == 1, "transaction rollback")
	var text string
	var data []byte
	must(db.QueryRow("SELECT text,data FROM messages WHERE id=1").Scan(&text, &data))
	check(text == "Привет, KolibriOS", "UTF-8 database text")
	check(len(data) == 4 && data[0] == 0 && data[1] == 1 && data[2] == 42 && data[3] == 255, "database blob")
}

func main() {
	if kos.LoaderParameters() == "sqlite-child" {
		child()
		return
	}
	console, ok := kos.OpenConsole("OpenCode SQLite port test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_SQLITE_START")
	check(!time.Now().UTC().IsDST(), "UTC daylight saving flag")
	check(!time.Now().In(time.FixedZone("UTC+3", 10800)).IsDST(), "fixed zone daylight saving flag")
	memory := open(":memory:")
	exercise(memory)
	must(memory.Close())
	kos.DebugString("SQLITE_MEMORY_OK")
	const name = diskName
	disk := open(name)
	exercise(disk)
	must(disk.Close())
	disk = open(name)
	verify(disk)
	var journal string
	must(disk.QueryRow("PRAGMA journal_mode=WAL").Scan(&journal))
	check(journal == "wal", "WAL journal mode")
	second := open(name)
	_, err := second.Exec("PRAGMA busy_timeout=0")
	must(err)
	verify(second)
	tx, err := disk.Begin()
	must(err)
	_, err = tx.Exec("UPDATE messages SET text=? WHERE id=1", "uncommitted")
	must(err)
	// WAL readers continue to see the committed version while a writer waits.
	verify(second)
	_, err = second.Exec("UPDATE messages SET text=? WHERE id=1", "conflicting writer")
	check(errors.Is(err, sqlite3.BUSY), "conflicting writer must return SQLITE_BUSY")
	must(tx.Rollback())
	verify(second)
	_, err = second.Exec("INSERT INTO messages(text) VALUES(?)", "committed")
	must(err)
	must(second.Close())
	must(disk.Close())
	kos.DebugString("SQLITE_DISK_OK")
	disk = open(name)
	var count int
	must(disk.QueryRow("SELECT count(*) FROM messages").Scan(&count))
	check(count == 2, "WAL committed data persists")
	runtime.LockOSThread()
	address, status := kos.OpenNamedMemory(controlName, 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
	check(address != 0 && status == 0, "parent control area")
	state := (*uint32)(unsafe.Pointer(address))
	pid, started := kos.StartApplication(kos.LoaderPath(), "sqlite-child", false)
	check(pid > 0 && started == kos.FileSystemOK, "start SQLite child")
	waitState(state, 1)
	must(disk.Close())
	runtime.GC()
	atomic.StoreUint32(state, 2)
	for i := 0; i < 2000 && kos.ThreadSlotByIdentifier(pid) != 0; i++ {
		kos.Sleep(1)
	}
	check(kos.ThreadSlotByIdentifier(pid) == 0 && atomic.LoadUint32(state) == 3, "database owner exited")
	disk = open(name)
	must(disk.QueryRow("SELECT count(*) FROM messages").Scan(&count))
	check(count == 2, "owner exit preserves committed rows")
	var text string
	must(disk.QueryRow("SELECT text FROM messages WHERE id=1").Scan(&text))
	check(text == "Привет, KolibriOS", "owner exit abandons uncommitted WAL transaction")
	_, err = disk.Exec("INSERT INTO messages(text) VALUES('after owner exit')")
	must(err)
	must(disk.Close())
	kos.CloseNamedMemory(controlName)
	runtime.UnlockOSThread()
	kos.DebugString("SQLITE_PROCESS_LOCKS_OK")
	kos.DebugString("SQLITE_WAL_LOCKS_OK")
	kos.DebugString("OPENCODE_SQLITE_PASS")
	console.WriteString("OPENCODE_SQLITE_PASS\n")
}
