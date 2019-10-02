package web

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/bmizerany/pat"

	"github.com/go-imsto/imsto/base"
	"github.com/go-imsto/imsto/config"
	"github.com/go-imsto/imsto/storage"
)

var (
	whiteList = []string{}
)

// Handler ...
func Handler() http.Handler {
	mux := pat.New()
	mux.Get("/imsto/roofs", http.HandlerFunc(roofsHandler))

	mux.Post("/imsto/ticket", http.HandlerFunc(ticketHandlerPost))
	mux.Get("/imsto/ticket", http.HandlerFunc(ticketHandlerGet))

	mux.Post("/imsto/token", http.HandlerFunc(tokenHandler))

	mux.Post("/imsto/:roof", secure(whiteList, postHandler))
	mux.Del("/imsto/:roof/:id", secure(whiteList, deleteHandler))
	mux.Get("/imsto/:roof/id", http.HandlerFunc(GetOrHeadHandler))
	mux.Get("/imsto/:roof/metas/count", http.HandlerFunc(countHandler))
	mux.Get("/imsto/:roof/metas", http.HandlerFunc(browseHandler))
	// mux.Post("/imsto/:roof/token", http.HandlerFunc(tokenHandler))
	// mux.Post("/imsto/:roof/ticket", http.HandlerFunc(ticketHandler))

	return mux
}

func StageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Server", "IMSTO STAGE")

	item, err := storage.LoadPath(r.URL.Path)

	if err != nil {
		logger().Warnw("loadPath fail", "ref", r.Referer(), "err", err)
		switch err.(type) {
		case *storage.HttpError:
			ie := err.(*storage.HttpError)
			if ie.Code == 302 {
				// log.Printf("redirect to %s", ie.Path)
				http.Redirect(w, r, ie.Path, ie.Code)
				return
			}
			// w.WriteHeader(ie.Code)
			http.Error(w, ie.Text, ie.Code)
			return
		}
		writeJsonError(w, r, err)

		return
	}

	// log.Print(item)

	c := func(file io.ReadSeeker) {
		http.ServeContent(w, r, item.Name(), item.Modified(), file)
	}
	err = item.Walk(c)
	if err != nil {
		logger().Warnw("item walk fail", "item", item, "err", err)
		writeJsonError(w, r, err)
		return
	}
}

func roofsHandler(w http.ResponseWriter, r *http.Request) {
	m := newApiMeta(true)
	// m["roofs"] = config.Sections()
	writeJsonQuiet(w, r, newApiRes(m, config.Sections()))
}

func browseHandler(w http.ResponseWriter, r *http.Request) {
	roof := r.URL.Query().Get(":roof")
	logger().Debugw("browse", "roof", roof, "query", r.URL.Query())
	var (
		limit  uint64
		offset uint64
	)

	if str := r.FormValue("rows"); str != "" {
		limit, _ = strconv.ParseUint(str, 10, 32)
		if limit < 1 {
			limit = 1
		}
	} else {
		limit = 20
	}

	if str := r.FormValue("skip"); str != "" {
		offset, _ = strconv.ParseUint(str, 10, 32)
	} else {
		var page uint64
		if str := r.FormValue("page"); str != "" {
			page, _ = strconv.ParseUint(str, 10, 32)
		}
		if page < 1 {
			page = 1
		}

		offset = limit * (page - 1)
	}

	sort := make(map[string]int)
	sort_name := r.FormValue("sort_name")
	sort_order := r.FormValue("sort_order")
	if sort_name != "" {
		var o int
		if strings.ToUpper(sort_order) == "ASC" {
			o = storage.ASCENDING
		} else {
			o = storage.DESCENDING
		}
		sort[sort_name] = o
	}

	filter := storage.MetaFilter{Tags: r.FormValue("tags")}

	mw := storage.NewMetaWrapper(roof)
	t, err := mw.Count(filter)
	if err != nil {
		// w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	a, err := mw.Browse(int(limit), int(offset), sort, filter)
	if err != nil {
		// w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	m := newApiMeta(true)
	m["rows"] = limit
	m["page"] = (offset + 1)
	m["skip"] = offset
	m["page_count"] = uint(math.Ceil(float64(t) / float64(limit)))

	m["total"] = t

	// thumb_path := config.GetValue(roof, "thumb_path")
	// m["thumb_path"] = strings.TrimSuffix(thumb_path, "/") + "/"
	m["url_prefix"] = getUrl(r.URL.Scheme, roof, "") + "/"
	m["version"] = config.Version
	writeJsonQuiet(w, r, newApiRes(m, a))
}

func countHandler(w http.ResponseWriter, r *http.Request) {
	roof := r.FormValue("roof")

	filter := storage.MetaFilter{Tags: r.FormValue("tags")}

	mw := storage.NewMetaWrapper(roof)
	t, err := mw.Count(filter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	m := newApiMeta(true)
	m["total"] = t
	m["version"] = config.Version
	writeJsonQuiet(w, r, newApiRes(m, nil))
}

func GetOrHeadHandler(w http.ResponseWriter, r *http.Request) {
	id, err := base.ParseID(r.URL.Query().Get(":id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	roof := r.URL.Query().Get(":roof")
	mw := storage.NewMetaWrapper(roof)
	entry, err := mw.GetMeta(id.String())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	if r.Method == "HEAD" {
		return
	}
	url := getUrl(r.URL.Scheme, roof, "orig/"+entry.Path)
	log.Printf("Get entry: %v", entry.Id)
	meta := newApiMeta(true)
	obj := struct {
		*storage.Entry
		OrigUrl string `json:"orig_url,omitempty"`
	}{
		Entry:   entry,
		OrigUrl: url,
	}
	writeJsonQuiet(w, r, newApiRes(meta, obj))
}

func getUrl(scheme, roof, size string) string {
	// thumbPath := config.GetValue(roof, "thumb_path")
	spath := path.Join("/", storage.ViewName, size)
	stageHost := config.GetValue(roof, "stage_host")
	if stageHost == "" {
		return spath
	}
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, stageHost, spath)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := storage.StoredRequest(r)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}
	// log.Print(entries[0].Path)
	meta := newApiMeta(true)
	var roof = r.FormValue("roof")

	meta["stage_host"] = config.GetValue(roof, "stage_host")
	meta["url_prefix"] = getUrl(r.URL.Scheme, roof, "") + "/"
	meta["version"] = config.Version

	writeJsonQuiet(w, r, newApiRes(meta, entries))
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	err := storage.Delete(r.URL.Query().Get(":roof"), r.URL.Query().Get(":id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	meta := newApiMeta(true)
	writeJsonQuiet(w, r, newApiRes(meta, nil))
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	token, err := storage.TokenRequestNew(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}

	meta := newApiMeta(true)
	meta["token"] = token.String()
	writeJsonQuiet(w, r, newApiRes(meta, nil))
}

func ticketHandlerPost(w http.ResponseWriter, r *http.Request) {
	token, err := storage.TicketRequestNew(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}
	meta := newApiMeta(false)
	meta["ok"] = true
	meta["token"] = token.String()
	writeJsonQuiet(w, r, newApiRes(meta, nil))
}

func ticketHandlerGet(w http.ResponseWriter, r *http.Request) {
	ticket, err := storage.TicketRequestLoad(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %s", err)
		writeJsonError(w, r, err)
		return
	}
	meta := newApiMeta(false)
	meta["ok"] = true
	meta["ticket"] = ticket

	writeJsonQuiet(w, r, newApiRes(meta, nil))
}