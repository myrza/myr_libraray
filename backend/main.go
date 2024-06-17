package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Author struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Surname   string `json:"surname"`
	Biography string `json:"biography"`
	Birthday  string `json:"birthday"`
}

type Book struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	AuthorID string `json:"authorid"`
	ISBN     string `json:"isbn"`
	Year     string `json:"year"`
}

// main function
func main() {
	// connect to db
	//db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	connectionStr := "user=postgres password=postgres dbname=library port=5432 sslmode=disable"

	//db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/postgres?sslmode=disable")
	db, err := sql.Open("library", connectionStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// логируем в файл
	flog, err := os.OpenFile("logfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer flog.Close()

	log.SetOutput(flog)

	// create table if not exists
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS authors (id SERIAL PRIMARY KEY, name TEXT, surname TEXT, biography TEXT, birthday DATE)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS books (id SERIAL PRIMARY KEY, title TEXT, authorid INTEGER, isbn TEXT, year INTEGER)")
	if err != nil {
		log.Fatal(err)
	}

	// create router
	router := mux.NewRouter()

	// операции по авторам книг
	router.HandleFunc("/api/go/authors", getAuthors(db)).Methods("GET")
	router.HandleFunc("/api/go/authors", createAuthor(db)).Methods("POST")
	router.HandleFunc("/api/go/authors/{id}", getAuthor(db)).Methods("GET")
	router.HandleFunc("/api/go/authors/{id}", updateAuthor(db)).Methods("PUT")
	router.HandleFunc("/api/go/authors/{id}", deleteAuthor(db)).Methods("DELETE")

	// операции по книгам
	router.HandleFunc("/api/go/books", getBooks(db)).Methods("GET")
	router.HandleFunc("/api/go/books", createBook(db)).Methods("POST")
	router.HandleFunc("/api/go/books/{id}", getBook(db)).Methods("GET")
	router.HandleFunc("/api/go/books/{id}", updateBook(db)).Methods("PUT")
	router.HandleFunc("/api/go/books/{id}", deleteBook(db)).Methods("DELETE")

	//Транзакционное обновление
	router.HandleFunc("/api/go/books/{book_id}/authors/{author_id}", updateBookAndAuthor(db)).Methods("PUT")

	// wrap the router with CORS and JSON content type middlewares
	enhancedRouter := enableCORS(jsonContentTypeMiddleware(router))

	// start server
	log.Fatal(http.ListenAndServe(":8000", enhancedRouter))
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Check if the request is for CORS preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass down the request to the next middleware (or final handler)
		next.ServeHTTP(w, r)
	})

}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set JSON Content-Type
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// все авторы
func getAuthors(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT * FROM authors")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		authors := []Author{}
		for rows.Next() {
			var a Author
			if err := rows.Scan(&a.ID, &a.Name, &a.Surname, &a.Biography, &a.Birthday); err != nil {
				log.Fatal(err)
			}
			authors = append(authors, a)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(authors)
	}
}

// Автор через id
func getAuthor(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var a Author
		err := db.QueryRow("SELECT * FROM authors WHERE id = $1", id).Scan(&a.ID, &a.Name, &a.Surname, &a.Biography, &a.Birthday)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(a)
	}
}

// создать автора
func createAuthor(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var a Author
		json.NewDecoder(r.Body).Decode(&a)

		err := db.QueryRow("INSERT INTO authors (name, surname, biography, birthday) VALUES ($1, $2, $3, $4) RETURNING id", a.Name, a.Surname, a.Biography, a.Birthday).Scan(&a.ID)
		if err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(a)
	}
}

// обновить автора
func updateAuthor(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var a Author
		json.NewDecoder(r.Body).Decode(&a)

		vars := mux.Vars(r)
		id := vars["id"]

		// Execute the update query
		_, err := db.Exec("UPDATE authors SET name = $1, surname = $2,biography=$3, birthday = $4  WHERE id = $5", a.Name, a.Surname, a.Biography, a.Birthday, id)
		if err != nil {
			log.Fatal(err)
		}

		// Retrieve the updated user data from the database
		var updatedAuthor Author
		err = db.QueryRow("SELECT * FROM authors WHERE id = $1", id).Scan(&updatedAuthor.ID, &updatedAuthor.Name, &updatedAuthor.Surname, &updatedAuthor.Biography, &updatedAuthor.Birthday)
		if err != nil {
			log.Fatal(err)
		}

		// Send the updated user data in the response
		json.NewEncoder(w).Encode(updatedAuthor)
	}
}

// удалить автора
func deleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var a Author
		err := db.QueryRow("SELECT * FROM authors WHERE id = $1", id).Scan(&a.ID, &a.Name, &a.Surname, &a.Biography, &a.Birthday)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			_, err := db.Exec("DELETE FROM users WHERE id = $1", id)
			if err != nil {
				//todo : fix error handling
				w.WriteHeader(http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode("Автор удален")
		}
	}
}

// книги
// все книги
func getBooks(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT * FROM books")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		books := []Book{} // array of users
		for rows.Next() {
			var b Book
			if err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.ISBN, &b.Year); err != nil {
				log.Fatal(err)
			}
			books = append(books, b)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(books)
	}
}

// книга через id
func getBook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var b Book
		err := db.QueryRow("SELECT * FROM books WHERE id = $1", id).Scan(&b.ID, &b.Title, &b.AuthorID, &b.ISBN, &b.Year)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(b)
	}
}

// создать книгу
func createBook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b Book
		json.NewDecoder(r.Body).Decode(&b)

		err := db.QueryRow("INSERT INTO books (title, authorid, isbn, year) VALUES ($1, $2, $3, $4) RETURNING id", b.Title, b.AuthorID, b.ISBN, b.Year).Scan(&b.ID)
		if err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(b)
	}
}

// обновить книгу
func updateBook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b Book
		json.NewDecoder(r.Body).Decode(&b)

		vars := mux.Vars(r)
		id := vars["id"]

		// Execute the update query
		_, err := db.Exec("UPDATE books SET title = $1, authorid = $2, isbn=$3, year = $4  WHERE id = $5", b.Title, b.AuthorID, b.ISBN, b.Year, id)
		if err != nil {
			log.Fatal(err)
		}

		// Retrieve the updated user data from the database
		var updatedBook Book
		err = db.QueryRow("SELECT * FROM books WHERE id = $1", id).Scan(&updatedBook.ID, &updatedBook.Title, &updatedBook.AuthorID, &updatedBook.ISBN, &updatedBook.Year)
		if err != nil {
			log.Fatal(err)
		}

		// Send the updated user data in the response
		json.NewEncoder(w).Encode(updatedBook)
	}
}

// удалить автора
func deleteBook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var b Book
		err := db.QueryRow("SELECT * FROM books WHERE id = $1", id).Scan(&b.ID, &b.Title, &b.AuthorID, &b.ISBN, &b.Year)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			_, err := db.Exec("DELETE FROM books WHERE id = $1", id)
			if err != nil {
				//todo : fix error handling
				w.WriteHeader(http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode("книга удалена")
		}
	}
}
