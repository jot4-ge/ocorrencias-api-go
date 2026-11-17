package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

type Ocorrencia struct {
	ID        int     `json:"id"`
	Rodovia   string  `json:"rodovia"`
	Tipo      string  `json:"tipo"`
	Km        int     `json:"km"`
	Descricao *string `json:"descricao"`
}

type AtualizacaoParcial struct {
	Rodovia   *string `json:"rodovia"`
	Tipo      *string `json:"tipo"`
	Km        *int    `json:"km"`
	Descricao *string `json:"descricao"`
}

var repo *Repo

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

func erroInterno(w http.ResponseWriter, contexto string, err error) {
	log.Println(contexto+":", err)
	erroJSON(w, http.StatusInternalServerError, "erro interno")
}

func idDaURL(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, "id inválido")
		return 0, false
	}
	return id, true
}

func listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resultado, err := repo.Listar(r.Context(), q.Get("rodovia"), q.Get("tipo"))
	if err != nil {
		erroInterno(w, "listar", err)
		return
	}
	escreverJSON(w, http.StatusOK, resultado)
}

func total(w http.ResponseWriter, r *http.Request) {
	n, err := repo.Total(r.Context())
	if err != nil {
		erroInterno(w, "total", err)
		return
	}
	escreverJSON(w, http.StatusOK, map[string]int{"total": n})
}

func buscar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaURL(w, r)
	if !ok {
		return
	}
	o, err := repo.Buscar(r.Context(), id)
	if errors.Is(err, ErrNaoEncontrada) {
		erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
		return
	}
	if err != nil {
		erroInterno(w, "buscar", err)
		return
	}
	escreverJSON(w, http.StatusOK, o)
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
	criada, err := repo.Criar(r.Context(), o)
	if err != nil {
		erroInterno(w, "criar", err)
		return
	}
	escreverJSON(w, http.StatusCreated, criada)
}

func atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaURL(w, r)
	if !ok {
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
	atualizada, err := repo.Atualizar(r.Context(), id, novo)
	if errors.Is(err, ErrNaoEncontrada) {
		erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
		return
	}
	if err != nil {
		erroInterno(w, "atualizar", err)
		return
	}
	escreverJSON(w, http.StatusOK, atualizada)
}

func atualizarParcial(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaURL(w, r)
	if !ok {
		return
	}
	var p AtualizacaoParcial
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		erroJSON(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	atual, err := repo.Buscar(r.Context(), id)
	if errors.Is(err, ErrNaoEncontrada) {
		erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
		return
	}
	if err != nil {
		erroInterno(w, "atualizarParcial/buscar", err)
		return
	}

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

	atualizada, err := repo.Atualizar(r.Context(), id, atual)
	if err != nil {
		erroInterno(w, "atualizarParcial/atualizar", err)
		return
	}
	escreverJSON(w, http.StatusOK, atualizada)
}

func deletar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaURL(w, r)
	if !ok {
		return
	}
	err := repo.Deletar(r.Context(), id)
	if errors.Is(err, ErrNaoEncontrada) {
		erroJSON(w, http.StatusNotFound, "Ocorrência não encontrada")
		return
	}
	if err != nil {
		erroInterno(w, "deletar", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	ctx := context.Background()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5433/ocorrencias_go"
	}

	var err error
	repo, err = NovoRepo(ctx, url)
	if err != nil {
		log.Fatal("conexão com o banco: ", err)
	}
	if err := repo.CriarTabela(ctx); err != nil {
		log.Fatal("criar tabela: ", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ocorrencias", listar)
	mux.HandleFunc("GET /ocorrencias/total", total)
	mux.HandleFunc("GET /ocorrencias/{id}", buscar)
	mux.HandleFunc("POST /ocorrencias", criar)
	mux.HandleFunc("PUT /ocorrencias/{id}", atualizar)
	mux.HandleFunc("PATCH /ocorrencias/{id}", atualizarParcial)
	mux.HandleFunc("DELETE /ocorrencias/{id}", deletar)

	log.Println("Servidor em http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}