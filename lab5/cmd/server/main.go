package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"lab5/search"
	"net/http"
	"strconv"
)

type app struct {
	searcher search.Searcher
	closer   interface{ Close() error }
}

type apiResponse struct {
	Query   string          `json:"query"`
	Results []search.Result `json:"results"`
	Error   string          `json:"error,omitempty"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "listen address")
	indexPath := flag.String("index", "", "mmap index file; empty uses built-in demo corpus")
	flag.Parse()

	searcher, closer, err := openSearcher(*indexPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	if closer != nil {
		defer closer.Close()
	}
	a := app{searcher: searcher, closer: closer}
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/api/search", a.handleSearch)
	fmt.Printf("lab5 search UI: http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Println(err)
	}
}

func openSearcher(path string) (search.Searcher, interface{ Close() error }, error) {
	if path == "" {
		return search.Build(search.SampleDocuments()), nil, nil
	}
	idx, err := search.OpenDiskIndex(path)
	if err != nil {
		return nil, nil, err
	}
	return idx, idx, nil
}

func (a app) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, nil)
}

func (a app) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := parseLimit(r.URL.Query().Get("limit"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if q == "" {
		json.NewEncoder(w).Encode(apiResponse{Error: "empty query"})
		return
	}
	results, err := a.searcher.Search(q, limit)
	if err != nil {
		json.NewEncoder(w).Encode(apiResponse{Query: q, Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(apiResponse{Query: q, Results: results})
}

func parseLimit(raw string) int {
	if raw == "" {
		return 10
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 10
	}
	return n
}

var page = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Lab5 Search</title>
<style>
body{margin:0;font:16px/1.45 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f7f9;color:#17202a}
main{max-width:980px;margin:0 auto;padding:32px 20px}
h1{font-size:28px;margin:0 0 18px}
form{display:grid;grid-template-columns:1fr auto;gap:10px;margin-bottom:18px}
input{font:inherit;padding:12px 14px;border:1px solid #b8c0cc;border-radius:6px;background:white}
button{font:inherit;padding:12px 18px;border:0;border-radius:6px;background:#1d4ed8;color:white;cursor:pointer}
.result{background:white;border:1px solid #d7dce3;border-radius:8px;padding:14px 16px;margin:10px 0}
.title{font-weight:700;margin-bottom:4px}
.meta{color:#5b6778;font-size:13px;margin-bottom:8px}
.snippet{white-space:pre-wrap}
.error{color:#b91c1c}
</style>
</head>
<body>
<main>
<h1>Lab5 Search</h1>
<form id="form">
<input id="q" autocomplete="off" value="history AND NOT (russia AND china)">
<button>Search</button>
</form>
<div id="out"></div>
</main>
<script>
const form=document.getElementById('form');
const q=document.getElementById('q');
const out=document.getElementById('out');
form.addEventListener('submit', async (event)=>{
  event.preventDefault();
  out.textContent='Searching...';
  const res=await fetch('/api/search?q='+encodeURIComponent(q.value)+'&limit=20');
  const data=await res.json();
  if(data.error){ out.innerHTML='<p class="error">'+escapeHtml(data.error)+'</p>'; return; }
  out.innerHTML=(data.results||[]).map(render).join('') || '<p>No results</p>';
});
function render(item){
  return '<article class="result"><div class="title">'+escapeHtml(item.title)+'</div>'+
    '<div class="meta">doc '+item.doc_id+' · score '+item.score.toFixed(4)+'</div>'+
    '<div class="snippet">'+escapeHtml(item.snippet)+'</div></article>';
}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
</script>
</body>
</html>`))
