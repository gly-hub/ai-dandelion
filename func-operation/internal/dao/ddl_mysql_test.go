package dao

import "testing"

func TestNormalizeDDLForMySQL(t *testing.T) {
	t.Parallel()

	bookDDL := "CREATE TABLE IF NOT EXISTS func_1_book (id INTEGER PRIMARY KEY AUTOINCREMENT, uuid TEXT UNIQUE NOT NULL, title TEXT NOT NULL)"
	got := normalizeDDLForMySQL(bookDDL)
	want := "CREATE TABLE IF NOT EXISTS func_1_book (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, uuid VARCHAR(255) UNIQUE NOT NULL, title VARCHAR(255) NOT NULL)"
	if got != want {
		t.Fatalf("normalizeDDLForMySQL() = %q, want %q", got, want)
	}

	recordDDL := "CREATE TABLE IF NOT EXISTS func_1_borrowing_record (id INTEGER PRIMARY KEY AUTOINCREMENT, book_id INTEGER NOT NULL REFERENCES func_1_book(id) ON DELETE CASCADE)"
	got = normalizeDDLForMySQL(recordDDL)
	want = "CREATE TABLE IF NOT EXISTS func_1_borrowing_record (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, book_id BIGINT NOT NULL REFERENCES func_1_book(id) ON DELETE CASCADE)"
	if got != want {
		t.Fatalf("normalizeDDLForMySQL() = %q, want %q", got, want)
	}

	statusDefaultDDL := "CREATE TABLE IF NOT EXISTS func_1_book (status TEXT NOT NULL DEFAULT 'available', notes TEXT DEFAULT '')"
	got = normalizeDDLForMySQL(statusDefaultDDL)
	want = "CREATE TABLE IF NOT EXISTS func_1_book (status VARCHAR(255) NOT NULL DEFAULT 'available', notes VARCHAR(255) DEFAULT '')"
	if got != want {
		t.Fatalf("normalizeDDLForMySQL() = %q, want %q", got, want)
	}

	mysqlDDL := "CREATE TABLE IF NOT EXISTS func_2_item (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL)"
	if got := normalizeDDLForMySQL(mysqlDDL); got != mysqlDDL {
		t.Fatalf("normalizeDDLForMySQL() changed mysql ddl: %q", got)
	}
}
