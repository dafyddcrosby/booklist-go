// Copyright 2016, David Crosby
// 2-clause BSD license
package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"flag"
	"fmt"
	"time"
)

var add_book_flag = flag.Bool("a", false, "Add a book record")
var edit_book_flag = flag.Uint64("e", 0, "Edit a book record")
var delete_book_flag = flag.Uint64("d", 0, "Delete a book record")
var read_book_flag = flag.Uint64("r", 0, "Read a book record")
var search_books_flag = flag.Bool("s", false, "Search for a book")
var list_book_flag = flag.Bool("l", false, "List all books in the database")

var id_flag = flag.Uint64("id", 0, "Book ID number")
var title_flag = flag.String("title", "", "Book title")
var author_flag = flag.String("author", "", "Book author")
var addn_authors_flag = flag.String("addn_authors", "", "Additional authors")
var state_flag = flag.String("state", "", "What state the book is in")
var date_read_flag = flag.String("date_read", "", "Date read (eg 2014-02-14)")
var stars_flag = flag.Uint("stars", 0, "Rating for book (generally 1-5)")

var init_flag = flag.Bool("init", false, "Initialize the database")
var version_flag = flag.Bool("version", false, "Get the booklist version")
var help_flag = flag.Bool("h", false, "Show this message")

type Book struct {
	id int64
	title string
	author string
	addn_authors string
	state string
	stars int64
	date_read time.Time
	created_at time.Time
	updated_at time.Time
}

func (book Book) print_record() {
	fmt.Println("ID:", book.id)
	fmt.Println("Title:", book.title)
	if book.author != "" {
		fmt.Println("Author:", book.author)
	}
	if book.addn_authors != "" {
		fmt.Println("Additional authors:", book.addn_authors)
	}
	if book.state != "" {
		fmt.Println("State:", book.state)
	}
	if ! book.date_read.IsZero() {
		fmt.Println("Date Read:", book.date_read.Format("2006-01-02"))
	}
	if book.stars > 0 {
		fmt.Println("Stars:", book.stars)
	}
	fmt.Println("")
}

func add_book(db *sql.DB) {
	var book Book
	if *title_flag != "" {
		book.title = *title_flag
	} else {
		panic("No title present!")
	}
	if *author_flag != "" {
		book.author = *author_flag
	}
	if *addn_authors_flag != "" {
		book.addn_authors = *addn_authors_flag
	}
	if *state_flag != "" {
		book.state = *state_flag
	}
	if *date_read_flag != "" {
		var t time.Time
		t, err := time.Parse("2006-01-02", *date_read_flag) // TODO better date parsing
		if err != nil {
			panic("bad date string")
		}
		book.date_read = t
	}
	if *stars_flag > 0 {
		book.stars = int64(*stars_flag)
	}

	result, err := db.Exec("INSERT INTO books (title, author, addn_authors, state, date_read, stars, created_at, updated_at) VALUES (?, ?, ?, ?, date(?), ?, datetime('now'), datetime('now'))", book.title, book.author, book.addn_authors, book.state, book.date_read, book.stars)
	if err != nil {
		fmt.Println(err)
	}
	result_id, err := result.LastInsertId()
	if err != nil {
		fmt.Println(err)
	}
	read_book(db, uint64(result_id))
}

func edit_book(db *sql.DB, id uint64) {
	// TODO implement better
	if id > 0 {
		// Use a transaction
		tx, err := db.Begin()
		if err != nil {
			fmt.Println(err)
			return // TODO - should panic instead?
		}
		defer tx.Commit()
		if *title_flag != "" {
			_, err := tx.Exec("UPDATE books SET title = ?, updated_at=datetime('now') WHERE id = ?", *title_flag, id)
			if err != nil {
				fmt.Println(err)
			}
		}
		if *author_flag != "" {
			_, err := tx.Exec("UPDATE books SET author = ?, updated_at=datetime('now') WHERE id = ?", *author_flag, id)
			if err != nil {
				fmt.Println(err)
			}
		}
		if *addn_authors_flag != "" {
			_, err := tx.Exec("UPDATE books SET addn_authors = ?, updated_at=datetime('now') WHERE id = ?", *addn_authors_flag, id)
			if err != nil {
				fmt.Println(err)
			}
		}
		if *state_flag != "" {
			_, err := tx.Exec("UPDATE books SET state = ?, updated_at=datetime('now') WHERE id = ?", *state_flag, id)
			if err != nil {
				fmt.Println(err)
			}
		}
		if *date_read_flag != "" {
			var t time.Time
			t, err := time.Parse("2006-01-02", *date_read_flag) // TODO better date parsing
			if err != nil {
				fmt.Println("bad date string")
				return
			}
			_, err = tx.Exec("UPDATE books SET date_read = date(?), updated_at = datetime('now') WHERE id = ?", t, id)
			if err != nil {
				fmt.Println(err)
			}
		}
		if *stars_flag > 0 {
			_, err := tx.Exec("UPDATE books SET stars = ?, updated_at=datetime('now') WHERE id = ?", *stars_flag, id)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func delete_book(db *sql.DB, id uint64) {
	read_book(db, id) // prints record first, if it exists
	if id > 0 {
		_, err := db.Exec("DELETE FROM books WHERE id = ?", id)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func read_book(db *sql.DB, id uint64) {
	if id > 0 {
		var book Book
		err := db.QueryRow("SELECT id, title, author, coalesce(addn_authors,\"\"), state, date_read, coalesce(stars,0) FROM books WHERE id = ?", id).Scan(&book.id, &book.title, &book.author, &book.addn_authors, &book.state, &book.date_read, &book.stars)
		switch {
		case err == sql.ErrNoRows:
		        fmt.Println("No book with that ID.")
		case err != nil:
		        fmt.Println(err)
		default:
			book.print_record()
		}
	}
}

func search_books(db *sql.DB) {
	if *title_flag != "" {
		rows, err := db.Query("SELECT id, title, author, coalesce(addn_authors,\"\"), state, date_read, coalesce(stars,0) FROM books WHERE title LIKE ?", *title_flag)
		if err != nil {
			fmt.Println(err)
		}
		defer rows.Close()
		for rows.Next() {
			var book Book
			// BUG - The Scan() can have issues with nil datestamps
			_ = rows.Scan(&book.id, &book.title, &book.author, &book.addn_authors, &book.state, &book.date_read, &book.stars)
			book.print_record()
		}
	}
	if *author_flag != "" {
		rows, err := db.Query("SELECT id, title, author, coalesce(addn_authors,\"\"), state, date_read, coalesce(stars,0) FROM books WHERE author LIKE ?", *author_flag)
		if err != nil {
			fmt.Println(err)
		}
		defer rows.Close()
		for rows.Next() {
			var book Book
			// BUG - The Scan() can have issues with nil datestamps
			_ = rows.Scan(&book.id, &book.title, &book.author, &book.addn_authors, &book.state, &book.date_read, &book.stars)
			book.print_record()
		}
	}
	if *addn_authors_flag != "" {
		rows, err := db.Query("SELECT id, title, author, coalesce(addn_authors,\"\"), state, date_read, coalesce(stars,0) FROM books WHERE addn_authors LIKE ?", *addn_authors_flag)
		if err != nil {
			fmt.Println(err)
		}
		defer rows.Close()
		for rows.Next() {
			var book Book
			// BUG - The Scan() can have issues with nil datestamps
			_ = rows.Scan(&book.id, &book.title, &book.author, &book.addn_authors, &book.state, &book.date_read, &book.stars)
			book.print_record()
		}
	}
	if *state_flag != "" {
		rows, err := db.Query("SELECT id, title, author, coalesce(addn_authors,\"\"), state, date_read, coalesce(stars,0) FROM books WHERE state LIKE ?", *state_flag)
		if err != nil {
			fmt.Println(err)
		}
		defer rows.Close()
		for rows.Next() {
			var book Book
			// BUG - The Scan() can have issues with nil datestamps
			_ = rows.Scan(&book.id, &book.title, &book.author, &book.addn_authors, &book.state, &book.date_read, &book.stars)
			book.print_record()
		}
	}
}

func list_all_books(db *sql.DB) {
	rows, err := db.Query("SELECT id, title, author, coalesce(addn_authors,\"\"), state, date_read, coalesce(stars,0) FROM books")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var book Book
		// BUG - The Scan() can have issues with nil datestamps
		_ = rows.Scan(&book.id, &book.title, &book.author, &book.addn_authors, &book.state, &book.date_read, &book.stars)
		book.print_record()
	}
}

func main() {
	flag.Parse()
	db, err := sql.Open("sqlite3", "/home/dave/.booklist/booklist.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		fmt.Println(err)
	}
	if *add_book_flag {
		if *title_flag != "" {
			add_book(db)
		} else {
			fmt.Println("A title is needed to add a record")
		}
	} else if *edit_book_flag > 0 {
		edit_book(db, *edit_book_flag)
		read_book(db, *edit_book_flag) // prints record after editing
	} else if *delete_book_flag > 0 {
		delete_book(db, *delete_book_flag)
	} else if *read_book_flag > 0 {
		read_book(db, *read_book_flag)
	} else if *search_books_flag {
		search_books(db)
	} else if *list_book_flag {
		list_all_books(db)
	}
}
