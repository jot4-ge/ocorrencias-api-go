package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Ocorrencia struct {
	ID        int     `json:"id"`
	Rodovia   string  `json:"rodovia"`
	Tipo      string  `json:"tipo"`
	Km        int     `json:"km"`
	Descricao *string `json:"descricao"`
}

// "Banco" em memória, protegido por mutex (requisições rodam em paralelo)
type Store struct {
	mu          sync.Mutex
	ocorrencias []Ocorrencia
	proximoID   int
}

var store = &Store{
	ocorrencias: []Ocorrencia{
		{ID: 1, Rodovia: "BR-040", Tipo: "acidente", Km: 12},
		{ID: 2, Rodovia: "BR-060", Tipo: "buraco", Km: 87},
		{ID: 3, Rodovia: "BR-040", Tipo: "alagamento", Km: 45},
	},
	proximoID: 4,
}

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func erroJSON(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, map[string]string{"detail": msg})
}

func listar(w http.ResponseWriter, r *http.Request) {
	rodovia := r.URL.Query().Get("rodovia")
	tipo := r.URL.Query().Get("tipo")

	store.mu.Lock()
	defer store.mu.Unlock()

	resultado := []Ocorrencia{}
	for _, o := range store.ocorrencias {
		if rodovia != "" && o.Rodovia != rodovia {
			continue
		}
		if tipo != "" && o.Tipo != tipo {
			continue
		}
		resultado = append(resultado, o)
	}
	escreverJSON(w, http.StatusOK, resultado)
}

func buscar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, "id inválido")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for _, o := range store.ocorrencias {
		if o.ID == id {
			escreverJSON(w, http.StatusOK, o)
			return
		}
	}
	erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ocorrencias", listar)
	mux.HandleFunc("GET /ocorrencias/{id}", buscar)

	log.Println("Servidor em http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}