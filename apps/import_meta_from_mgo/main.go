package main

import (
	"flag"
	"github.com/go-imsto/imsto/config"
	"github.com/go-imsto/imsto/image"
	"github.com/go-imsto/imsto/storage"
	"github.com/go-imsto/imsto/storage/types"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"path"
	"time"
)

// url: mongodb://db.wp.net,db20.wp.net/storage
// url: localhost

var (
	cfgDir      string
	mgo_url     string
	mgo_db      string
	roof        string
	mgo_coll    string
	skip, limit int
	mgoSession  *mgo.Session
	id          string
	odir        string
)

type entryOut struct {
	Id          string       `bson:"_id" json:"id"`
	Name        string       `bson:"name"`
	Path        string       `bson:"path"`
	Mime        string       `bson:"mime"`
	ContentType string       `bson:"contentType,omitempty"`
	Size        uint32       `bson:"size"`
	Ids         []string     `bson:"ids"`
	Meta        types.JsonKV `bson:"meta"`
	Sev         types.JsonKV `bson:"sev"`
	Created     time.Time    `bson:"created"`
	AppId       uint8        `bson:"app_id"`
	Width       uint16       `bson:"width,omitempty"`
	Height      uint16       `bson:"height,omitempty"`
	Filename    string       `bson:"filename,omitempty"`
	Length      uint32       `bson:"length,omitempty"`
	ImgType     uint8        `bson:"type,omitempty"`
	Hashes      []string     `bson:"hashes,omitempty"`
	Hash        string       `bson:"hash,omitempty"`
	Md5         string       `bson:"md5,omitempty"`
	UploadDate  time.Time    `bson:"uploadDate,omitempty"`
}

func (eo entryOut) toEntry() (entry *storage.Entry, err error) {
	// log.Print(eo)

	if eo.Path == "" && eo.Filename != "" {
		eo.Path = eo.Filename
	}

	if eo.Size == 0 && eo.Length != 0 {
		eo.Size = eo.Length
	}
	if eo.Meta == nil {
		eo.Meta = make(db.Hstore)
	}
	if eo.Width > 0 {
		eo.Meta.Set("width", eo.Width)
	}
	if eo.Height > 0 {
		eo.Meta.Set("height", eo.Height)
	}
	if eo.ImgType > 0 {
		typeid := image.TypeId(eo.ImgType)
		eo.Meta.Set("format", typeid.String())
	}
	if eo.Mime == "" && eo.ContentType != "" {
		eo.Mime = eo.ContentType
	}
	if eo.Mime != "" {
		eo.Meta.Set("mime", eo.Mime)
	}

	if eo.Hash == "" && eo.Md5 != "" {
		eo.Hash = eo.Md5
	}

	if len(eo.Hashes) == 0 && eo.Hash != "" {
		eo.Hashes = []string{eo.Hash}
	}

	if len(eo.Ids) == 0 {
		eo.Ids = []string{eo.Id}
	} else {
		exists := false
		for _, i := range eo.Ids {
			if i == eo.Id {
				exists = true
			}
		}
		if !exists {
			eo.Ids = append(eo.Ids, eo.Id)
		}
	}

	if eo.Created.IsZero() && !eo.UploadDate.IsZero() {
		eo.Created = eo.UploadDate
	}

	if eo.Created.IsZero() {
		log.Printf("zero Created '%v'", eo.Created)
	}
	// log.Printf("eo %s %s %s %d hahes: %s ids: %s", eo.Id, eo.Path, eo.Mime, eo.Size, eo.Hashes, eo.Ids)
	// log.Printf("meta %s", eo.Meta)

	entry, err = storage.NewEntryConvert(eo.Id, eo.Name, eo.Path, eo.Mime, eo.Size, eo.Meta, eo.Sev, eo.Hashes, eo.Ids, eo.Created)
	if err != nil {
		log.Printf("pre eo: %v", eo)
		log.Printf("toEntry error: %s", err)
		return
	}
	return
}

func init() {
	flag.StringVar(&mgo_url, "h", "mongodb://localhost/storage", "mongodb server url")
	flag.StringVar(&mgo_db, "d", "storage", "mongodb database name")
	flag.StringVar(&mgo_coll, "c", "", "mongodb collection name (without '.files')")
	flag.StringVar(&roof, "s", "", "config section name")
	flag.IntVar(&skip, "skip", 0, "skip")
	flag.IntVar(&limit, "limit", 10, "limit")
	flag.StringVar(&cfgDir, "conf", "/etc/imsto", "app conf dir")
	flag.StringVar(&id, "id", "", "single item id")
	flag.StringVar(&odir, "o", "", "export the single item to a special directory")
	flag.Parse()
	if cfgDir != "" {
		config.SetRoot(cfgDir)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := config.Load()
	if err != nil {
		log.Print("config load error: ", err)
	}

}

func main() {
	if mgo_coll == "" || roof == "" || config.Root() == "" {
		flag.PrintDefaults()
		return
	}
	collectionName := mgo_coll + ".files"

	log.Printf("import : %s", roof)
	var q bson.M
	if id != "" {
		log.Printf("id %s", id)
		eo, err := QueryEntry(collectionName, id)
		if err != nil {
			log.Printf("query error: %s", err)
			return
		}

		if odir != "" {
			f := func(fs *mgo.GridFS) error {
				r, err := fs.OpenId(id)
				if err != nil {
					log.Printf("OpenId(%s) error: %s", id, err)
					return err
				}
				defer r.Close()
				data, err := ioutil.ReadAll(r)
				if err != nil {
					return err
				}
				name := path.Join(odir, r.Name())
				err = storage.SaveFile(name, data)
				if err == nil {
					log.Printf("save %s to %s ok", id, name)
				}
				return err
			}
			err = withFs(mgo_coll, f)
			if err != nil {
				log.Printf("fs error: %s", err)
			}

			return
		}
		mw := storage.NewMetaWrapper(roof)
		entry, err := eo.toEntry()
		if err != nil {
			log.Printf("to entry error: %s", err)
			return
		}
		log.Printf("entry %v", entry)
		err = mw.Save(entry, false)
		if err != nil {
			log.Printf("save error: %s", err)
			return
		}
		return
	}
	q = bson.M{}

	total, err := CountEntry(collectionName, q)
	if err != nil {
		log.Printf("count error: %s", err)
		return
	}
	log.Printf("total: %d", total)
	// skip := 0
	// limit := 5
	for skip < total {
		log.Printf("start %d/%d", skip, total)
		results, err := QueryEntries(collectionName, q, skip, limit)
		if err != nil {
			log.Printf("query error: %s", err)
		}
		// log.Printf("results: %s", results)
		entries := make([]*storage.Entry, len(results))
		for i, e := range results {
			// log.Printf("%d %s\n", i, e.Id)
			entries[i], err = e.toEntry()
			if err != nil {
				log.Printf("toEntry error: %s", err)
				return
			}
		}
		mw := storage.NewMetaWrapper(roof)
		err = mw.BatchSave(entries)
		if err != nil {
			log.Printf("BatchSave error: %s", err)
			return
		}
		skip += limit
	}

	log.Printf("%s.%s => [%s] (%d) all done!", mgo_db, mgo_coll, roof, total)
}

func getSession() (*mgo.Session, error) {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial(mgo_url)
		if err != nil {
			log.Printf("error: %s", err)
			return nil, err
		}
	}
	return mgoSession.Clone(), nil
}

func withCollection(collection string, s func(*mgo.Collection) error) error {
	session, err := getSession()
	if err != nil {
		return err
	}
	defer session.Close()
	c := session.DB(mgo_db).C(collection)
	// log.Printf("connection: %v", c)
	return s(c)
}

func QueryEntries(collection string, q interface{}, skip int, limit int) (results []entryOut, err error) {
	results = []entryOut{}
	query := func(c *mgo.Collection) error {
		fn := c.Find(q).Skip(skip).Limit(limit).All(&results)
		if limit < 1 {
			fn = c.Find(q).Skip(skip).All(&results)
		}
		return fn
	}
	search := func() error {
		return withCollection(collection, query)
	}
	err = search()
	return
}

func CountEntry(collection string, q interface{}) (n int, err error) {
	query := func(c *mgo.Collection) error {
		n, err = c.Count()
		return err
	}
	count := func() error {
		return withCollection(collection, query)
	}
	err = count()
	return
}

func QueryEntry(collection string, id string) (eo entryOut, err error) {
	query := func(c *mgo.Collection) error {
		fn := c.FindId(id).One(&eo)
		return fn
	}
	search := func() error {
		return withCollection(collection, query)
	}
	err = search()
	return
}

func withFs(prefix string, f func(*mgo.GridFS) error) error {
	session, err := getSession()
	if err != nil {
		return err
	}
	defer session.Close()
	fs := session.DB(mgo_db).GridFS(prefix)
	return f(fs)
}