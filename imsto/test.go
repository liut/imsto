package main

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	cimg "wpst.me/calf/image"
	"wpst.me/calf/storage"
)

var cmdTest = &Command{
	UsageLine: "test attr|mime|image|thumb filename [destname]",
	Short:     "run all tests from the command-line",
	Long: `
Just a test command
`,
}

var (
	turl  = cmdTest.Flag.String("path", "", "entry path for load")
	tfile = cmdTest.Flag.String("file", "", "test a entry from a file")
)

func init() {
	cmdTest.Run = testApp
}

func testApp(args []string) bool {
	if *turl != "" {
		fmt.Println("url: ", *turl)
		item, err := storage.LoadPath(*turl)
		if err != nil {
			fmt.Println("Err: ", err)
			return false
		}
		c := func(file http.File) {
			fi, err := file.Stat()
			if err != nil {
				fmt.Print(err)
			}
			fmt.Printf("file size: %d, mod: %s\n", fi.Size(), fi.ModTime())
		}
		err = item.Walk(c)
		if err != nil {
			fmt.Println("Err: ", err)
			return false
		}
		return true
	}

	if *tfile != "" {
		data, err := ioutil.ReadFile(*tfile)
		if err != nil {
			fmt.Println("read file error: ", err)
			return false
		}
		entry, err := storage.NewEntry(data, path.Base(*tfile))
		if err != nil {
			fmt.Println("new entry error: ", err)
			return false
		}
		err = entry.Trek("demo")
		if err != nil {
			fmt.Println("trek error: ", err)
			return false
		}

		fmt.Printf("new id: %v, size: %d, path: %v, %v\n", entry.Id, entry.Size, entry.Path, entry.Hashes)
		return true
	}

	al := len(args)
	fmt.Println(args)
	if al == 0 {
		fmt.Println("nothing")
		return false
	}

	if al > 1 && args[0] == "attr" {

		file, err := os.Open(args[1])

		if err != nil {
			fmt.Println(err)
			return false
		}
		defer file.Close()
		var (
			im cimg.Image
		)
		im, err = cimg.Open(file)
		fmt.Printf("attr: %s", im.GetAttr())
		return true
	}

	if al > 1 && args[0] == "mime" {

		ext := path.Ext(args[1])
		fmt.Println(ext)
		mimetype := mime.TypeByExtension(ext)
		fmt.Println(mimetype)
		return true
	}

	if al > 2 && args[0] == "thumb" {
		topt := cimg.ThumbOption{Width: 120, Height: 120, IsFit: true, IsCrop: true}
		err := cimg.ThumbnailFile(args[1], args[2], topt)
		if err != nil {
			fmt.Printf("thumb error: %s", err)
			return false
		}
		return false
	}

	if al > 1 && args[0] == "image" {
		im, format, err := readImage(args[1])
		if err != nil {
			fmt.Println(err)
			return false
		}
		fmt.Println(im.Bounds(), format, err)

		if al > 2 {
			var outfile *os.File
			outfile, err = os.Create(args[2])
			if err != nil {
				fmt.Println(err)
				return false
			}
			if format == "png" {
				err = png.Encode(outfile, im)
			} else if format == "jpeg" {
				err = jpeg.Encode(outfile, im, &jpeg.Options{Quality: 75})
			} else {
				fmt.Println("unsupported format")
				return false
			}
			if err != nil {
				fmt.Println(err)
				return false
			}
		}
	}

	return true
}

func readImage(filename string) (image.Image, string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	return image.Decode(bufio.NewReader(f))
}

func readImageConfig(filename string) (image.Config, string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return image.Config{}, "", err
	}
	defer f.Close()
	return image.DecodeConfig(bufio.NewReader(f))
}
