package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
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

var rodoviaRegex = regexp.MustCompile(`^BR-\d{3}$`)

var tiposValidos = map[string]bool{
	"acidente": true, "buraco": true, "alagamento": true, "obstrucao": true,
}

func validar(o Ocorrencia) error {
	if !rodoviaRegex.MatchString(o.Rodovia) {
		return fmt.Errorf("rodovia inválida: %q (use o formato BR-040)", o.Rodovia)
	}
	if !tiposValidos[o.Tipo] {
		return fmt.Errorf("tipo inválido: %q", o.Tipo)
	}
	if o.Km < 0 || o.Km > 5000 {
		return fmt.Errorf("km fora do intervalo (0 a 5000): %d", o.Km)
	}
	if o.Descricao != nil && len(*o.Descricao) > 200 {
		return fmt.Errorf("descrição passa de 200 caracteres")
	}
	return nil
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

func total(w http.ResponseWriter, r *http.Request) {
	store.mu.Lock()
	defer store.mu.Unlock()
	escreverJSON(w, http.StatusOK, map[string]int{"total": len(store.ocorrencias)})
}

func criar(w http.ResponseWriter, r *http.Request) {
	var o Ocorrencia
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		erroJSON(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if err := validar(o); err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	o.ID = store.proximoID // o cliente não escolhe o id
	store.proximoID++
	store.ocorrencias = append(store.ocorrencias, o)
	escreverJSON(w, http.StatusCreated, o)
}

func deletar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, "id inválido")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for i, o := range store.ocorrencias {
		if o.ID == id {
			store.ocorrencias = append(store.ocorrencias[:i], store.ocorrencias[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
}

type AtualizacaoParcial struct {
	Rodovia   *string `json:"rodovia"`
	Tipo      *string `json:"tipo"`
	Km        *int    `json:"km"`
	Descricao *string `json:"descricao"`
}

func atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, "id inválido")
		return
	}

	var novo Ocorrencia
	if err := json.NewDecoder(r.Body).Decode(&novo); err != nil {
		erroJSON(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if err := validar(novo); err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for i := range store.ocorrencias {
		if store.ocorrencias[i].ID == id {
			novo.ID = id
			store.ocorrencias[i] = novo
			escreverJSON(w, http.StatusOK, novo)
			return
		}
	}
	erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
}

func atualizarParcial(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, "id inválido")
		return
	}

	var p AtualizacaoParcial
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		erroJSON(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for i := range store.ocorrencias {
		if store.ocorrencias[i].ID != id {
			continue
		}
		atual := store.ocorrencias[i] // cópia: só gravamos se for válida
		if p.Rodovia != nil {
			atual.Rodovia = *p.Rodovia
		}
		if p.Tipo != nil {
			atual.Tipo = *p.Tipo
		}
		if p.Km != nil {
			atual.Km = *p.Km
		}
		if p.Descricao != nil {
			atual.Descricao = p.Descricao
		}
		if err := validar(atual); err != nil {
			erroJSON(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		store.ocorrencias[i] = atual
		escreverJSON(w, http.StatusOK, atual)
		return
	}
	erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ocorrencias", listar)
	mux.HandleFunc("GET /ocorrencias/total", total)
	mux.HandleFunc("GET /ocorrencias/{id}", buscar)
	mux.HandleFunc("POST /ocorrencias", criar)
	mux.HandleFunc("DELETE /ocorrencias/{id}", deletar)
	mux.HandleFunc("PUT /ocorrencias/{id}", atualizar)
    mux.HandleFunc("PATCH /ocorrencias/{id}", atualizarParcial)

	log.Println("Servidor em http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}