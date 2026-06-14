package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"lab5/search"
	"net/http"
	"strconv"
	"time"
)

type app struct {
	searcher detailedSearcher
	closer   interface{ Close() error }
}

type detailedSearcher interface {
	search.Searcher
	SearchDetailed(string, int) (search.SearchStats, error)
}

type apiResponse struct {
	Query        string          `json:"query"`
	Results      []search.Result `json:"results"`
	TotalMatches int             `json:"total_matches"`
	Returned     int             `json:"returned"`
	Limit        int             `json:"limit"`
	ElapsedMS    float64         `json:"elapsed_ms"`
	Error        string          `json:"error,omitempty"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "listen address")
	indexPath := flag.String("index", "data/russian_classics.lab5", "mmap index file")
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

func openSearcher(path string) (detailedSearcher, interface{ Close() error }, error) {
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
	start := time.Now()
	stats, err := a.searcher.SearchDetailed(q, limit)
	if err != nil {
		json.NewEncoder(w).Encode(apiResponse{Query: q, Error: err.Error()})
		return
	}
	elapsed := float64(time.Since(start).Microseconds()) / 1000
	json.NewEncoder(w).Encode(apiResponse{
		Query:        q,
		Results:      stats.Results,
		TotalMatches: stats.TotalMatches,
		Returned:     len(stats.Results),
		Limit:        stats.Limit,
		ElapsedMS:    elapsed,
	})
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
main{max-width:1200px;margin:0 auto;padding:32px 20px}
h1{font-size:28px;margin:0 0 18px}
form{display:grid;grid-template-columns:1fr 110px auto;gap:10px;margin-bottom:10px}
input{font:inherit;padding:12px 14px;border:1px solid #b8c0cc;border-radius:6px;background:white}
button{font:inherit;padding:12px 18px;border:0;border-radius:6px;background:#1d4ed8;color:white;cursor:pointer}
.layout{display:grid;grid-template-columns:minmax(0,1fr) minmax(220px,280px);gap:16px;align-items:start}
.examples{background:white;border:1px solid #d7dce3;border-radius:8px;padding:14px}
.examples h2{font-size:16px;margin:0 0 10px}
.example{width:100%;display:block;text-align:left;background:#f8fafc;color:#17202a;border:1px solid #d7dce3;border-radius:6px;padding:10px 11px;margin:8px 0}
.example:hover{background:#eef4ff;border-color:#93b4f5}
.example b{display:block;font-size:13px;margin-bottom:2px}
.example span{display:block;color:#526070;font-size:12px;overflow-wrap:anywhere}
#stats{color:#465365;font-size:14px;margin:0 0 14px}
.result{background:white;border:1px solid #d7dce3;border-radius:8px;padding:14px 16px;margin:10px 0}
.title{font-weight:700;margin-bottom:4px}
.meta{color:#5b6778;font-size:13px;margin-bottom:8px}
.snippet{white-space:pre-wrap}
.error{color:#b91c1c}
@media (max-width:900px){
  form{grid-template-columns:1fr}
}
@media (max-width:640px){
  .layout{grid-template-columns:1fr}
}
</style>
</head>
<body>
<main>
<h1>Lab5 Search</h1>
<div class="layout">
<section>
  <form id="form">
  <input id="q" autocomplete="off" value="князь AND NOT (француз AND император)">
  <input id="limit" type="number" min="1" max="100" value="5" title="Top K">
  <button>Search</button>
  </form>
  <div id="stats"></div>
  <div id="out"></div>
</section>
<aside class="examples">
  <h2>Примеры запросов</h2>
  <button class="example" type="button" data-query="князь"><b>Частый терм</b><span>князь</span></button>
  <button class="example" type="button" data-query="ростовы"><b>Редкий терм</b><span>ростовы</span></button>
  <button class="example" type="button" data-query="князь AND андрей"><b>AND</b><span>князь AND андрей</span></button>
  <button class="example" type="button" data-query="пьер OR наташа"><b>OR</b><span>пьер OR наташа</span></button>
  <button class="example" type="button" data-query="князь AND NOT (француз AND император)"><b>NOT + скобки</b><span>князь AND NOT (француз AND император)</span></button>
  <button class="example" type="button" data-query="пьер ADJ безухов"><b>ADJ</b><span>пьер ADJ безухов</span></button>
  <button class="example" type="button" data-query="андрей NEAR/5 болконский"><b>NEAR/5</b><span>андрей NEAR/5 болконский</span></button>
  <button class="example" type="button" data-query="наташа NEAR/20 ростова"><b>NEAR/20</b><span>наташа NEAR/20 ростова</span></button>
  <button class="example" type="button" data-query="мертвые AND души"><b>Unicode</b><span>мертвые AND души</span></button>
  <button class="example" type="button" data-query="несуществующийтерм"><b>Нет выдачи</b><span>несуществующийтерм</span></button>
</aside>
</div>
</main>
<script>
const form=document.getElementById('form');
const q=document.getElementById('q');
const limit=document.getElementById('limit');
const out=document.getElementById('out');
const stats=document.getElementById('stats');
document.querySelectorAll('.example').forEach((button)=>{
  button.addEventListener('click',()=>{
    q.value=button.dataset.query;
    form.requestSubmit();
  });
});
form.addEventListener('submit', async (event)=>{
  event.preventDefault();
  out.textContent='Searching...';
  stats.textContent='';
  const res=await fetch('/api/search?q='+encodeURIComponent(q.value)+'&limit='+encodeURIComponent(limit.value || '5'));
  const data=await res.json();
  if(data.error){ out.innerHTML='<p class="error">'+escapeHtml(data.error)+'</p>'; return; }
  stats.textContent='elapsed '+data.elapsed_ms.toFixed(3)+' ms · L0 matches '+data.total_matches+' · returned '+data.returned+' / limit '+data.limit;
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
