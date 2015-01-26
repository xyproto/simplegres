// Simple way to use a MySQL/MariaDB database
package db

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"strings"
)

type Host struct {
	db     *sql.DB
	dbname string
}

// Common for each of the db datastructures used here
type dbDatastructure struct {
	host  *Host
	table string
}

type (
	List     dbDatastructure
	Set      dbDatastructure
	HashMap  dbDatastructure
	KeyValue dbDatastructure
)

const (
	// Version number. Stable API within major version numbers.
	Version = 1.0

	// The default "username:password@host:port/database" that the database is running at
	defaultDatabaseServer = ""     // "username:password@server:port/"
	defaultDatabaseName   = "test" // "main"
	defaultStringLength   = 42     // using VARCHAR, so this will be expanded up to 65535 characters as needed, unless mysql strict mode is enabled
	defaultPort           = 3306

	listColName = "list_col"
	setColName  = "set_col"
	hashColName = "hash_col"
	kvColName   = "kv_col"
)

// Test if the local database server is up and running.
func TestConnection() (err error) {
	return TestConnectionHost(defaultDatabaseServer)
}

// Test if a given database server is up and running.
// connectionString may be on the form "username:password@host:port/database".
func TestConnectionHost(connectionString string) (err error) {
	newConnectionString, _ := rebuildConnectionString(connectionString)
	// Connect to the given host:port
	db, err := sql.Open("mysql", newConnectionString)
	defer db.Close()
	err = db.Ping()
	if Verbose {
		if err != nil {
			log.Println("Ping: failed")
		} else {
			log.Println("Ping: ok")
		}
	}
	return err
}

/* --- Host functions --- */

// Create a new database connection.
// connectionString may be on the form "username:password@host:port/database".
func NewHost(connectionString string) *Host {

	newConnectionString, dbname := rebuildConnectionString(connectionString)

	db, err := sql.Open("mysql", newConnectionString)
	if err != nil {
		log.Fatalln("Could not connect to " + newConnectionString + "!")
	}
	host := &Host{db, dbname}
	if err := db.Ping(); err != nil {
		log.Fatalln("Database does not reply to ping: " + err.Error())
	}
	if err := host.createDatabase(); err != nil {
		log.Fatalln("Could not create database " + host.dbname + ": " + err.Error())
	}
	if err := host.useDatabase(); err != nil {
		panic("Could not use database " + host.dbname + ": " + err.Error())
	}
	return host
}

// The default database connection
func New() *Host {
	connectionString := defaultDatabaseServer + defaultDatabaseName
	if !strings.HasSuffix(defaultDatabaseServer, "/") {
		connectionString = defaultDatabaseServer + "/" + defaultDatabaseName
	}
	return NewHost(connectionString)
}

// Select a different database. Create the database if needed.
func (host *Host) SelectDatabase(dbname string) error {
	host.dbname = dbname
	if err := host.createDatabase(); err != nil {
		return err
	}
	if err := host.useDatabase(); err != nil {
		return err
	}
	return nil
}

// Will create the database if it does not already exist.
func (host *Host) createDatabase() error {
	if _, err := host.db.Exec("CREATE DATABASE IF NOT EXISTS " + host.dbname + " CHARACTER SET = utf8"); err != nil {
		return err
	}
	if Verbose {
		log.Println("Created database " + host.dbname)
	}
	return nil
}

// Use the host.dbname database.
func (host *Host) useDatabase() error {
	if _, err := host.db.Exec("USE " + host.dbname); err != nil {
		return err
	}
	if Verbose {
		log.Println("Using database " + host.dbname)
	}
	return nil
}

// Close the connection.
func (host *Host) Close() {
	host.db.Close()
}

/* --- List functions --- */

// Create a new list. Lists are ordered.
func NewList(host *Host, name string) *List {
	l := &List{host, name}
	// list is the name of the column
	if _, err := l.host.db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (id INT PRIMARY KEY AUTO_INCREMENT, " + listColName + " VARCHAR(" + strconv.Itoa(defaultStringLength) + "))"); err != nil {
		// This is more likely to happen at the start of the program,
		// hence the panic.
		panic("Could not create table " + name + ": " + err.Error())
	}
	if Verbose {
		log.Println("Created table " + name + " in database " + host.dbname)
	}
	return l
}

// Add an element to the list
func (rl *List) Add(value string) error {
	// list is the name of the column
	_, err := rl.host.db.Exec("INSERT INTO "+rl.table+" ("+listColName+") VALUES (?)", value)
	return err
}

// Get all elements of a list
func (rl *List) GetAll() ([]string, error) {
	rows, err := rl.host.db.Query("SELECT " + listColName + " FROM " + rl.table + " ORDER BY id")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	var (
		values []string
		value  string
	)
	for rows.Next() {
		err = rows.Scan(&value)
		values = append(values, value)
		if err != nil {
			panic(err.Error())
		}
	}
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}
	return values, nil
}

// Get the last element of a list
func (rl *List) GetLast() (string, error) {
	// Fetches the item with the largest id.
	// Faster than "ORDER BY id DESC limit 1" for large tables.
	rows, err := rl.host.db.Query("SELECT " + listColName + " FROM " + rl.table + " WHERE id = (SELECT MAX(id) FROM " + rl.table + ")")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	var value string
	// Get the value. Will only loop once.
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			panic(err.Error())
		}
	}
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}
	return value, nil
}

// Get the last N elements of a list
func (rl *List) GetLastN(n int) ([]string, error) {
	rows, err := rl.host.db.Query("SELECT " + listColName + " FROM (SELECT * FROM " + rl.table + " ORDER BY id DESC limit " + strconv.Itoa(n) + ")sub ORDER BY id ASC")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	var (
		values []string
		value  string
	)
	for rows.Next() {
		err = rows.Scan(&value)
		values = append(values, value)
		if err != nil {
			panic(err.Error())
		}
	}
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}
	if len(values) < n {
		return []string{}, errors.New("Too few elements in table at GetLastN")
	}
	return values, nil
}

// Remove this list
func (rl *List) Remove() error {
	// Remove the table
	_, err := rl.host.db.Exec("DROP TABLE " + rl.table)
	return err
}

// Clear the list contents
func (rl *List) Clear() error {
	// Clear the table
	_, err := rl.host.db.Exec("TRUNCATE TABLE " + rl.table)
	return err
}

/* --- Set functions --- */

// Create a new set
func NewSet(host *Host, name string) *Set {
	s := &Set{host, name}
	// list is the name of the column
	if _, err := s.host.db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (" + setColName + " VARCHAR(" + strconv.Itoa(defaultStringLength) + "))"); err != nil {
		// This is more likely to happen at the start of the program, hence the panic.
		panic("Could not create table " + name + ": " + err.Error())
	}
	if Verbose {
		log.Println("Created table " + name + " in database " + host.dbname)
	}
	return s
}

// Add an element to the set
func (s *Set) Add(value string) error {
	// Check if the value is not already there before adding
	has, err := s.Has(value)
	if !has && (err == nil) {
		// set is the name of the column
		_, err = s.host.db.Exec("INSERT INTO "+s.table+" ("+setColName+") VALUES (?)", value)
	}
	return err
}

// Check if a given value is in the set
func (s *Set) Has(value string) (bool, error) {
	rows, err := s.host.db.Query("SELECT " + setColName + " FROM " + s.table + " WHERE " + setColName + " = '" + value + "'")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	var scanValue string
	// Get the value. Should not loop more than once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&scanValue)
		if err != nil {
			panic(err.Error())
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}
	if counter > 1 {
		panic("Duplicate members in set! " + value)
	}
	return counter > 0, nil
}

// Get all elements of the set
func (s *Set) GetAll() ([]string, error) {
	rows, err := s.host.db.Query("SELECT " + setColName + " FROM " + s.table)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	var (
		values []string
		value  string
	)
	for rows.Next() {
		err = rows.Scan(&value)
		values = append(values, value)
		if err != nil {
			panic(err.Error())
		}
	}
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}
	return values, nil
}

// Remove an element from the set
func (s *Set) Del(value string) error {
	// Remove a value from the table
	_, err := s.host.db.Exec("DELETE FROM " + s.table + " WHERE " + setColName + " = " + value)
	return err
}

// Remove this set
func (s *Set) Remove() error {
	// Remove the table
	_, err := s.host.db.Exec("DROP TABLE " + s.table)
	return err
}

// Clear the list contents
func (s *Set) Clear() error {
	// Clear the table
	_, err := s.host.db.Exec("TRUNCATE TABLE " + s.table)
	return err
}

///* --- HashMap functions --- */
//
//// Create a new hashmap
//func NewHashMap(host *sql.DB, table string) *HashMap {
//	return &HashMap{host, table, defaultDatabaseName}
//}
//
//// Select a different database
//func (rh *HashMap) SelectDatabase(dbname string) {
//	rh.dbname = dbname
//}
//
//// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
//func (rh *HashMap) Set(elementid, key, value string) error {
//	db := rh.host.Get(rh.dbname)
//	_, err := db.Do("HSET", rh.table+":"+elementid, key, value)
//	return err
//}
//
//// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password")
//func (rh *HashMap) Get(elementid, key string) (string, error) {
//	db := rh.host.Get(rh.dbname)
//	result, err := db.String(db.Do("HGET", rh.table+":"+elementid, key))
//	if err != nil {
//		return "", err
//	}
//	return result, nil
//}
//
//// Check if a given elementid + key is in the hash map
//func (rh *HashMap) Has(elementid, key string) (bool, error) {
//	db := rh.host.Get(rh.dbname)
//	retval, err := db.Do("HEXISTS", rh.table+":"+elementid, key)
//	if err != nil {
//		panic(err)
//	}
//	return db.Bool(retval, err)
//}
//
//// Check if a given elementid exists as a hash map at all
//func (rh *HashMap) Exists(elementid string) (bool, error) {
//	// TODO: key is not meant to be a wildcard, check for "*"
//	return hasKey(rh.host, rh.table+":"+elementid, rh.dbname)
//}
//
//// Get all elementid's for all hash elements
//func (rh *HashMap) GetAll() ([]string, error) {
//	db := rh.host.Get(rh.dbname)
//	result, err := db.Values(db.Do("KEYS", rh.table+":*"))
//	strs := make([]string, len(result))
//	idlen := len(rh.table)
//	for i := 0; i < len(result); i++ {
//		strs[i] = getString(result, i)[idlen+1:]
//	}
//	return strs, err
//}
//
//// Remove a key for an entry in a hashmap (for instance the email field for a user)
//func (rh *HashMap) DelKey(elementid, key string) error {
//	db := rh.host.Get(rh.dbname)
//	_, err := db.Do("HDEL", rh.table+":"+elementid, key)
//	return err
//}
//
//// Remove an element (for instance a user)
//func (rh *HashMap) Del(elementid string) error {
//	db := rh.host.Get(rh.dbname)
//	_, err := db.Do("DEL", rh.table+":"+elementid)
//	return err
//}
//
//// Remove this hashmap
//func (rh *HashMap) Remove() error {
//	db := rh.host.Get(rh.dbname)
//	_, err := db.Do("DEL", rh.table)
//	return err
//}
//
///* --- KeyValue functions --- */
//
//// Create a new key/value
//func NewKeyValue(host *sql.DB, table string) *KeyValue {
//	return &KeyValue{host, table, defaultDatabaseName}
//}
//
//// Select a different database
//func (rkv *KeyValue) SelectDatabase(dbname string) {
//	rkv.dbname = dbname
//}
//
//// Set a key and value
//func (rkv *KeyValue) Set(key, value string) error {
//	db := rkv.host.Get(rkv.dbname)
//	_, err := db.Do("SET", rkv.table+":"+key, value)
//	return err
//}
//
//// Get a value given a key
//func (rkv *KeyValue) Get(key string) (string, error) {
//	db := rkv.host.Get(rkv.dbname)
//	result, err := db.String(db.Do("GET", rkv.table+":"+key))
//	if err != nil {
//		return "", err
//	}
//	return result, nil
//}
//
//// Remove a key
//func (rkv *KeyValue) Del(key string) error {
//	db := rkv.host.Get(rkv.dbname)
//	_, err := db.Do("DEL", rkv.table+":"+key)
//	return err
//}
//
//// Remove this key/value
//func (rkv *KeyValue) Remove() error {
//	db := rkv.host.Get(rkv.dbname)
//	_, err := db.Do("DEL", rkv.table)
//	return err
//}
//
//// --- Generic db functions ---
//
//// Check if a key exists. The key can be a wildcard (ie. "user*").
//func hasKey(host *sql.DB, wildcard string, dbname string) (bool, error) {
//	db := host.Get(dbname)
//	result, err := db.Values(db.Do("KEYS", wildcard))
//	if err != nil {
//		return false, err
//	}
//	return len(result) > 0, nil
//}
