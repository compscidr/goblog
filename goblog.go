package main

func main() {
  //structure of this taken from: https://rshipp.com/go-web-api/
  a := &App{}
  a.Initialize("sqlite3", "test.db")
  a.Listen(8000)
  defer a.DB.Close()
}
