package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	pb "gopkg.in/cheggaaa/pb.v1"

	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
)

const tpl = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Diskotek</title>
    <style>
      * { box-sizing: border-box; font-family: monospace; }

      #search-input {
        width: 100%; font-size: 16px; padding: 12px 20px 12px 20px;
        border: 1px solid #ddd; margin-bottom: 12px; text-align: center;
      }

      #artists { list-style-type: none; padding: 0; margin: 0; }

      #artists li a {
        border: 1px solid #ddd; margin-top: -1px; background-color: #f6f6f6;
        padding: 12px; text-decoration: none; font-size: 14px;
        text-align: center; color: black; display: block;
      }

      #artists li a:hover:not(.header) { background-color: #eee; }

	  .albums { display: none; }

	  a[name="top"] {
		top: -5px;
		position: absolute;
	  }

	  #top-link {
	    font-size: 150%;
		position: fixed;
		bottom: 17px;
		right: 15px;
		z-index: 99;
		border: none;
		outline: none;
		background-color: #D3D3D3;
		color: black;
		cursor: pointer;
		border-radius: 5px;
		opacity: 0.8;
		height: 40px;
		width: 40px;
		text-align: center;
		margin: 0;
	  }

	  #top-link a:hover {
		text-decoration: none;
	  }

	  #top-link a {
		display: block;
		height: 100%;
		width: 100%;
		text-decoration: none;
		padding: 7px;
	  }

	  #top-link a:visited {
		color: black;
	  }
  </style>
  </head>
	<body>
	<a name="top"></a>
	<input type="text" id="search-input" onkeyup="search()" placeholder="Search for artists..">
    <ul id="artists">{{range .Artists}}
      <li class="artist-item">
        <a class="artist-name">{{.Name}}</a>
        <ul class="albums">{{range .Albums}}
          <li><div class="album-name">{{ . }}</div></li>
        {{end}}</ul>
      </li>
		{{end}}</ul>
	<div id="top-link"><a href="#top">&uarr;</a></div>
  <script>
    function slugify(text) {
      return text.toString().toLowerCase()
        .replace(/\s+/g, '-')            // Replace spaces with -
        .replace(/-/g, '')               // Replace multiple - with single -
        .replace(/^-+/, '')              // Trim - from start of text
        .replace(/^-+/, '')              // Replace "&" into and
        .replace(/-+$/, '')              // Trim - from end of text
        .normalize('NFD').replace(/[\u0300-\u036f]/g, "")  // Remove accents
        .replace(/[^\w\-]+/g, '');        // Remove all non-word chars
      }

    function search() {
      var input, filter, ul, li, a, i;
      input = document.getElementById('search-input');
      filter = slugify(input.value);
      ul = document.getElementById("artists");
      li = ul.getElementsByClassName('artist-item');

      for (i = 0; i < li.length; i++) {
        a = li[i].getElementsByTagName("a")[0];
        if (slugify(a.innerHTML).indexOf(filter) > -1) {
            li[i].style.display = "";
        } else {
            li[i].style.display = "none";
        }
      }
		}
  </script>
</body>
</html>`

var (
	bar            *pb.ProgressBar
	db             *sql.DB
	latestArtist   string
	latestAlbum    string
	directoryCount int
)

func countDirectories(path string) {
	filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			directoryCount = directoryCount + 1
		}
		return nil
	})
}

func visit(path string, f os.FileInfo, err error) error {
	var (
		statement   *sql.Stmt
		artist      string
		artistID    int64
		album       string
		albumResult string
	)
	if f.IsDir() {
		bar.Increment()
		return nil
	}

	file, err := os.Open(path)
	metaData, err := tag.ReadFrom(file)
	if err != nil {
		return nil
	}

	artist = strings.Trim(metaData.Artist(), " ")

	if artist == "" {
		return nil
	}

	album = strings.Trim(metaData.Album(), " ")
	if album == "" {
		album = "---"
	}

	if artist != latestArtist {
		err = db.QueryRow("SELECT id FROM artist WHERE name LIKE ? LIMIT 1", artist).Scan(&artistID)
		switch {
		case err == sql.ErrNoRows:
			statement, _ = db.Prepare("INSERT INTO artist (name) VALUES (?)")
			result, _ := statement.Exec(artist)
			artistID, _ = result.LastInsertId()
		case err != nil:
			log.Fatal(err)
		default:
		}
	}

	if album != latestAlbum {
		err = db.QueryRow("SELECT id FROM album WHERE artist_id=? AND name LIKE ? LIMIT 1", artistID, album).Scan(&albumResult)
		switch {
		case err == sql.ErrNoRows:
			statement, _ = db.Prepare("INSERT INTO album (name, artist_id) VALUES (?, ?)")
			statement.Exec(album, artistID)
		case err != nil:
			log.Fatal(err)
		default:
		}
	}

	latestArtist = artist
	latestAlbum = album

	return nil
}

func generateDB(path string) error {
	fmt.Println("Counting is hard...")
	countDirectories(path)
	bar = pb.StartNew(directoryCount)
	err := filepath.Walk(path, visit)
	bar.FinishPrint("Done.")
	return err
}

func generateHTML() error {
	type Artist struct {
		Name   string
		Albums []string
	}

	t, err := template.New("index").Parse(tpl)

	artists := []Artist{}

	artistRows, _ := db.Query("SELECT id, name FROM artist ORDER BY LOWER(name)")
	var (
		artistID   int64
		artistName string
	)
	for artistRows.Next() {
		artistRows.Scan(&artistID, &artistName)
		artist := Artist{
			Name:   artistName,
			Albums: []string{},
		}

		albumRows, _ := db.Query("SELECT name FROM album WHERE artist_id=?", artistID)
		var albumName string
		for albumRows.Next() {
			albumRows.Scan(&albumName)
			artist.Albums = append(artist.Albums, albumName)
		}

		artists = append(artists, artist)
	}

	err = t.Execute(os.Stdout, struct {
		Artists []Artist
	}{
		Artists: artists,
	})
	return err
}

func openDB() {
	db, _ = sql.Open("sqlite3", "./diskotek.db")
	statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS artist (id INTEGER PRIMARY KEY, name TEXT)")
	statement.Exec()
	statement, _ = db.Prepare("CREATE TABLE IF NOT EXISTS album (id INTEGER PRIMARY KEY, name TEXT, artist_id INTEGER, FOREIGN KEY(artist_id) REFERENCES artist(id))")
	statement.Exec()
}

func main() {

	flag.Usage = func() {
		fmt.Printf("Usage: diskotek [-scan] [-library-path] [-html]\n\n")
		flag.PrintDefaults()
	}

	versionPtr := flag.Bool("version", false, "0.1.0")
	scanPtr := flag.Bool("scan", false, "Scan music library (with -library-path)")
	pathPtr := flag.String("library-path", "", "Music library path")
	htmlPtr := flag.Bool("html", false, "Generate HTML")

	flag.Parse()

	if *versionPtr {
		fmt.Printf("Version %s\n", (flag.Lookup("version")).Usage)
		os.Exit(0)
	}

	if !*scanPtr && !*htmlPtr {
		flag.Usage()
		os.Exit(2)
	}

	if *scanPtr && *pathPtr == "" {
		fmt.Printf("Error: need -library-path with -scan\n\n")
		flag.Usage()
		os.Exit(3)
	}

	openDB()

	if *scanPtr {
		generateDB(*pathPtr)
	}

	if *htmlPtr {
		generateHTML()
	}

	os.Exit(0)
}
