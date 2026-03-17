package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"
)

var db *sql.DB
var store = sessions.NewCookieStore([]byte("barslab-secret-key-2024"))

const adminUser = "admin"
const adminPass = "admin"

func main() {

	var err error
	db, err = sql.Open("sqlite", "./barslab.db")
	if err != nil {
		log.Fatal(err)
	}

	createTables()

	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/new-post", newPostHandler)
	http.HandleFunc("/edit-post", editPostHandler)
	http.HandleFunc("/delete-post", deletePostHandler)

	log.Println("BarsLab running at http://localhost:1453 🚀")
	http.ListenAndServe(":1453", nil)
}

func createTables() {
	query := `
	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		content TEXT,
		image_url TEXT
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Tablo oluşturulamadı:", err)
	}
	
	seedPosts()
}

func seedPosts() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	
	if count > 0 {
		return 
	}
	
	posts := []struct {
		title   string
		content string
	}{
		{
			"YILDIZ CTI BLOG",
			"Yıldız Siber Tehdit İstihbaratı Takımı, Siber Vatan programı kapsamında faaliyet gösteren, siber tehditlerin tespiti ve analizi üzerine uzmanlaşmış bir ekiptir. Güncel tehditleri anlamak ve savunma stratejileri geliştirmek için eğitimler ve senaryolar düzenler. Misyonu, üyelerini yetkin siber güvenlik profesyonelleri haline getirmek ve yerli projelerle milli güvenliğe katkı sağlamaktır.",
		},
		{
			"YAVUZLAR BLOG",
			"Yavuzlar Web Güvenliği ve Yazılım Takımı, Siber Vatan programı kapsamında kurulan, web güvenliği ve yazılım geliştirme odaklı çalışan bir takımdır. Misyonumuz, her bir takım üyesini kendi alanında yetkin bir düzeye yükseltmek ve ülkemizin siber güvenlik sektöründeki yetenekli insan gücünü, Türkçe kaynakları ve yerli projeleri artırmak amacıyla çalışmaktır.",
		},
		{
			"ZAYOTEM BLOG",
			"Zararlı Yazılım Önleme ve Tersine Mühendislik Takımı (ZAYOTEM), Siber Vatan projesi kapsamında kurulan, zararlı yazılım analiz-önleme, binary exploitation ve diğer low level konseptleri çalışma alanı olarak belirlemiş; bu alanlarda çözümler sunan, projeler geliştiren ve donanımlı uzmanlar yetiştiren bir Tersine Mühendislik takımıdır.",
		},
		{
			"ALTAY BLOG",
			"Siber Vatan programının bir parçası olan Altay takımı, 2024 yılında kurularak faaliyete geçmiş ve özellikle mavi takım tarafında uzmanlaşan bir takımdır. Misyonumuz, her bir takım üyesini kendi alanında yetkin bir düzeye yükseltmek ve ülkemizin siber güvenlik sektöründeki yetenekli insan gücünü, Türkçe kaynakları ve yerli projeleri artırmak amacıyla çalışmaktır.",
		},
	}
	
	for _, post := range posts {
		_, err := db.Exec("INSERT INTO posts(title, content) VALUES (?, ?)", post.title, post.content)
		if err != nil {
			log.Println("Seed hatası:", err)
		}
	}
	
	log.Println("Blog yazıları başarıyla eklendi!")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	type Post struct {
		Title    string
		Content  string
		ImageURL string
	}

	type PageData struct {
		Posts           []Post
		IsAuthenticated bool
	}

	rows, err := db.Query("SELECT title, content, COALESCE(image_url, '') FROM posts")
	if err != nil {
		http.Error(w, "Veritabanı hatası", http.StatusInternalServerError)
		log.Println("Query hatası:", err)
		return
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.Title, &p.Content, &p.ImageURL); err != nil {
			log.Println("Scan hatası:", err)
			continue
		}
		posts = append(posts, p)
	}

	data := PageData{
		Posts:           posts,
		IsAuthenticated: isAuthenticated(r),
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template hatası", http.StatusInternalServerError)
		log.Println("Template hatası:", err)
		return
	}
	tmpl.Execute(w, data)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "barslab-session")

	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		user := r.FormValue("username")
		pass := r.FormValue("password")

		if user == adminUser && pass == adminPass {
			session.Values["authenticated"] = true
			session.Save(r, w)
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
	}

	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "Template hatası", http.StatusInternalServerError)
		log.Println("Template hatası:", err)
		return
	}
	tmpl.Execute(w, nil)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "barslab-session")
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func isAuthenticated(r *http.Request) bool {
	session, _ := store.Get(r, "barslab-session")
	auth, ok := session.Values["authenticated"].(bool)
	return ok && auth
}

func deletePostHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		id := r.FormValue("id")
		_, err := db.Exec("DELETE FROM posts WHERE id = ?", id)
		if err != nil {
			log.Println("Silme hatası:", err)
		}
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func editPostHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	type Post struct {
		ID       int
		Title    string
		Content  string
		ImageURL string
	}

	if r.Method == "POST" {
		id := r.FormValue("id")
		title := r.FormValue("title")
		content := r.FormValue("content")
		imageURL := r.FormValue("image_url")

		_, err := db.Exec("UPDATE posts SET title = ?, content = ?, image_url = ? WHERE id = ?", title, content, imageURL, id)
		if err != nil {
			http.Error(w, "Güncelleme hatası", http.StatusInternalServerError)
			log.Println("Update hatası:", err)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	id := r.URL.Query().Get("id")
	var post Post
	err := db.QueryRow("SELECT id, title, content, COALESCE(image_url, '') FROM posts WHERE id = ?", id).Scan(&post.ID, &post.Title, &post.Content, &post.ImageURL)
	if err != nil {
		http.Error(w, "Yazı bulunamadı", http.StatusNotFound)
		log.Println("Query hatası:", err)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit_post.html")
	if err != nil {
		http.Error(w, "Template hatası", http.StatusInternalServerError)
		log.Println("Template hatası:", err)
		return
	}
	tmpl.Execute(w, post)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	type Post struct {
		ID       int
		Title    string
		Content  string
		ImageURL string
	}

	rows, err := db.Query("SELECT id, title, content, COALESCE(image_url, '') FROM posts ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Veritabanı hatası", http.StatusInternalServerError)
		log.Println("Query hatası:", err)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.ImageURL); err != nil {
			log.Println("Scan hatası:", err)
			continue
		}
		posts = append(posts, p)
	}

	tmpl, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		http.Error(w, "Template hatası", http.StatusInternalServerError)
		log.Println("Template hatası:", err)
		return
	}
	tmpl.Execute(w, posts)
}

func newPostHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		title := r.FormValue("title")
		content := r.FormValue("content")
		imageURL := r.FormValue("image_url")

		_, err := db.Exec("INSERT INTO posts(title, content, image_url) VALUES (?,?,?)", title, content, imageURL)
		if err != nil {
			http.Error(w, "Yazı kaydedilemedi", http.StatusInternalServerError)
			log.Println("Insert hatası:", err)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/new_post.html")
	if err != nil {
		http.Error(w, "Template hatası", http.StatusInternalServerError)
		log.Println("Template hatası:", err)
		return
	}
	tmpl.Execute(w, nil)
}